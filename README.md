# Cache-Redis

[![Go Version](https://img.shields.io/badge/go-1.19+-blue.svg)](https://golang.org)
[![Go Reference](https://pkg.go.dev/badge/github.com/lianglong/cache-redis.svg)](https://pkg.go.dev/github.com/lianglong/cache-redis)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

Redis 驱动实现，用于 [cache](https://github.com/lianglong/cache) 通用缓存接口库。

## 简介

cache-redis 是 cache 库的 Redis 驱动实现，基于 [go-redis/redis](https://github.com/redis/go-redis) v9 客户端，提供完整的缓存操作支持。

## 特性

- ✅ **完整实现** - 实现 cache.Cache 接口的所有 30+ 方法
- 🔌 **自动注册** - 导入即自动注册到 cache 驱动系统
- ⚡ **高性能** - 基于 go-redis v9，支持连接池和管道
- 🔧 **灵活配置** - 完整的连接池和超时配置
- 🛡️ **错误处理** - 统一的错误类型，便于错误判断
- 🔄 **连接验证** - 启动时自动验证 Redis 连接
- 📊 **数据结构** - 支持 String、Hash、List、Set 等数据结构
- 🔢 **原子操作** - Incr、Decr 等原子计数操作
- ⏰ **TTL 管理** - 灵活的过期时间控制

## 安装

```bash
go get github.com/lianglong/cache-redis
```

**依赖：**
```bash
go get github.com/lianglong/cache
go get github.com/redis/go-redis/v9
```

## 快速开始

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/lianglong/cache"
    _ "github.com/lianglong/cache-redis"  // 导入 Redis 驱动（自动注册）
)

func main() {
    ctx := context.Background()

    // 创建 Redis 缓存实例
    c, err := cache.New("redis", cache.Config{
        Addr:     "127.0.0.1:6379",
        Password: "your-password",  // 如果没有密码可留空
        DB:       0,                // Redis 数据库索引
    })
    if err != nil {
        log.Fatalf("Failed to create cache: %v", err)
    }
    defer c.Close()

    // 设置值
    err = c.Set(ctx, "hello", "world", time.Hour)
    if err != nil {
        log.Fatal(err)
    }

    // 获取值
    value, err := c.Get(ctx, "hello")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(value)  // 输出: world
}
```

### 完整配置

```go
c, err := cache.New("redis", cache.Config{
    // 基础配置
    Addr:     "127.0.0.1:6379",
    Password: "your-password",
    DB:       0,

    // 超时配置
    DialTimeout:  5 * time.Second,   // 连接超时
    ReadTimeout:  3 * time.Second,   // 读超时
    WriteTimeout: 3 * time.Second,   // 写超时

    // 连接池配置
    Pool: cache.PoolConfig{
        MaxIdleConns:    10,                  // 最大空闲连接数
        MaxActiveConns:  100,                 // 最大活跃连接数
        IdleTimeout:     5 * time.Minute,     // 空闲连接超时
        MaxConnLifetime: 1 * time.Hour,       // 连接最大生存时间
    },
})
```

### 使用默认配置

```go
config := cache.DefaultConfig()
config.Addr = "127.0.0.1:6379"
config.Password = "your-password"

c, err := cache.New("redis", config)
```

## 功能示例

### 1. 基础 Key-Value 操作

```go
ctx := context.Background()

// 设置值（带过期时间）
c.Set(ctx, "user:1", "Alice", time.Hour)

// 获取值
value, err := c.Get(ctx, "user:1")
if err != nil {
    if cache.IsNotFound(err) {
        fmt.Println("Key not found")
    }
}

// 删除值
c.Delete(ctx, "user:1")

// 检查键是否存在
count, _ := c.Exists(ctx, "user:1", "user:2")
fmt.Printf("Exists count: %d\n", count)

// 仅当键不存在时设置
success, _ := c.SetNX(ctx, "user:1", "Alice", time.Hour)
if success {
    fmt.Println("Key set successfully")
}
```

### 2. 批量操作

```go
// 批量设置
c.MSet(ctx, map[string]interface{}{
    "user:1": "Alice",
    "user:2": "Bob",
    "user:3": "Charlie",
})

// 批量获取
values, _ := c.MGet(ctx, "user:1", "user:2", "user:3")
for i, val := range values {
    fmt.Printf("user:%d = %v\n", i+1, val)
}

// 批量删除
c.MDelete(ctx, "user:1", "user:2", "user:3")
```

### 3. 计数器操作

```go
// 递增
count, _ := c.Incr(ctx, "page_views")
fmt.Printf("Page views: %d\n", count)

// 增加指定值
count, _ = c.IncrBy(ctx, "page_views", 10)
fmt.Printf("Page views after +10: %d\n", count)

// 递减
count, _ = c.Decr(ctx, "inventory:item:123")
fmt.Printf("Inventory: %d\n", count)

// 减少指定值
count, _ = c.DecrBy(ctx, "inventory:item:123", 5)
fmt.Printf("Inventory after -5: %d\n", count)
```

### 4. TTL（过期时间）管理

```go
// 设置键的过期时间
c.Set(ctx, "session:abc", "data", time.Hour)

// 获取剩余 TTL
ttl, _ := c.TTL(ctx, "session:abc")
fmt.Printf("TTL: %v\n", ttl)

// 更新过期时间
c.Expire(ctx, "session:abc", 2*time.Hour)

// 移除过期时间（设为永久）
c.Persist(ctx, "session:abc")
```

### 5. Hash 操作

```go
// 设置 Hash 字段
c.HSet(ctx, "user:1:profile", "name", "Alice")
c.HSet(ctx, "user:1:profile", "age", 30)
c.HSet(ctx, "user:1:profile", "email", "alice@example.com")

// 获取单个字段
name, _ := c.HGet(ctx, "user:1:profile", "name")
fmt.Printf("Name: %s\n", name)

// 获取所有字段
profile, _ := c.HGetAll(ctx, "user:1:profile")
for field, value := range profile {
    fmt.Printf("%s: %s\n", field, value)
}

// 删除字段
c.HDel(ctx, "user:1:profile", "email")
```

### 6. 列表操作

```go
// 从左侧推入
c.LPush(ctx, "queue:tasks", "task1", "task2", "task3")

// 从右侧推入
c.RPush(ctx, "queue:tasks", "task4")

// 从左侧弹出
task, _ := c.LPop(ctx, "queue:tasks")
fmt.Printf("Popped task: %s\n", task)

// 从右侧弹出（FIFO 队列）
task, _ = c.RPop(ctx, "queue:tasks")
fmt.Printf("Popped task: %s\n", task)

// 获取列表长度
length, _ := c.LLen(ctx, "queue:tasks")
fmt.Printf("Queue length: %d\n", length)
```

### 7. 集合操作

```go
// 添加成员
c.SAdd(ctx, "tags:article:1", "golang", "redis", "cache")

// 获取所有成员
tags, _ := c.SMembers(ctx, "tags:article:1")
fmt.Printf("Tags: %v\n", tags)

// 删除成员
c.SRem(ctx, "tags:article:1", "cache")

// 检查成员数量
tags, _ = c.SMembers(ctx, "tags:article:1")
fmt.Printf("Tags count: %d\n", len(tags))
```

### 8. 管理操作

```go
// 健康检查
if err := c.Ping(ctx); err != nil {
    log.Fatal("Redis connection lost")
}

// 查找键（谨慎使用，生产环境不推荐）
keys, _ := c.Keys(ctx, "user:*")
fmt.Printf("Found keys: %v\n", keys)

// 清空当前数据库（危险操作！）
// c.FlushDB(ctx)

// 关闭连接
c.Close()
```

## 命名空间使用

配合 cache 的命名空间功能，实现键隔离：

```go
import "github.com/lianglong/cache"

// 创建基础缓存
c, _ := cache.New("redis", cache.Config{
    Addr: "127.0.0.1:6379",
})

// 创建不同业务模块的命名空间
userCache := cache.NewNamespace(c, "user")
sessionCache := cache.NewNamespace(c, "session")
productCache := cache.NewNamespace(c, "product")

// 相同的键在不同命名空间中是隔离的
userCache.Set(ctx, "123", "Alice", time.Hour)       // 实际键: user:123
sessionCache.Set(ctx, "123", "SessionData", time.Hour)  // 实际键: session:123
productCache.Set(ctx, "123", "ProductInfo", time.Hour)  // 实际键: product:123

// 获取值
userName, _ := userCache.Get(ctx, "123")       // "Alice"
sessionData, _ := sessionCache.Get(ctx, "123") // "SessionData"
productInfo, _ := productCache.Get(ctx, "123") // "ProductInfo"
```

更多命名空间功能请参考 [cache 命名空间文档](https://github.com/lianglong/cache/blob/main/NAMESPACE_GUIDE.md)。

## 实际应用场景

### 场景 1：用户会话管理

```go
package main

import (
    "context"
    "time"

    "github.com/lianglong/cache"
    _ "github.com/lianglong/cache-redis"
)

type SessionManager struct {
    cache *cache.Namespace
}

func NewSessionManager(c cache.Cache) *SessionManager {
    return &SessionManager{
        cache: cache.NewNamespace(c, "session"),
    }
}

func (sm *SessionManager) Create(ctx context.Context, sessionID, userID string) error {
    // 存储会话数据
    sm.cache.HSet(ctx, sessionID, "user_id", userID)
    sm.cache.HSet(ctx, sessionID, "created_at", time.Now().Unix())

    // 设置 30 分钟过期
    return sm.cache.Expire(ctx, sessionID, 30*time.Minute)
}

func (sm *SessionManager) Get(ctx context.Context, sessionID string) (map[string]string, error) {
    return sm.cache.HGetAll(ctx, sessionID)
}

func (sm *SessionManager) Extend(ctx context.Context, sessionID string) error {
    // 延长会话时间
    return sm.cache.Expire(ctx, sessionID, 30*time.Minute)
}

func (sm *SessionManager) Destroy(ctx context.Context, sessionID string) error {
    return sm.cache.Delete(ctx, sessionID)
}
```

### 场景 2：限流器

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/lianglong/cache"
    _ "github.com/lianglong/cache-redis"
)

type RateLimiter struct {
    cache *cache.Namespace
}

func NewRateLimiter(c cache.Cache) *RateLimiter {
    return &RateLimiter{
        cache: cache.NewNamespace(c, "ratelimit"),
    }
}

// Allow 检查是否允许请求
// userID: 用户标识
// limit: 时间窗口内的最大请求数
// window: 时间窗口
func (rl *RateLimiter) Allow(ctx context.Context, userID string, limit int64, window time.Duration) (bool, error) {
    key := fmt.Sprintf("user:%s", userID)

    // 递增计数
    count, err := rl.cache.Incr(ctx, key)
    if err != nil {
        return false, err
    }

    // 首次访问，设置过期时间
    if count == 1 {
        rl.cache.Expire(ctx, key, window)
    }

    // 检查是否超过限制
    return count <= limit, nil
}

// Example usage:
// allowed, _ := limiter.Allow(ctx, "user123", 100, time.Minute)
// if !allowed {
//     return errors.New("rate limit exceeded")
// }
```

### 场景 3：分布式锁

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/lianglong/cache"
    _ "github.com/lianglong/cache-redis"
)

type DistributedLock struct {
    cache *cache.Namespace
}

func NewDistributedLock(c cache.Cache) *DistributedLock {
    return &DistributedLock{
        cache: cache.NewNamespace(c, "lock"),
    }
}

// Acquire 获取锁
func (dl *DistributedLock) Acquire(ctx context.Context, resource string, ttl time.Duration) (bool, error) {
    lockKey := fmt.Sprintf("resource:%s", resource)
    return dl.cache.SetNX(ctx, lockKey, "locked", ttl)
}

// Release 释放锁
func (dl *DistributedLock) Release(ctx context.Context, resource string) error {
    lockKey := fmt.Sprintf("resource:%s", resource)
    return dl.cache.Delete(ctx, lockKey)
}

// WithLock 使用锁执行函数
func (dl *DistributedLock) WithLock(ctx context.Context, resource string, ttl time.Duration, fn func() error) error {
    acquired, err := dl.Acquire(ctx, resource, ttl)
    if err != nil {
        return err
    }
    if !acquired {
        return fmt.Errorf("failed to acquire lock for resource: %s", resource)
    }

    defer dl.Release(ctx, resource)
    return fn()
}

// Example usage:
// lock := NewDistributedLock(c)
// err := lock.WithLock(ctx, "order:123", 10*time.Second, func() error {
//     // 处理订单的临界区代码
//     return processOrder("123")
// })
```

### 场景 4：缓存穿透防护

```go
package main

import (
    "context"
    "time"

    "github.com/lianglong/cache"
    _ "github.com/lianglong/cache-redis"
)

type DataCache struct {
    cache *cache.Namespace
}

func NewDataCache(c cache.Cache) *DataCache {
    return &DataCache{
        cache: cache.NewNamespace(c, "data"),
    }
}

// GetWithFallback 获取数据，如果不存在则从数据库加载
func (dc *DataCache) GetWithFallback(ctx context.Context, key string, loader func() (string, error)) (string, error) {
    // 先从缓存获取
    value, err := dc.cache.Get(ctx, key)
    if err == nil {
        return value, nil
    }

    if !cache.IsNotFound(err) {
        return "", err
    }

    // 缓存未命中，从数据库加载
    value, err = loader()
    if err != nil {
        // 如果数据库也没有，缓存一个空标记（防止缓存穿透）
        dc.cache.Set(ctx, key, "__NULL__", 5*time.Minute)
        return "", err
    }

    // 缓存数据
    dc.cache.Set(ctx, key, value, time.Hour)
    return value, nil
}

// Example usage:
// dataCache := NewDataCache(c)
// value, err := dataCache.GetWithFallback(ctx, "user:123", func() (string, error) {
//     return db.GetUserByID("123")
// })
```

## 错误处理

cache-redis 使用统一的错误类型：

```go
value, err := c.Get(ctx, "non-existent-key")
if err != nil {
    if cache.IsNotFound(err) {
        // 键不存在
        fmt.Println("Key not found")
    } else if cache.IsTimeout(err) {
        // 操作超时
        fmt.Println("Operation timeout")
    } else {
        // 其他错误
        fmt.Printf("Error: %v\n", err)
    }
}
```

**标准错误类型：**
- `cache.ErrNotFound` - 键不存在
- `cache.ErrTimeout` - 操作超时
- `cache.ErrConnectionLost` - 连接断开
- `cache.ErrInvalidValue` - 无效的值
- `cache.ErrKeyExpired` - 键已过期
- `cache.ErrCacheFull` - 缓存已满

## 性能优化建议

### 1. 使用批量操作

```go
// ❌ 不推荐：循环单次操作
for _, key := range keys {
    c.Get(ctx, key)
}

// ✅ 推荐：使用批量操作
values, _ := c.MGet(ctx, keys...)
```

### 2. 合理设置过期时间

```go
// 热点数据：短过期时间
c.Set(ctx, "trending:news:1", data, 5*time.Minute)

// 冷数据：长过期时间
c.Set(ctx, "archive:2020:data", data, 24*time.Hour)

// 永久数据：不设置过期时间
c.Set(ctx, "config:app", data, 0)
```

### 3. 使用连接池

```go
config := cache.Config{
    Addr: "127.0.0.1:6379",
    Pool: cache.PoolConfig{
        MaxIdleConns:    10,           // 根据并发量调整
        MaxActiveConns:  100,          // 根据 Redis 服务器能力调整
        IdleTimeout:     5 * time.Minute,
        MaxConnLifetime: 1 * time.Hour,
    },
}
```

### 4. 避免大键

```go
// ❌ 不推荐：存储大对象
c.Set(ctx, "large:data", largeObject, time.Hour)

// ✅ 推荐：使用 Hash 分片存储
for i, chunk := range chunks {
    c.HSet(ctx, "large:data", fmt.Sprintf("chunk:%d", i), chunk)
}
```

### 5. 谨慎使用 Keys 命令

```go
// ❌ 不推荐：生产环境使用 Keys（会阻塞 Redis）
keys, _ := c.Keys(ctx, "user:*")

// ✅ 推荐：使用 Set 或特定键名维护索引
c.SAdd(ctx, "index:users", "user:1", "user:2", "user:3")
members, _ := c.SMembers(ctx, "index:users")
```

## 配置说明

### 超时配置

| 配置项 | 说明 | 默认值 | 建议值 |
|-------|------|--------|--------|
| DialTimeout | 连接超时 | 5s | 5-10s |
| ReadTimeout | 读超时 | 3s | 3-5s |
| WriteTimeout | 写超时 | 3s | 3-5s |

### 连接池配置

| 配置项 | 说明 | 默认值 | 建议值 |
|-------|------|--------|--------|
| MaxIdleConns | 最大空闲连接数 | 10 | 10-50 |
| MaxActiveConns | 最大活跃连接数 | 100 | 100-500 |
| IdleTimeout | 空闲连接超时 | 5分钟 | 5-30分钟 |
| MaxConnLifetime | 连接最大生存时间 | 1小时 | 1-24小时 |

### 重试配置

Redis 驱动内置重试机制：
- 最大重试次数：3 次
- 最小重试间隔：8ms
- 最大重试间隔：512ms

## 常见问题

### 1. 连接失败

**问题：** `failed to connect to redis: dial tcp 127.0.0.1:6379: i/o timeout`

**解决方案：**
- 检查 Redis 服务是否启动：`redis-cli ping`
- 检查防火墙和网络连接
- 检查 Redis 配置的 `bind` 地址
- 增加 DialTimeout：`config.DialTimeout = 10 * time.Second`

### 2. 键不存在错误

**问题：** 如何判断键是否存在？

**解决方案：**
```go
value, err := c.Get(ctx, "key")
if cache.IsNotFound(err) {
    // 键不存在
} else if err != nil {
    // 其他错误
} else {
    // 键存在，value 可用
}
```

### 3. 连接池耗尽

**问题：** `redis: connection pool timeout`

**解决方案：**
- 增加 MaxActiveConns：`config.Pool.MaxActiveConns = 200`
- 检查是否有连接泄漏（未调用 Close）
- 增加 PoolTimeout（默认 4s）

### 4. 序列化问题

**问题：** 如何存储结构体？

**解决方案：**
```go
import "encoding/json"

// 存储
data, _ := json.Marshal(user)
c.Set(ctx, "user:1", string(data), time.Hour)

// 读取
value, _ := c.Get(ctx, "user:1")
var user User
json.Unmarshal([]byte(value), &user)
```

## 依赖版本

- Go >= 1.19
- github.com/lianglong/cache >= v1.0.0
- github.com/redis/go-redis/v9 >= v9.0.0

## 测试

```bash
# 运行测试（需要本地 Redis）
go test -v

# 运行测试（带竞态检测）
go test -v -race

# 运行基准测试
go test -bench=. -benchmem
```

**测试前准备：**
```bash
# 启动 Redis（Docker）
docker run -d -p 6379:6379 redis:7-alpine

# 或使用本地 Redis
redis-server
```

## 贡献

欢迎提交 Issue 和 Pull Request！

贡献指南：
1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

MIT License - 查看 [LICENSE](LICENSE) 文件

## 相关项目

- [cache](https://github.com/lianglong/cache) - 通用缓存接口库
- [go-redis](https://github.com/redis/go-redis) - Redis Go 客户端

## 支持

- 📖 [文档](https://pkg.go.dev/github.com/lianglong/cache-redis)
- 🐛 [报告问题](https://github.com/lianglong/cache-redis/issues)
- 💬 [讨论区](https://github.com/lianglong/cache-redis/discussions)

