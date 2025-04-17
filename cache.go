package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
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

	var cache *ristretto.Cache[string, *ristrettoItem]
	if cacheSize != 0 {
		cache = lo.Must(ristretto.NewCache[string, *ristrettoItem](&ristretto.Config[string, *ristrettoItem]{
			NumCounters:        int64(cacheSize * 10), // number of keys to track frequency of (10M).
			MaxCost:            int64(cacheSize),
			BufferItems:        64, // number of keys per Get buffer.
			Metrics:            true,
			IgnoreInternalCost: true,
			OnEvict: func(item *ristretto.Item[*ristrettoItem]) {
				go func() {
					log.Debug().Str("key", item.Value.key).Msg("OnEvict")
					_, err := s3Client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
						Bucket: &s3bucket,
						Key:    &item.Value.key,
					})
					if err != nil {
						log.Err(err).Str("key", item.Value.key).Msg("failed to remove object")
					}
				}()
			},
		}))
	}

	c := &Cache{
		s3:     s3Client,
		s3TTL:  s3TTL,
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
	}

	return c
}

type Cache struct {
	memory *ristretto.Cache[string, *ristrettoItem]
	s3     *s3.Client
	s3TTL  time.Duration

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

	obj, err := c.s3.GetObject(ctx, &s3.GetObjectInput{Bucket: &s3bucket, Key: &key})
	if err != nil {
		var e smithy.APIError
		if errors.As(err, &e) {
			if e.ErrorCode() == "NoSuchKey" {
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
	var expire *time.Time = nil
	if c.s3TTL != 0 {
		expire = lo.ToPtr(time.Now().Add(c.s3TTL))
	}

	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Body:        bytes.NewReader(value.body),
		Bucket:      &s3bucket,
		ContentType: &value.contentType,
		Key:         &key,
		Expires:     expire,
	})
	if err != nil {
		return err
	}

	if c.memory != nil {
		c.memory.Set(key, &ristrettoItem{key: key}, 1)
	}

	return nil
}
