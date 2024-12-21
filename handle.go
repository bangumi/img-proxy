package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/go-resty/resty/v2"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

func NewHandler() Handle {
	h := Handle{
		cache:  NewCache(),
		client: resty.New(),
		cachedCounter: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "chii_img_cached_request_count",
				Help: "Count of cached image request",
			},
		),

		requestCounter: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "chii_img_all_request_count",
				Help: "Count of all image request",
			},
		),

		cachedRequestHist: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "chii_img_cached_request_duration_seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .2, .3, .4, .5, 0.75, 1, 2},
		}),

		uncachedRequestHist: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "chii_img_uncached_request_duration_seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .2, .3, .4, .5, 0.75, 1, 2, 3, 4, 5, 7.5, 10},
		}),
	}

	func() {
		_, err := h.cache.s3.HeadBucket(context.Background(), &s3.HeadBucketInput{Bucket: &s3bucket})
		if err == nil {
			return
		}

		var e smithy.APIError
		if !errors.As(err, &e) {
			panic(err)
		}

		if e.ErrorCode() != "NoSuchBucket" && e.ErrorCode() != "NotFound" && e.ErrorCode() != "NoSuchBucketException" {
			panic(err)
		}

		_, err = h.cache.s3.CreateBucket(context.Background(), &s3.CreateBucketInput{Bucket: &s3bucket})
		if err != nil {
			panic(err)
		}
	}()

	return h
}

type Handle struct {
	cache *Cache

	client *resty.Client

	cachedCounter       prometheus.Counter
	requestCounter      prometheus.Counter
	cachedRequestHist   prometheus.Histogram
	uncachedRequestHist prometheus.Histogram
}

func (h Handle) fetchRawImage(ctx context.Context, p string, hd bool) ([]byte, string, error) {
	s3Path := strings.TrimPrefix(p, "pic/")
	if hd {
		s3Path = "hd/" + s3Path
	}

	// 生产环境走的是内网，不能用 https
	sourceURL := "http://lain.bgm.tv/" + p
	if hd {
		sourceURL += "?hd=1"
	}

	img, err := h.client.R().SetContext(ctx).Get(sourceURL)
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

func (h Handle) processImage(c echo.Context, upstream *url.URL, p string, size Size, hd bool) (Image, error) {
	cachedPath := localCacheFilePath(p, size, hd)

	ctx := c.Request().Context()

	return h.withS3Cached(c, cachedPath, func() (Image, error) {
		img, ct, err := h.fetchRawImage(ctx, p, hd)
		if err != nil {
			return Image{}, err
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

		switch path.Ext(path.Base(p)) {
		case ".jpg":
			qs.Set("type", "jpeg")
		case ".webp":
			qs.Set("type", "webp")
		}

		upstreamUrl := upstream.String() + "/" + action + "?" + qs.Encode()

		resp, err := h.client.R().SetContext(ctx).
			SetMultipartField("file", filepath.Base(p), ct, bytes.NewBuffer(img)).
			Post(upstreamUrl)
		if err != nil {
			return Image{}, err
		}

		content := resp.Body()

		contentType := resp.Header().Get(echo.HeaderContentType)

		if resp.StatusCode() > 300 {
			return Image{}, echo.NewHTTPError(http.StatusInternalServerError, "failed to process image: "+resp.String())
		}

		return Image{body: content, contentType: contentType}, nil
	})
}

func (h Handle) withS3Cached(c echo.Context, filepath string, getter func() (Image, error)) (Image, error) {
	ctx := c.Request().Context()
	item, cached, err := h.cache.Get(ctx, filepath)
	if err != nil {
		return Image{}, err
	}

	c.Set("cached", cached)

	if cached {
		return item, nil
	}

	image, err := getter()
	if err != nil {
		return Image{}, err
	}

	if err := h.cache.Set(ctx, filepath, image); err != nil {
		log.Err(err).Msg("failed to set cache")
	}

	return image, nil
}
