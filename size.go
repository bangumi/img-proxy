package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
	"strings"
)

type Size struct {
	Height uint64
	Width  uint64
}

var invalidSizeErr = echo.NewHTTPError(http.StatusBadRequest, "invalid size format, check readme for more details or file an issue")

// ParseSize return a error when size in invalid format
func ParseSize(s string) (Size, error) {
	userSize := strings.ToLower(s)

	var size Size
	var err error
	if strings.Contains(userSize, "x") {
		s := strings.SplitN(userSize, "x", 2)
		if len(s) != 2 {
			return Size{}, invalidSizeErr
		}

		size.Width, err = strconv.ParseUint(s[0], 10, 64)
		if err != nil {
			return Size{}, invalidSizeErr
		}

		size.Height, err = strconv.ParseUint(s[1], 10, 64)
		if err != nil {
			return Size{}, invalidSizeErr
		}
	} else {
		size.Width, err = strconv.ParseUint(userSize, 10, 64)
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
