//go:build integration

package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestAuthRegisterRateLimitThresholdHitReturns429(t *testing.T) {
	rdb := startAuthRouteRedis(t)

	router := newAuthRoutesTestRouter(rdb)
	const path = "/api/v1/auth/register"

	for i := 1; i <= 6; i++ {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "198.51.100.10:23456"

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if i <= 5 {
			require.Equal(t, http.StatusBadRequest, w.Code, "第 %d 次请求应先进入业务校验", i)
			continue
		}
		require.Equal(t, http.StatusTooManyRequests, w.Code, "第 6 次请求应命中限流")
		require.Contains(t, w.Body.String(), "rate limit exceeded")
	}
}

func startAuthRouteRedis(t *testing.T) *redis.Client {
	t.Helper()

	miniRedis := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
		DB:   0,
	})
	t.Cleanup(func() {
		_ = rdb.Close()
	})
	return rdb
}
