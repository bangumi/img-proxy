// SPDX-License-Identifier: AGPL-3.0-only
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, version 3.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
// See the GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>

package main

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/samber/lo"
	"github.com/spf13/pflag"
)

//go:embed readme.md
var readmeMD string

var s3entryPoint string
var s3accessKey string
var s3secretKey string
var s3bucket string
var s3RawBucket string
var rawUpstream string

func init() {
	pflag.StringVar(&s3entryPoint, "s3.entrypoint", "", "s3 url")
	pflag.StringVar(&s3accessKey, "s3.access-key", "", "s3 access key")
	pflag.StringVar(&s3secretKey, "s3.secret-key", "", "s3 secret key")
	pflag.StringVar(&s3bucket, "s3.bucket", "img-resize", "s3 bucket name")
	pflag.StringVar(&s3RawBucket, "s3.raw-bucket", "", "s3 bucket to store raw files")
	pflag.StringVar(&rawUpstream, "upstream", "", "upstream imaginary url")
}

func main() {
	if lo.Contains(os.Args[1:], "--help") || lo.Contains(os.Args[1:], "-h") {
		pflag.PrintDefaults()
		os.Exit(0)
	}

	pflag.Parse()

	upstream, err := url.Parse(rawUpstream)
	if err != nil {
		panic("failed to parse upstream url: " + err.Error())
	}

	e := echo.New()

	e.Renderer = newRender()

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		var ee *echo.HTTPError
		if errors.As(err, &ee) {
			if msg, ok := ee.Message.(string); ok {
				_ = c.String(ee.Code, msg)
			} else {
				_ = c.JSON(ee.Code, ee.Message)
			}
		} else {
			_ = c.String(http.StatusInternalServerError, err.Error())
		}
	}

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("x-version", version)
			return next(c)
		}
	})

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "readme.gohtml", map[string]string{"readme": readmeMD, "version": version})
	})

	e.GET("/r/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "readme.gohtml", map[string]string{"readme": readmeMD, "version": version})
	})

	h := Handle{
		s3: s3(),
	}

	e.GET("/r/:size/*", func(c echo.Context) error {
		p := c.Param("*")
		if p == "" {
			return c.String(http.StatusNotFound, "")
		}

		if len(p) >= 100 {
			return c.String(http.StatusBadRequest, "too lang url")
		}

		err = blockedPath(p)
		if err != nil {
			return err
		}

		userSize := c.Param("size")
		if userSize == "" {
			return c.String(http.StatusNotFound, "")

		}

		size, err := ParseSize(userSize)
		if err != nil {
			return err
		}

		if !validSize(size) {
			return invalidSizeErr
		}

		var hd bool
		if c.QueryParams().Has("hd") {
			hd, err = strconv.ParseBool(c.QueryParams().Get("hd"))
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid query 'hd' %q, should present a bool", c.QueryParams().Get("hd")))
			}
		}

		b, mimeType, err := h.fetchImage(c.Request().Context(), upstream, p, size, hd)
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=31536000, immutable")
		return c.Blob(http.StatusOK, mimeType, b)
	})

	host := os.Getenv("HTTP_HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8003"
	}

	e.Logger.Fatal(e.Start(host + ":" + port))
}

var client = resty.New()

type Handle struct {
	s3 *minio.Client
}

func (h Handle) fetchRawImage(ctx context.Context, p string, hd bool) ([]byte, string, error) {
	s3Path := strings.TrimPrefix(p, "pic/")
	if hd {
		s3Path = "hd/" + s3Path
	}

	getter := func() ([]byte, string, error) {
		// 生产环境走的是内网，不能用 https
		sourceURL := "http://lain.bgm.tv/" + p
		if hd {
			sourceURL += "?hd=1"
		}

		img, err := client.R().SetContext(ctx).Get(sourceURL)
		if err != nil {
			return nil, "", err
		}
		if img.StatusCode() == 404 {
			return nil, "", echo.NewHTTPError(http.StatusNotFound, "image not found")
		}

		if img.StatusCode() >= 300 {
			return nil, "", echo.NewHTTPError(http.StatusInternalServerError, img.String())
		}

		return img.Body(), img.Header().Get(echo.HeaderContentType), nil
	}

	if s3RawBucket == "" {
		return getter()
	}

	return h.withS3Cached(ctx, s3RawBucket, s3Path, getter)
}

func (h Handle) fetchImage(ctx context.Context, upstream *url.URL, p string, size Size, hd bool) ([]byte, string, error) {
	cachedPath := localCacheFilePath(p, size, hd)

	return h.withS3Cached(ctx, s3bucket, cachedPath, func() ([]byte, string, error) {
		img, ct, err := h.fetchRawImage(ctx, p, hd)
		if err != nil {
			return nil, "", err
		}

		action := "smartcrop"
		if size.Height == 0 || size.Width == 0 {
			action = "resize"
		}

		qs := url.Values{
			"height": {strconv.FormatUint(size.Height, 10)},
			"width":  {strconv.FormatUint(size.Width, 10)},
			"field":  {"file"},
		}

		if path.Ext(path.Base(p)) == ".jpg" {
			qs.Set("type", "jpeg")
		}

		upstreamUrl := upstream.String() + "/" + action + "?" + qs.Encode()

		resp, err := client.R().SetContext(ctx).
			SetMultipartField("file", filepath.Base(p), ct, bytes.NewBuffer(img)).
			Post(upstreamUrl)
		if err != nil {
			return nil, "", err
		}

		content := resp.Body()

		contentType := resp.Header().Get(echo.HeaderContentType)

		if resp.StatusCode() > 300 {
			return nil, "", echo.NewHTTPError(http.StatusInternalServerError, "failed to process image: "+resp.String())
		}

		return content, contentType, nil
	})
}

func (h Handle) withS3Cached(ctx context.Context, bucket, filepath string, getter func() ([]byte, string, error)) ([]byte, string, error) {
	stat, err := h.s3.StatObject(ctx, bucket, filepath, minio.GetObjectOptions{})
	if err == nil {
		obj, err := h.s3.GetObject(ctx, bucket, filepath, minio.GetObjectOptions{})
		if err != nil {
			return nil, "", fmt.Errorf("failed to get raw image from s3: %w", err)
		}
		defer obj.Close()

		raw, err := io.ReadAll(obj)
		return raw, stat.ContentType, err
	}

	// stupid golang error handling
	var e minio.ErrorResponse
	if errors.As(err, &e) {
		if e.Code != "NoSuchKey" {
			return nil, "", err
		}
	} else {
		return nil, "", err
	}

	img, contentType, err := getter()
	if err != nil {
		return nil, "", err
	}

	_, err = h.s3.PutObject(ctx, bucket, filepath, bytes.NewReader(img), int64(len(img)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return nil, "", fmt.Errorf("failed to save raw image to s3 %w", err)
	}

	return img, contentType, nil
}

func localCacheFilePath(p string, size Size, hd bool) string {
	fs := hashFilename(p, size)

	if hd {
		return "/hd" + fs
	}

	return fs
}

func hashFilename(p string, size Size) string {
	return fmt.Sprintf("/%s/%s@%dx%d", path.Dir(p), path.Base(p), size.Width, size.Height)
}

func blockedPath(p string) error {
	if strings.HasPrefix(p, "pic/cover/") {
		if !strings.HasPrefix(p, "pic/cover/l/") {
			return echo.NewHTTPError(http.StatusBadRequest, "please use '/r/<size>/pic/cover/l/' path instead")
		}
	}

	if strings.HasPrefix(p, "pic/photo/") {
		if !strings.HasPrefix(p, "pic/photo/l/") {
			return echo.NewHTTPError(http.StatusBadRequest, "please use '/r/<size>/pic/photo/l/' path instead")
		}
	}

	if strings.HasPrefix(p, "pic/crt/") {
		if !strings.HasPrefix(p, "pic/crt/l/") {
			return echo.NewHTTPError(http.StatusBadRequest, "please use '/r/<size>/pic/crt/l/' path instead")
		}
	}

	if strings.HasPrefix(p, "pic/user/") {
		if !strings.HasPrefix(p, "pic/user/l/") {
			return echo.NewHTTPError(http.StatusBadRequest, "please use '/r/<size>/pic/user/l/' path instead")
		}
	}

	return nil
}
