package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

type Size struct {
	Height uint64
	Width  uint64
}

var invalidSizeErr = echo.NewHTTPError(http.StatusBadRequest, "invalid size format, read document for more details or file an issue")

// ParseSize return a error when size in invalid format
func ParseSize(s string) (Size, error) {
	userSize := strings.ToLower(s)

	var size Size
	var err error

	width, height, found := strings.Cut(userSize, "x")
	size.Width, err = strconv.ParseUint(width, 10, 64)
	if err != nil {
		return Size{}, invalidSizeErr
	}

	if found {
		size.Height, err = strconv.ParseUint(height, 10, 64)
		if err != nil {
			return Size{}, invalidSizeErr
		}
	}

	return size, nil
}

func validWidth(width uint64) bool {
	if width == 100 || width == 200 || width == 400 || width == 600 || width == 800 || width == 1200 {
		return true
	}
	return false
}

func validHeight(height uint64) bool {
	if height == 100 || height == 200 || height == 400 || height == 600 || height == 800 || height == 1200 {
		return true
	}
	return false
}

func validSize(size Size) bool {
	if size.Height == 0 {
		return validWidth(size.Width)
	}

	if size.Width == 0 {
		return validHeight(size.Height)
	}

	return validWidth(size.Width) && validHeight(size.Height)
}
