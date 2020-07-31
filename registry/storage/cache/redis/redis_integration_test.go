// +build integration

package redis_test

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/docker/distribution/registry/storage/cache/cachecheck"
	rediscache "github.com/docker/distribution/registry/storage/cache/redis"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
)

var (
	addr            string
	mainName        string
	password        string
	db              int
	dialTimeout     time.Duration
	readTimeout     time.Duration
	writeTimeout    time.Duration
	poolMaxIdle     int
	poolMaxActive   int
	poolIdleTimeout time.Duration
)

func init() {
	addr = os.Getenv("REDIS_ADDR")
	mainName = os.Getenv("REDIS_MAIN_NAME")
	password = os.Getenv("REDIS_PASSWORD")
	db = mustParseEnvVarAsInt("REDIS_DB")
	dialTimeout = mustParseEnvVarAsDuration("REDIS_DIAL_TIMEOUT")
	readTimeout = mustParseEnvVarAsDuration("REDIS_READ_TIMEOUT")
	writeTimeout = mustParseEnvVarAsDuration("REDIS_WRITE_TIMEOUT")
	poolMaxIdle = mustParseEnvVarAsInt("REDIS_POOL_MAX_IDLE")
	poolMaxActive = mustParseEnvVarAsInt("REDIS_POOL_MAX_ACTIVE")
	poolIdleTimeout = mustParseEnvVarAsDuration("REDIS_POOL_IDLE_TIMEOUT")
}

func mustParseEnvVarAsInt(name string) int {
	s := os.Getenv(name)
	if s == "" {
		return 0
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("error parsing %q environment variable: %v", name, err)
	}

	return i
}

func mustParseEnvVarAsDuration(name string) time.Duration {
	s := os.Getenv(name)
	if s == "" {
		return 0
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf("error parsing %q environment variable: %v", name, err)
	}

	return d
}

func isEligible(t *testing.T) {
	t.Helper()

	if addr == "" {
		t.Skip("the 'REDIS_ADDR' environment variable must be set to enable these tests")
	}
}

func poolOptsFromEnv() *rediscache.PoolOpts {
	return &rediscache.PoolOpts{
		Addr:            addr,
		MainName:        mainName,
		Password:        password,
		DB:              db,
		DialTimeout:     dialTimeout,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		PoolMaxIdle:     poolMaxIdle,
		PoolMaxActive:   poolMaxActive,
		PoolIdleTimeout: poolIdleTimeout,
	}
}

func flushDB(t *testing.T, pool *redis.Pool) {
	t.Helper()

	conn := pool.Get()
	defer conn.Close()

	if _, err := conn.Do("FLUSHDB"); err != nil {
		t.Fatalf("unexpected error flushing redis db: %v", err)
	}
}

// TestRedisLayerInfoCache exercises a live redis instance using the cache
// implementation.
func TestRedisBlobDescriptorCacheProvider(t *testing.T) {
	isEligible(t)

	pool := rediscache.NewPool(poolOptsFromEnv())
	flushDB(t, pool)

	cachecheck.CheckBlobDescriptorCache(t, rediscache.NewRedisBlobDescriptorCacheProvider(pool))
}
