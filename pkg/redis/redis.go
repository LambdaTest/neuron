package redis

import (
	"context"
	"crypto/tls"
	"runtime"
	"strings"
	"time"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/go-redis/redis/v8"
)

const minClusterNodes = 2

type redisDB struct {
	client redis.UniversalClient
}

// New initializes a pool redis client connections.
func New(ctx context.Context, cfg *config.Config, logger lumber.Logger) (core.RedisDB, error) {
	addrs := strings.Split(cfg.Redis.Addr, ",")

	if len(addrs) >= minClusterNodes {
		logger.Debugf("Creating Redis Cluster Client")
	} else {
		logger.Debugf("Creating Redis Client")
	}

	// TODO: find optimal values for PoolSize, IdleTimeout MaxConnAge
	options := &redis.UniversalOptions{
		Addrs:              addrs,
		IdleTimeout:        5 * time.Minute,
		IdleCheckFrequency: 1 * time.Minute,
		// 10 connections per every available CPU as reported by runtime.GOMAXPROCS
		PoolSize:   10 * runtime.GOMAXPROCS(0),
		MaxRetries: 3,
	}

	if cfg.Env != constants.Dev {
		options.Password = cfg.Redis.Password
		if cfg.Redis.TLS {
			options.TLSConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}
	}

	// if the number of Addrs is two or more, a ClusterClient is returned
	// otherwise a single-node Client is returned.
	client := redis.NewUniversalClient(options)

	// ping the redis to check the connection.
	if _, err := client.Ping(ctx).Result(); err != nil {
		client.Close()
		return nil, err
	}
	logger.Infof("Redis connection created successfully.")

	return &redisDB{
		client: client,
	}, nil
}

// Client exposes redis client interface
func (r *redisDB) Client() redis.UniversalClient {
	return r.client
}
