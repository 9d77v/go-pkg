package redis

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/9d77v/go-pkg/env"
	redis "github.com/go-redis/redis/v8"
)

//环境变量
var (
	redisAddress  = env.GetEnvStr("REDIS_ADDRESS", "domain.local:7000,domain.local:7001,domain.local:7002,domain.local:7003,domain.local:7004,domain.local:7005")
	redisPassword = env.GetEnvStr("REDIS_PASSWORD", "")
	client        *Client
	once          sync.Once
)

type Client struct {
	redis.UniversalClient
}

//GetClient get redis connection
func GetClient() *Client {
	once.Do(func() {
		client = newClient()
	})
	return client
}

func newClient() *Client {
	return &Client{redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    strings.Split(redisAddress, ","),
		Password: redisPassword,
	})}
}

//DLock 分布式锁
func (c *Client) DLock(ctx context.Context, key string, expire time.Duration, f func()) {
	value := rand.Int()
	err := c.SetEX(ctx, key, value, expire).Err()
	if err == nil {
		f()
		c.DelIfEquals(ctx, key, value)
	} else {
		log.Println("get redis lock failed", err)
	}
}

//DelIfEquals 删除key,value都相等的key
func (c *Client) DelIfEquals(ctx context.Context, key string, value interface{}) {
	script := redis.NewScript(`
		if redis.call("get",KEYS[1])==ARGV[1] then
			return redis.call("del",KEYS[1])
		else
			return 0
		end
	`)
	sha, err := script.Load(ctx, c).Result()
	if err != nil {
		log.Panicln("script is wrong,err:", err)
	}
	_, err = c.EvalSha(ctx, sha, []string{key}, value).Result()
	if err != nil {
		log.Println("exec failed,err:", err)
	}
}
