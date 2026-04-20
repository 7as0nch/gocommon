// Package redis 是对 go-redis/v9 的薄封装。
// 只负责：
//  1. 统一连接/Ping 超时与默认池参数；
//  2. 暴露 Cache、TokenStore 等业务侧常用接口，方便解耦。
//
// 真正的 Redis 调用仍然直接复用 go-redis 的 API，
// Client.Raw() 返回底层 *redis.Client 供业务按需扩展。
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Config 连接配置；未显式设置的字段按 go-redis 默认值。
type Config struct {
	// Addr 形如 "127.0.0.1:6379"；集群/哨兵请另用 go-redis 的对应构造器。
	Addr string
	// Username ACL 模式下使用，可选。
	Username string
	// Password 单密码模式使用，可选。
	Password string
	// DB 数据库编号，默认 0。
	DB int
	// PoolSize 连接池大小，默认 10。
	PoolSize int
	// MinIdleConns 最小空闲连接数。
	MinIdleConns int
	// DialTimeout 建连超时，默认 5s。
	DialTimeout time.Duration
	// ReadTimeout 读超时，默认 3s。
	ReadTimeout time.Duration
	// WriteTimeout 写超时，默认 3s。
	WriteTimeout time.Duration
	// PingTimeout NewClient 内部 Ping 校验使用，默认 3s。
	PingTimeout time.Duration
}

// Client 是对 *redis.Client 的封装。零值不可用，必须通过 NewClient 构造。
type Client struct {
	rdb *goredis.Client
}

// NewClient 根据 Config 创建连接并做一次 Ping，确保链路可用。
// 调用方退出前应 defer c.Close()。
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Addr == "" {
		return nil, errors.New("redis: addr is required")
	}
	opt := &goredis.Options{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
	rdb := goredis.NewClient(opt)

	pingCtx := ctx
	if cfg.PingTimeout > 0 {
		var cancel context.CancelFunc
		pingCtx, cancel = context.WithTimeout(ctx, cfg.PingTimeout)
		defer cancel()
	}
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis: ping failed: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

// NewClientFromRaw 直接用外部已构造的 *redis.Client 包一层，便于共享连接。
func NewClientFromRaw(rdb *goredis.Client) *Client {
	return &Client{rdb: rdb}
}

// Raw 返回底层 *redis.Client，方便业务调用未封装的 API。
func (c *Client) Raw() *goredis.Client {
	if c == nil {
		return nil
	}
	return c.rdb
}

// Close 关闭底层连接。
func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

// ========== 常用 key/value 操作 ==========

// Get 取 string 值；key 不存在时返回 ErrNil。
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	v, err := c.rdb.Get(ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		return "", ErrNil
	}
	return v, err
}

// Set 写 string 值；ttl<=0 时不设过期。
func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// SetNX 仅在 key 不存在时写入，返回是否写入成功。
func (c *Client) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, key, value, ttl).Result()
}

// Del 批量删除 key，返回被删除的 key 数。
func (c *Client) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	return c.rdb.Del(ctx, keys...).Result()
}

// Exists 返回存在的 key 数量。
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	return c.rdb.Exists(ctx, keys...).Result()
}

// Expire 给 key 设置过期时间。
func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return c.rdb.Expire(ctx, key, ttl).Result()
}

// ========== Hash 操作 ==========

// HSet 写 hash field。values 约定为 field1, value1, field2, value2...
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return c.rdb.HSet(ctx, key, values...).Result()
}

// HGet 读 hash field；不存在时返回 ErrNil。
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	v, err := c.rdb.HGet(ctx, key, field).Result()
	if errors.Is(err, goredis.Nil) {
		return "", ErrNil
	}
	return v, err
}

// HDel 删除 hash field。
func (c *Client) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	if len(fields) == 0 {
		return 0, nil
	}
	return c.rdb.HDel(ctx, key, fields...).Result()
}

// HExists 判断 hash field 是否存在。
func (c *Client) HExists(ctx context.Context, key, field string) (bool, error) {
	return c.rdb.HExists(ctx, key, field).Result()
}

// HGetAll 读取整个 hash。
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.rdb.HGetAll(ctx, key).Result()
}

// HLen 返回 hash field 数量。
func (c *Client) HLen(ctx context.Context, key string) (int64, error) {
	return c.rdb.HLen(ctx, key).Result()
}

// ========== 计数/限流 ==========

// Incr 自增 1；如需设置首次过期，通过 SetNX + Incr 组合。
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.rdb.Incr(ctx, key).Result()
}

// IncrBy 按步长自增。
func (c *Client) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	return c.rdb.IncrBy(ctx, key, delta).Result()
}
