package main

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

func (h Handle) Metrics(c echo.Context) error {
	buf := bytes.NewBuffer(nil)

	fmt.Fprintf(buf, "img_proxy_cache_ratio{} %f\n", h.cache.cache.Metrics.Ratio())

	fmt.Fprintf(buf, "img_proxy_cache_hit{} %d\n", h.cache.cache.Metrics.Hits())
	fmt.Fprintf(buf, "img_proxy_cache_miss{} %d\n", h.cache.cache.Metrics.Misses())

	fmt.Fprintf(buf, "img_proxy_cache_key_add{} %d\n", h.cache.cache.Metrics.KeysAdded())
	fmt.Fprintf(buf, "img_proxy_cache_key_update{} %d\n", h.cache.cache.Metrics.KeysUpdated())
	fmt.Fprintf(buf, "img_proxy_cache_key_evict{} %d\n", h.cache.cache.Metrics.KeysEvicted())

	fmt.Fprintf(buf, "img_proxy_cache_cost_added{} %d\n", h.cache.cache.Metrics.CostAdded())
	fmt.Fprintf(buf, "img_proxy_cache_cost_evicted{} %d\n", h.cache.cache.Metrics.CostEvicted())

	fmt.Fprintf(buf, "img_proxy_cache_cost_drop_sets{} %d\n", h.cache.cache.Metrics.SetsDropped())
	fmt.Fprintf(buf, "img_proxy_cache_cost_reject_sets{} %d\n", h.cache.cache.Metrics.SetsRejected())

	fmt.Fprintf(buf, "img_proxy_cache_cost_drop_gets{} %d\n", h.cache.cache.Metrics.GetsDropped())
	fmt.Fprintf(buf, "img_proxy_cache_cost_keep_gets{} %d\n", h.cache.cache.Metrics.GetsKept())

	return c.String(http.StatusOK, buf.String())
}
