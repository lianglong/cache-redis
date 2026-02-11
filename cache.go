package cacheredis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lianglong/cache"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
)

// Logger 接口定义，用于集成自定义日志系统
// 实现此接口以使用你自己的日志库（如 logrus, zap, slog 等）
type Logger interface {
	Printf(ctx context.Context, format string, v ...interface{})
}

// redisLoggerAdapter 适配器，用于将自定义 Logger 适配到 go-redis 内部使用
// 虽然我们不能直接引用 internal.Logging 类型，但可以通过结构体实现相同的方法签名
type redisLoggerAdapter struct {
	logger Logger
}

func (a *redisLoggerAdapter) Printf(ctx context.Context, format string, v ...interface{}) {
	if a.logger != nil {
		a.logger.Printf(ctx, format, v...)
	}
}

// SetRedisLogger 设置 go-redis 的全局日志记录器
// 注意：这会影响所有使用 go-redis 的客户端实例
func SetRedisLogger(logger Logger) {
	if logger != nil {
		adapter := &redisLoggerAdapter{logger: logger}
		// 使用类型断言绕过编译器检查，将适配器传递给 redis.SetLogger
		// 这利用了 Go 的鸭子类型特性：只要方法签名匹配即可
		redis.SetLogger(adapter)
	}
}

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

	// 处理 MaintNotificationsConfig
	var maintNotificationsConfig *maintnotifications.Config
	if config.Extra != nil {
		// 检查是否禁用 maintnotifications
		if disableMaint, ok := config.Extra["DisableMaintNotifications"].(bool); ok && disableMaint {
			maintNotificationsConfig = &maintnotifications.Config{
				Mode: maintnotifications.ModeDisabled,
			}
		}

		// 从 Extra 中获取 Logger 并设置全局日志记录器
		if logger, ok := config.Extra["Logger"].(Logger); ok && logger != nil {
			SetRedisLogger(logger)
		}
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

		// 维护通知配置
		MaintNotificationsConfig: maintNotificationsConfig,
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
// Pub/Sub 操作
// ============================================

func (rc *redisCache) Publish(ctx context.Context, channel string, message string) error {
	return rc.client.Publish(ctx, channel, message).Err()
}

func (rc *redisCache) Subscribe(ctx context.Context, channels ...string) (cache.PubSub, error) {
	pubsub := rc.client.Subscribe(ctx, channels...)
	// 等待订阅确认
	_, err := pubsub.Receive(ctx)
	if err != nil {
		pubsub.Close()
		return nil, err
	}
	return &redisPubSub{pubsub: pubsub}, nil
}

// redisPubSub Redis Pub/Sub 实现
type redisPubSub struct {
	pubsub *redis.PubSub
}

func (p *redisPubSub) Channel() <-chan *cache.Message {
	ch := make(chan *cache.Message)
	go func() {
		defer close(ch)
		for msg := range p.pubsub.Channel() {
			ch <- &cache.Message{
				Channel: msg.Channel,
				Payload: msg.Payload,
			}
		}
	}()
	return ch
}

func (p *redisPubSub) Close() error {
	return p.pubsub.Close()
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

// ============================================
// Lua 脚本操作
// ============================================

// Eval 执行 Lua 脚本
// script: Lua 脚本内容
// keys: KEYS 数组，脚本中通过 KEYS[1], KEYS[2] 等访问
// args: ARGV 数组，脚本中通过 ARGV[1], ARGV[2] 等访问
// 返回脚本执行结果
func (rc *redisCache) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return rc.client.Eval(ctx, script, keys, args...).Result()
}

// EvalSha 通过 SHA1 哈希执行预加载的 Lua 脚本
// sha1: 脚本的 SHA1 哈希值（通过 ScriptLoad 获得）
// keys: KEYS 数组
// args: ARGV 数组
func (rc *redisCache) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	return rc.client.EvalSha(ctx, sha1, keys, args...).Result()
}

// ScriptLoad 加载 Lua 脚本到 Redis 服务器
// 返回脚本的 SHA1 哈希值，可用于后续的 EvalSha 调用
func (rc *redisCache) ScriptLoad(ctx context.Context, script string) (string, error) {
	return rc.client.ScriptLoad(ctx, script).Result()
}

// ScriptExists 检查一个或多个脚本是否已加载到 Redis
// sha1s: 要检查的脚本 SHA1 哈希值列表
// 返回布尔数组，表示每个脚本是否存在
func (rc *redisCache) ScriptExists(ctx context.Context, sha1s ...string) ([]bool, error) {
	return rc.client.ScriptExists(ctx, sha1s...).Result()
}

// ScriptFlush 清除 Redis 服务器上所有已加载的 Lua 脚本
func (rc *redisCache) ScriptFlush(ctx context.Context) error {
	return rc.client.ScriptFlush(ctx).Err()
}

// Script 封装了一个可重复使用的 Lua 脚本
type Script struct {
	script *redis.Script
}

// NewScript 创建一个新的脚本对象
// script: Lua 脚本内容
// 返回的 Script 对象会自动管理脚本的加载和执行
func (rc *redisCache) NewScript(script string) *Script {
	return &Script{
		script: redis.NewScript(script),
	}
}

// Run 执行脚本
// ctx: 上下文
// client: Redis 客户端（传入 redisCache.client）
// keys: KEYS 数组
// args: ARGV 数组
func (s *Script) Run(ctx context.Context, client *redis.Client, keys []string, args ...interface{}) (interface{}, error) {
	return s.script.Run(ctx, client, keys, args...).Result()
}

// Load 加载脚本到 Redis
func (s *Script) Load(ctx context.Context, client *redis.Client) (string, error) {
	return s.script.Load(ctx, client).Result()
}

// Exists 检查脚本是否已加载
func (s *Script) Exists(ctx context.Context, client *redis.Client) ([]bool, error) {
	return s.script.Exists(ctx, client).Result()
}

// Hash 返回脚本的 SHA1 哈希值
func (s *Script) Hash() string {
	return s.script.Hash()
}

// GetClient 获取底层的 Redis 客户端，用于 Script 操作
// 使用示例：
//
//	script := cache.NewScript("return redis.call('get', KEYS[1])")
//	result, err := script.Run(ctx, cache.GetClient(), []string{"mykey"})
func (rc *redisCache) GetClient() *redis.Client {
	return rc.client
}
