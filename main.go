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
	"encoding/hex"
	"github.com/go-resty/resty/v2"
	roundrobin "github.com/hlts2/round-robin"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/blake2b"
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

func validHeight(height int) bool {
	if height == 100 || height == 200 || height == 400 || height == 600 || height == 800 || height == 1200 {
		return true
	}

	return false
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
		_ = c.String(http.StatusInternalServerError, err.Error())
	}

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, readme)
	})

	e.GET("/r/:height/*", func(c echo.Context) error {
		path := c.Param("*")
		if path == "" {
			return c.String(http.StatusNotFound, "")
		}

		height, err := strconv.Atoi(c.Param("height"))
		if err != nil {
			return c.String(http.StatusBadRequest, "height is not valid int")
		}

		if !validHeight(height) {
			return c.String(http.StatusBadRequest, "not valid height, only 100/200/400/600/800/1200 are allowed")
		}

		host := rr.Next()

		u := &url.URL{
			Scheme: "http",
			Host:   "lain.bgm.tv",
			Path:   "/" + path,
		}

		bytes, mimeType, err := fetchImage(host, u, height)
		if err != nil {
			return err
		}

		return c.Blob(http.StatusOK, mimeType, bytes)
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

func fetchImage(upstream, u *url.URL, height int) ([]byte, string, error) {
	cachedPath, err := localCacheFilePath(u, height)

	f, err := os.ReadFile(cachedPath)
	if err == nil {
		return f, mime.TypeByExtension(filepath.Ext(cachedPath)), nil
	}

	if !os.IsNotExist(err) {
		return nil, "", err
	}

	upstreamUrl := upstream.String() + "/resize?" + url.Values{
		"height": {strconv.Itoa(height)},
		"url":    {u.String()},
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

func localCacheFilePath(u *url.URL, height int) (string, error) {
	fs, err := hashFilename(u, height)
	if err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, string(fs[0]), string(fs[1]), fs), nil
}

func hashFilename(u *url.URL, height int) (string, error) {
	h, err := blake2b.New256(nil)
	if err != nil {
		return "", err
	}

	h.Write([]byte(u.String()))

	ext := path.Ext(u.Path)

	return hex.EncodeToString(h.Sum(nil)) + "@" + strconv.Itoa(height) + ext, nil
}
