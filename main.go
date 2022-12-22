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
	_ "embed"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	roundrobin "github.com/hlts2/round-robin"
	"github.com/labstack/echo/v4"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
)
import "github.com/natefinch/atomic"
import "github.com/spf13/pflag"

//go:embed readme.md
var readme string

var cacheDir string
var upstream []string

func init() {
	pflag.StringSliceVar(&upstream, "upstream", nil, "upstream imaginary url")
	pflag.StringVar(&cacheDir, "cache", "tmp/.cache", "local cache dir")
	pflag.Parse()
}

func main() {
	var upstreams []*url.URL
	for _, s := range upstream {
		u, err := url.Parse(s)
		if err != nil {
			panic("failed to parse upstream url: " + err.Error())
		}

		u.Path = ""

		upstreams = append(upstreams, u)
	}
	rr, err := roundrobin.New(upstreams...)

	if err != nil {
		panic(err)
	}

	e := echo.New()

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		var ee *echo.HTTPError
		if errors.As(err, &ee) {
			_ = c.JSON(ee.Code, err.Error())
		} else {
			_ = c.String(http.StatusInternalServerError, err.Error())
		}
	}

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, readme)
	})

	e.GET("/r/", func(c echo.Context) error {
		return c.String(http.StatusOK, readme)
	})

	e.GET("/r/:size/*", func(c echo.Context) error {
		p := c.Param("*")
		if p == "" {
			return c.String(http.StatusNotFound, "")
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

		host := rr.Next()
		b, mimeType, err := fetchImage(host, p, size)
		if err != nil {
			return err
		}

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

func fetchImage(upstream *url.URL, p string, size Size) ([]byte, string, error) {
	cachedPath := localCacheFilePath(p, size)

	f, err := os.ReadFile(cachedPath)
	if err == nil {
		return f, mime.TypeByExtension(filepath.Ext(cachedPath)), nil
	}

	if !os.IsNotExist(err) {
		return nil, "", err
	}

	action := "crop"
	if size.Height == 0 || size.Width == 0 {
		action = "resize"
	}

	upstreamUrl := upstream.String() + "/" + action + "?" + url.Values{
		"height": {strconv.FormatUint(size.Height, 10)},
		"width":  {strconv.FormatUint(size.Width, 10)},
		"url":    {"http://lain.bgm.tv/" + p},
	}.Encode()

	resp, err := client.R().Get(upstreamUrl)
	if err != nil {
		return nil, "", err
	}

	content := resp.Body()

	if resp.StatusCode() > 300 {
		return content, resp.Header().Get(echo.HeaderContentType), nil
	}

	err = os.MkdirAll(filepath.Dir(cachedPath), 0644)
	if err != nil {
		return nil, "", err
	}

	err = atomic.WriteFile(cachedPath, bytes.NewReader(content))
	if err != nil {
		return nil, "", err
	}

	return content, resp.Header().Get(echo.HeaderContentType), err
}

func localCacheFilePath(p string, size Size) string {
	fs := hashFilename(p, size)

	return filepath.Join(cacheDir, fs)
}

func hashFilename(p string, size Size) string {
	ext := path.Ext(p)
	return fmt.Sprintf("%s%s@%dx%d%s", path.Dir(p), path.Base(p), size.Width, size.Height, ext)
}
