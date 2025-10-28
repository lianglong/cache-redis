package cacheredis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lianglong/cache"
	"github.com/redis/go-redis/v9"
)

func init() {
	cache.Register("redis", func(config cache.Config) (cache.Cache, error) {
		client, err := NewRedisStore(config)
		return &redisCache{client: client}, err
	})
}

func NewRedisStore(config cache.Config) (*redis.Client, error) {
	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// 应用默认配置
	if config.DialTimeout == 0 {
		config.DialTimeout = 5 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 3 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 3 * time.Second
	}

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,

		// 连接池配置
		PoolSize:        config.Pool.MaxActiveConns,
		MinIdleConns:    config.Pool.MaxIdleConns / 2, // 最小空闲连接为最大的一半
		MaxIdleConns:    config.Pool.MaxIdleConns,
		ConnMaxIdleTime: config.Pool.IdleTimeout,
		ConnMaxLifetime: config.Pool.MaxConnLifetime,
		PoolTimeout:     4 * time.Second,

		// 其他配置
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	})

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis at %s: %w", config.Addr, err)
	}
	return client, nil
}

type redisCache struct {
	client *redis.Client
}

// ============================================
// 基础操作
// ============================================

func (rc *redisCache) Get(ctx context.Context, key string) (string, error) {
	result, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", cache.ErrNotFound
		}
		return "", err
	}
	return result, nil
}

func (rc *redisCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	result, err := rc.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, cache.ErrNotFound
		}
		return nil, err
	}
	return result, nil
}

func (rc *redisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return rc.client.Set(ctx, key, value, expiration).Err()
}

func (rc *redisCache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return rc.client.SetNX(ctx, key, value, expiration).Result()
}

func (rc *redisCache) Delete(ctx context.Context, key string) error {
	return rc.client.Del(ctx, key).Err()
}

func (rc *redisCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	return rc.client.Exists(ctx, keys...).Result()
}

// ============================================
// 批量操作
// ============================================

func (rc *redisCache) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return rc.client.MGet(ctx, keys...).Result()
}

func (rc *redisCache) MSet(ctx context.Context, pairs map[string]interface{}) error {
	// 将 map 转换为 key-value 交替的切片
	args := make([]interface{}, 0, len(pairs)*2)
	for k, v := range pairs {
		args = append(args, k, v)
	}
	return rc.client.MSet(ctx, args...).Err()
}

func (rc *redisCache) MDelete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return rc.client.Del(ctx, keys...).Err()
}

// ============================================
// 数值操作
// ============================================

func (rc *redisCache) Incr(ctx context.Context, key string) (int64, error) {
	return rc.client.Incr(ctx, key).Result()
}

func (rc *redisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return rc.client.IncrBy(ctx, key, value).Result()
}

func (rc *redisCache) Decr(ctx context.Context, key string) (int64, error) {
	return rc.client.Decr(ctx, key).Result()
}

func (rc *redisCache) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return rc.client.DecrBy(ctx, key, value).Result()
}

// ============================================
// TTL 操作
// ============================================

func (rc *redisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := rc.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Redis 返回 -2 表示键不存在，-1 表示没有过期时间
	if ttl < 0 {
		return 0, cache.ErrNotFound
	}
	return ttl, nil
}

func (rc *redisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return rc.client.Expire(ctx, key, expiration).Err()
}

func (rc *redisCache) Persist(ctx context.Context, key string) error {
	return rc.client.Persist(ctx, key).Err()
}

// ============================================
// Hash 操作
// ============================================

func (rc *redisCache) HGet(ctx context.Context, key, field string) (string, error) {
	result, err := rc.client.HGet(ctx, key, field).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", cache.ErrNotFound
		}
		return "", err
	}
	return result, nil
}

func (rc *redisCache) HSet(ctx context.Context, key, field string, value interface{}) error {
	return rc.client.HSet(ctx, key, field, value).Err()
}

func (rc *redisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return rc.client.HGetAll(ctx, key).Result()
}

func (rc *redisCache) HDel(ctx context.Context, key string, fields ...string) error {
	return rc.client.HDel(ctx, key, fields...).Err()
}

// ============================================
// 列表操作
// ============================================

func (rc *redisCache) LPush(ctx context.Context, key string, values ...interface{}) error {
	return rc.client.LPush(ctx, key, values...).Err()
}

func (rc *redisCache) RPush(ctx context.Context, key string, values ...interface{}) error {
	return rc.client.RPush(ctx, key, values...).Err()
}

func (rc *redisCache) LPop(ctx context.Context, key string) (string, error) {
	result, err := rc.client.LPop(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", cache.ErrNotFound
		}
		return "", err
	}
	return result, nil
}

func (rc *redisCache) RPop(ctx context.Context, key string) (string, error) {
	result, err := rc.client.RPop(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", cache.ErrNotFound
		}
		return "", err
	}
	return result, nil
}

func (rc *redisCache) LLen(ctx context.Context, key string) (int64, error) {
	return rc.client.LLen(ctx, key).Result()
}

// ============================================
// 集合操作
// ============================================

func (rc *redisCache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return rc.client.SAdd(ctx, key, members...).Err()
}

func (rc *redisCache) SMembers(ctx context.Context, key string) ([]string, error) {
	return rc.client.SMembers(ctx, key).Result()
}

func (rc *redisCache) SRem(ctx context.Context, key string, members ...interface{}) error {
	return rc.client.SRem(ctx, key, members...).Err()
}

// ============================================
// 管理操作
// ============================================

func (rc *redisCache) Ping(ctx context.Context) error {
	return rc.client.Ping(ctx).Err()
}

func (rc *redisCache) FlushDB(ctx context.Context) error {
	return rc.client.FlushDB(ctx).Err()
}

func (rc *redisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return rc.client.Keys(ctx, pattern).Result()
}

func (rc *redisCache) Close() error {
	return rc.client.Close()
}
