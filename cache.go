package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/dgraph-io/ristretto"
	"github.com/minio/minio-go/v7"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/samber/lo"
)

type ristrettoItem struct {
	key string
}

type Image struct {
	body []byte

	contentType string
}

func NewCache() *Cache {
	s3 := newS3Client()

	var cache *ristretto.Cache
	if cacheSize != 0 {
		cache = lo.Must(ristretto.NewCache(&ristretto.Config{
			NumCounters:        int64(cacheSize * 10), // number of keys to track frequency of (10M).
			MaxCost:            int64(cacheSize),
			BufferItems:        64, // number of keys per Get buffer.
			Metrics:            true,
			IgnoreInternalCost: true,
			OnEvict: func(item *ristretto.Item) {
				v := item.Value.(*ristrettoItem)
				logger.Debug().Str("key", v.key).Msg("OnEvict")
				err := s3.RemoveObject(context.Background(), s3bucket, v.key, minio.RemoveObjectOptions{})
				if err != nil {
					logger.Err(err).Str("key", v.key).Msg("failed to remove object")
				}
			},
		}))
	}

	c := &Cache{
		s3:     s3,
		memory: cache,
		bucket: s3bucket,

		memoryCacheRatio: prometheus.NewGauge(prometheus.GaugeOpts{Name: "chii_img_memory_cache_hit_radio"}),
		memoryCacheHit:   prometheus.NewGauge(prometheus.GaugeOpts{Name: "chii_img_memory_cache_hit_count"}),
		memoryCacheMiss:  prometheus.NewGauge(prometheus.GaugeOpts{Name: "chii_img_memory_cache_miss_count"}),
		memorySize:       prometheus.NewGauge(prometheus.GaugeOpts{Name: "chii_img_memory_cache_size"}),
	}

	if cacheSize != 0 {
		go func() {
			files := c.s3.ListObjects(context.Background(), s3bucket, minio.ListObjectsOptions{
				Prefix:    "/",
				Recursive: true,
			})

			for file := range files {
				if file.Err != nil {
					logger.Err(file.Err).Msg("failed to list files")
					break
				}

				key := "/" + file.Key
				logger.Debug().Str("key", key).Msg("set memory cache")
				c.memory.Set(key, &ristrettoItem{key: key}, 1)
			}
		}()
	}

	return c
}

type Cache struct {
	memory *ristretto.Cache
	s3     *minio.Client

	bucket string

	memoryCacheRatio prometheus.Gauge
	memoryCacheMiss  prometheus.Gauge
	memoryCacheHit   prometheus.Gauge
	memorySize       prometheus.Gauge
}

func (c *Cache) Describe(chan<- *prometheus.Desc) {
}

func (c *Cache) Collect(metrics chan<- prometheus.Metric) {
	if cacheSize == 0 {
		return
	}

	c.memoryCacheRatio.Set(c.memory.Metrics.Ratio())
	metrics <- c.memoryCacheRatio

	c.memoryCacheHit.Set(float64(c.memory.Metrics.Hits()))
	metrics <- c.memoryCacheHit

	c.memoryCacheMiss.Set(float64(c.memory.Metrics.Misses()))
	metrics <- c.memoryCacheMiss

	c.memorySize.Set(float64(c.memory.Metrics.CostAdded() - c.memory.Metrics.CostEvicted()))
	metrics <- c.memorySize
}

func (c *Cache) Get(ctx context.Context, key string) (item Image, exist bool, err error) {
	if c.memory != nil {
		if _, cached := c.memory.Get(key); !cached {
			return item, false, nil
		}
	}

	stat, err := c.s3.StatObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		// stupid golang error handling
		var e minio.ErrorResponse
		if errors.As(err, &e) {
			if e.Code == "NoSuchKey" {
				return item, false, nil
			}
		}

		return item, false, err
	}

	obj, err := c.s3.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return item, false, fmt.Errorf("failed to get raw image from newS3Client: %w", err)
	}
	defer obj.Close()

	raw, err := io.ReadAll(obj)
	return Image{body: raw, contentType: stat.Metadata.Get("Content-Type")}, true, err
}

func (c *Cache) Set(ctx context.Context, key string, value Image) error {
	_, err := c.s3.PutObject(ctx, c.bucket, key, bytes.NewBuffer(value.body), int64(len(value.body)), minio.PutObjectOptions{
		ContentType: value.contentType,
	})
	if err != nil {
		return err
	}

	if c.memory != nil {
		c.memory.Set(key, &ristrettoItem{key: key}, 1)
	}

	return nil
}
