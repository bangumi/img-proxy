package main

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type Handle struct {
	cache *Cache[string, Item]
}

func (h Handle) fetchImage(upstream *url.URL, p string, size Size) ([]byte, string, error) {
	cachedPath := localCacheFilePath(p, size)

	value, found := h.cache.Get(cachedPath)
	if found {
		return value.content, value.contentType, nil
	}

	action := "smartcrop"
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

	contentType := resp.Header().Get(echo.HeaderContentType)

	if resp.StatusCode() > 300 {
		return content, contentType, nil
	}

	h.cache.SetWithTTL(cachedPath, Item{content: content, contentType: contentType}, int64(len(content)), time.Hour)

	return content, contentType, nil
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

	return nil
}
