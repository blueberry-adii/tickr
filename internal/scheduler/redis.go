package scheduler

import (
	"github.com/go-redis/redis/v8"
)

/*
Redis struct to hold single redis client, to access
redis methods on lists and sets
*/
type Redis struct {
	client *redis.Client
}

/*
Redis constructor
*/
func NewRedis(addr string) *Redis {
	return &Redis{
		client: redis.NewClient(&redis.Options{Addr: addr}),
	}
}
