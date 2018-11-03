package cache

import (
	"encoding/json"
	"time"

	redisClient "github.com/go-redis/redis"
)

type redis struct {
	client *redisClient.Client
}

func Redis(options *redisClient.Options) Storage {
	clnt := redisClient.NewClient(options)
	return redis{
		client: clnt,
	}
}

func (r redis) Write(key string, v interface{}, expiration time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return r.client.Set(key, string(b), expiration).Err()
}

func (r redis) Read(key string) (interface{}, error) {
	result := r.client.Get(key)
	if result.Err() != nil {
		if result.Err() == redisClient.Nil {
			return nil, ErrKeyNotExist
		}
		return nil, result.Err()
	}
	return result.Result()
}

func (r redis) Delete(key string) error {
	return r.client.Del(key).Err()
}

func (r redis) Flush() error {
	return r.Flush()
}
