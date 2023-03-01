package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dgraph-io/ristretto"
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
	s3Client := newS3Client()

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
				_, err := s3Client.DeleteObject(&s3.DeleteObjectInput{
					Bucket: &s3bucket,
					Key:    &v.key,
				})
				if err != nil {
					logger.Err(err).Str("key", v.key).Msg("failed to remove object")
				}
			},
		}))
	}

	c := &Cache{
		s3:     s3Client,
		memory: cache,
		bucket: s3bucket,
	}

	if cacheSize != 0 {
		c.memoryCacheRatio = prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{Name: "chii_img_memory_cache_hit_radio"},
			func() float64 {
				return c.memory.Metrics.Ratio()
			},
		)

		c.memoryCacheHit = prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{Name: "chii_img_memory_cache_hit_count"},
			func() float64 {
				return float64(c.memory.Metrics.Hits())
			},
		)

		c.memoryCacheMiss = prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{Name: "chii_img_memory_cache_miss_count"}, func() float64 {
				return float64(c.memory.Metrics.Misses())
			},
		)

		c.memorySize = prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{Name: "chii_img_memory_cache_size"},
			func() float64 {
				return float64(c.memory.Metrics.CostAdded() - c.memory.Metrics.CostEvicted())
			},
		)

		go func() {
			err := c.s3.ListObjectsV2Pages(&s3.ListObjectsV2Input{
				Bucket: &s3bucket,
				Prefix: lo.ToPtr("/"),
			}, func(o *s3.ListObjectsV2Output, b bool) bool {
				for _, file := range o.Contents {
					key := "/" + *file.Key
					logger.Debug().Str("key", key).Msg("set memory cache")
					c.memory.Set(key, &ristrettoItem{key: key}, 1)
				}

				return true
			})
			if err != nil {
				logger.Fatal().Err(err).Msg("failed to list objects")
			} else {
				logger.Info().Msg("finish loading s3 objects")
			}
		}()
	}

	return c
}

type Cache struct {
	memory *ristretto.Cache
	s3     *s3.S3

	bucket string

	memoryCacheRatio prometheus.GaugeFunc
	memoryCacheMiss  prometheus.GaugeFunc
	memoryCacheHit   prometheus.GaugeFunc
	memorySize       prometheus.GaugeFunc
}

func (c *Cache) Describe(chan<- *prometheus.Desc) {
}

func (c *Cache) Collect(metrics chan<- prometheus.Metric) {
	if cacheSize == 0 {
		return
	}

	metrics <- c.memoryCacheRatio
	metrics <- c.memoryCacheHit
	metrics <- c.memoryCacheMiss
	metrics <- c.memorySize
}

func (c *Cache) Get(ctx context.Context, key string) (item Image, exist bool, err error) {
	if c.memory != nil {
		if _, cached := c.memory.Get(key); !cached {
			return item, false, nil
		}
	}

	obj, err := c.s3.GetObjectWithContext(ctx, &s3.GetObjectInput{Bucket: &s3bucket, Key: &key})
	if err != nil {
		var e awserr.Error
		if errors.As(err, &e) {
			if e.Code() == s3.ErrCodeNoSuchKey {
				if c.memory != nil {
					c.memory.Del(key)
				}
				return item, false, nil
			}
		}

		return item, false, fmt.Errorf("failed to get raw image from s3: %w", err)
	}
	defer obj.Body.Close()

	raw, err := io.ReadAll(obj.Body)
	return Image{body: raw, contentType: *obj.ContentType}, true, err
}

func (c *Cache) Set(ctx context.Context, key string, value Image) error {
	_, err := c.s3.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Body:        bytes.NewReader(value.body),
		Bucket:      &s3bucket,
		ContentType: &value.contentType,
		Key:         &key,
	})
	if err != nil {
		return err
	}

	if c.memory != nil {
		c.memory.Set(key, &ristrettoItem{key: key}, 1)
	}

	return nil
}
