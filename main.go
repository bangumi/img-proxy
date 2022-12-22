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
	_ "embed"
	"errors"
	"net/http"
	"net/url"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/labstack/echo/v4"
	"github.com/spf13/pflag"
)

//go:embed readme.md
var readmeMD string

var cacheSize int64
var rawUpstream string

func init() {
	pflag.StringVar(&rawUpstream, "upstream", "", "upstream imaginary url")
	pflag.Int64Var(&cacheSize, "cache.max-size", 1<<30, "maximum bytes of memory cache")
	pflag.CommandLine.ParseErrorsWhitelist.UnknownFlags = true
	pflag.Parse()
}

func main() {
	upstream, err := url.Parse(rawUpstream)
	if err != nil {
		panic(err)
	}

	e := echo.New()

	e.Renderer = newRender()

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		var ee *echo.HTTPError
		if errors.As(err, &ee) {
			_ = c.String(ee.Code, err.Error())
		} else {
			_ = c.String(http.StatusInternalServerError, err.Error())
		}
	}

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "readme.gohtml", readmeMD)
	})

	e.GET("/r/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "readme.gohtml", readmeMD)
	})

	h := Handle{
		cache: NewCache(),
	}

	e.GET("/r/:size/*", func(c echo.Context) error {
		p := c.Param("*")
		if p == "" {
			return c.String(http.StatusNotFound, "")
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

		b, mimeType, err := h.fetchImage(upstream, p, size)
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=31536000, immutable")
		return c.Blob(http.StatusOK, mimeType, b)
	})

	e.GET("/metrics", h.Metrics)

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
