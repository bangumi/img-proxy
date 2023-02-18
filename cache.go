package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
)

type Image struct {
	body []byte

	contentType string
}

func NewCache() *Cache {
	s3Client := s3()

	cache, err := lru.NewWithEvict[string, bool](cacheSize, func(key string, value bool) {
		err := s3Client.RemoveObject(context.Background(), s3bucket, key, minio.RemoveObjectOptions{})
		if err != nil {
			logger.Err(err).Str("key", key).Msg("failed to remove object")
		}
	})

	if err != nil {
		panic(err)
	}

	return &Cache{
		s3:     s3Client,
		lru:    cache,
		bucket: s3bucket,
		cacheSizeCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "chii_img_lru_size",
		}),
	}
}

type Cache struct {
	lru *lru.Cache[string, bool]
	s3  *minio.Client

	bucket string

	cacheSizeCount prometheus.Gauge
}

func (c *Cache) Describe(descs chan<- *prometheus.Desc) {
}

func (c *Cache) Collect(metrics chan<- prometheus.Metric) {
	c.cacheSizeCount.Set(float64(c.lru.Len()))
	metrics <- c.cacheSizeCount
}

func (c *Cache) Get(ctx context.Context, key string) (item Image, exist bool, err error) {
	_, cached := c.lru.Get(key)
	if !cached {
		return Image{}, false, nil
	}

	stat, err := c.s3.StatObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err == nil {
		obj, err := c.s3.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
		if err != nil {
			return Image{}, false, fmt.Errorf("failed to get raw image from s3: %w", err)
		}
		defer obj.Close()

		raw, err := io.ReadAll(obj)
		return Image{body: raw, contentType: stat.Metadata.Get("Content-Type")}, true, nil
	}

	// stupid golang error handling
	var e minio.ErrorResponse
	if errors.As(err, &e) {
		if e.Code != "NoSuchKey" {
			return Image{}, false, nil
		}
	} else {
		return Image{}, false, err
	}

	return Image{}, false, nil
}

func (c *Cache) Set(ctx context.Context, key string, value Image) error {
	_, err := c.s3.PutObject(ctx, c.bucket, key, bytes.NewBuffer(value.body), int64(len(value.body)), minio.PutObjectOptions{
		ContentType: value.contentType,
	})
	if err != nil {
		return err
	}

	c.lru.Add(key, true)

	return nil
}
