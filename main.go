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
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/samber/lo"
	"github.com/spf13/pflag"
)

//go:embed readme.md
var readmeMD string

var s3entryPoint string
var s3accessKey string
var s3secretKey string
var s3bucket string
var rawUpstream string
var cacheSize int

func init() {
	pflag.StringVar(&s3entryPoint, "s3.entrypoint", "", "s3 url")
	pflag.StringVar(&s3accessKey, "s3.access-key", "", "s3 access key")
	pflag.StringVar(&s3secretKey, "s3.secret-key", "", "s3 secret key")
	pflag.StringVar(&s3bucket, "s3.bucket", "img-resize", "s3 bucket name")
	pflag.StringVar(&rawUpstream, "upstream", "", "upstream imaginary url")
	pflag.IntVar(&cacheSize, "cache-size", 100000, "lru cache size")
}

var logLevel = os.Getenv("LOG_LEVEL")

var logger = zerolog.New(os.Stdout).
	Level(lo.Must(zerolog.ParseLevel(lo.If(logLevel != "", logLevel).Else("info")))).
	With().Timestamp().
	Logger()

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
	e.HideBanner = true

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

	h := NewHandler()

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

		image, err := h.processImage(c, upstream, p, size, hd)
		if err != nil {
			return err
		}

		//c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=60, immutable")
		c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=5, immutable")
		return c.Blob(http.StatusOK, image.contentType, image.body)
	}, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			if err := next(c); err != nil {
				return err
			}

			if c.Response().Status != 200 {
				return nil
			}

			duration := time.Since(start).Seconds()
			h.requestCounter.Inc()
			if cached, ok := c.Get("cached").(bool); ok {
				if cached {
					h.cachedRequestHist.Observe(duration)
					h.cachedCounter.Inc()
				} else {
					h.uncachedRequestHist.Observe(duration)
				}
			} else {
				logger.Info().Str("path", c.Request().URL.Path).Msg("request without ctx")
			}

			return nil
		}
	})

	{
		prometheus.MustRegister(
			h.requestCounter,
			h.cachedCounter,
			h.cachedRequestHist,
			h.uncachedRequestHist,
			h.cache,
		)

		e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	}

	host := os.Getenv("HTTP_HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8003"
	}

	{
		ok, err := h.cache.s3.BucketExists(context.Background(), s3bucket)
		if err != nil {
			panic(err)
		}

		if !ok {
			err := h.cache.s3.MakeBucket(context.Background(), s3bucket, minio.MakeBucketOptions{})
			if err != nil {
				panic(err)
			}
		}
	}

	logger.Info().Msg("start server")
	e.Logger.Fatal(e.Start(host + ":" + port))
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
