package scheduler

import (
	"github.com/go-redis/redis/v8"
)

type Redis struct {
	client *redis.Client
}

func NewRedis(addr string) *Redis {
	return &Redis{
		client: redis.NewClient(&redis.Options{Addr: addr}),
	}
}
