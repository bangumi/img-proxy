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
var s3keyID string
var s3accessKey string
var s3secretKey string
var s3bucket string
var rawUpstream string

func init() {
	pflag.StringVar(&s3entryPoint, "s3.entrypoint", "", "s3 url")
	pflag.StringVar(&s3keyID, "s3.key-id", "", "s3 token")
	pflag.StringVar(&s3accessKey, "s3.access-key", "", "s3 access key")
	pflag.StringVar(&s3secretKey, "s3.secret-key", "", "s3 secret key")
	pflag.StringVar(&s3bucket, "s3.bucket", "img-resize", "s3 bucket name")
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
		panic(err)
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

		b, mimeType, err := h.fetchImage(c.Request().Context(), upstream, p, size)
		if err != nil {
			return err
		}
		defer b.Close()

		c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=31536000, immutable")
		return c.Stream(http.StatusOK, mimeType, b)
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

func (h Handle) fetchImage(ctx context.Context, upstream *url.URL, p string, size Size) (io.ReadCloser, string, error) {
	cachedPath := localCacheFilePath(p, size)

	stat, err := h.s3.StatObject(ctx, s3bucket, cachedPath, minio.GetObjectOptions{})
	if err == nil {
		obj, err := h.s3.GetObject(ctx, s3bucket, cachedPath, minio.GetObjectOptions{})
		return obj, stat.ContentType, err
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

	sourceURL := "http://lain.bgm.tv/" + p

	img, err := client.R().Get(sourceURL)
	if err != nil {
		return nil, "", err
	}
	if img.StatusCode() == 404 {
		return nil, "", echo.NewHTTPError(http.StatusNotFound, "image not found")
	}

	if img.StatusCode() >= 300 {
		return nil, "", echo.NewHTTPError(http.StatusBadGateway, img.String())
	}

	action := "smartcrop"
	if size.Height == 0 || size.Width == 0 {
		action = "resize"
	}

	upstreamUrl := upstream.String() + "/" + action + "?" + url.Values{
		"height": {strconv.FormatUint(size.Height, 10)},
		"width":  {strconv.FormatUint(size.Width, 10)},
		"field":  {"file"},
	}.Encode()

	resp, err := client.R().SetMultipartField(
		"file", filepath.Base(p), img.Header().Get(echo.HeaderContentType), bytes.NewBuffer(img.Body()),
	).Post(upstreamUrl)
	if err != nil {
		return nil, "", err
	}

	content := resp.Body()

	contentType := resp.Header().Get(echo.HeaderContentType)

	if resp.StatusCode() > 300 {
		return nil, "", echo.NewHTTPError(http.StatusBadGateway, resp.String())
	}

	_, err = h.s3.PutObject(ctx,
		s3bucket,
		cachedPath,
		bytes.NewReader(resp.Body()),
		int64(len(resp.Body())),
		minio.PutObjectOptions{
			ContentType: contentType,
		})
	if err != nil {
		return nil, "", fmt.Errorf("failed to put s3 object %w", err)
	}

	return io.NopCloser(bytes.NewReader(content)), contentType, nil
}

func localCacheFilePath(p string, size Size) string {
	fs := hashFilename(p, size)

	return fs
}

func hashFilename(p string, size Size) string {
	return fmt.Sprintf("%s%s@%dx%d", path.Dir(p), path.Base(p), size.Width, size.Height)
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

	return nil
}
