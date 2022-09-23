package core

import "github.com/go-redis/redis/v8"

// JwtIDPrefix is the jwt-id prefix while creating the key
const JwtIDPrefix = "jti:"

// RedisDB wrapper around the redis db client.
type RedisDB interface {
	// Client is the redis client
	Client() redis.UniversalClient
}
