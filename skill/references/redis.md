# Redis 模块速查

Package：`github.com/7as0nch/gocommon/redis`
框架依赖：**无**（任何 Go 项目可用）

## Config

```go
type Config struct {
    Addr         string        // 形如 "127.0.0.1:6379"；集群/哨兵请直接用 go-redis 自带构造器
    Username     string        // ACL 模式
    Password     string
    DB           int
    PoolSize     int           // 默认 10
    MinIdleConns int
    DialTimeout  time.Duration // 默认 5s
    ReadTimeout  time.Duration // 默认 3s
    WriteTimeout time.Duration // 默认 3s
    PingTimeout  time.Duration // 构造时 Ping 校验，默认 3s
}
```

## 创建客户端

```go
import "github.com/7as0nch/gocommon/redis"

cli, err := redis.NewClient(ctx, redis.Config{Addr: "127.0.0.1:6379"})
if err != nil { return err }
defer cli.Close()
```

或在已有 `*go-redis/v9` 实例上包一层：

```go
cli := redis.NewClientFromRaw(existingRdb)
```

## 常用 API

```go
// KV
val, err := cli.Get(ctx, "k")                  // ErrNil 表示不存在
_ = cli.Set(ctx, "k", "v", time.Hour)
ok, _ := cli.SetNX(ctx, "lock", "1", 30*time.Second)
n, _ := cli.Del(ctx, "a", "b")
n, _ := cli.Exists(ctx, "a", "b")
_, _ = cli.Expire(ctx, "k", time.Hour)

// Hash
_, _ = cli.HSet(ctx, "h", "f1", "v1", "f2", "v2")
v, _ := cli.HGet(ctx, "h", "f1")               // ErrNil 表示不存在
all, _ := cli.HGetAll(ctx, "h")

// Counter
_, _ = cli.Incr(ctx, "counter")
_, _ = cli.IncrBy(ctx, "counter", 10)
```

需要未封装的 API（Pipeline / Pub/Sub / Stream / ZSet / Lua 等）：

```go
rdb := cli.Raw()  // 返回 *go-redis/v9 Client
rdb.ZAdd(ctx, ...)
```

## TokenStore（JWT 主动吊销）

`TokenStore` 是 auth 模块的依赖接口，用于：登出、改密码后挤下线、单设备登录等场景。

```go
type TokenStore interface {
    Save(ctx context.Context, userID int64, jti string, ttl time.Duration) error
    Exists(ctx context.Context, userID int64, jti string) (bool, error)
    Revoke(ctx context.Context, userID int64, jti string) error
    RevokeAll(ctx context.Context, userID int64) error
}
```

Redis 实现：

```go
store := redis.NewRedisTokenStore(cli, "")  // 第二参为 ""，使用默认 keyPattern "user:auth:tokens:%s"
_ = store.Save(ctx, userID, jti, 24*time.Hour)
ok, _ := store.Exists(ctx, userID, jti)
_ = store.Revoke(ctx, userID, jti)
_ = store.RevokeAll(ctx, userID)  // 改密码 / 安全事件
```

实现细节：用 Redis hash 把 userID 当桶，jti 当 field，整桶 TTL 设为最长 token TTL。
JWT 的 exp 仍由签名校验本身把关，所以即使桶里还残留过期 jti 也不会造成安全问题。

## 资源管理

调用方负责持有 `*Client` 并在程序退出时 `cli.Close()`：

```go
rdb, _ := redis.NewClient(ctx, cfg)
defer rdb.Close()
```

把 client 放到全局变量 / 应用结构体 / DI 容器都可以，gocommon 不强制。

## 常见错误

- `redis: addr is required` — Config.Addr 为空
- `redis: ping failed` — 连接失败；检查地址 / 端口 / 防火墙
- `ErrNil` (redis.ErrNil) — key 不存在，业务上一般不视为错误

## 相关常量（Redis key 模式）

`github.com/7as0nch/gocommon/consts` 集中维护：

```go
consts.USER_AUTH_LOCK_KEY            // "user:auth:lock:%s:%s"
consts.USER_AUTH_USER_TOKENS_KEY     // "user:auth:tokens:%s" — 配合 RedisTokenStore
consts.USER_AUTH_SMS_CAPTCHA_KEY     // "user:auth:sms:captcha:%s"
consts.USER_PAY_LOCK_KEY             // "user:pay:lock:%s:%s"
// 详见 consts/redis.go
```
