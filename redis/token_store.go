package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// TokenStore 抽象"已签发 token"的存储。
// 业务侧实现 redis/内存/DB 等，注入到 auth.AuthRepo。
//
// 语义：以 userID 为一个"桶"，桶内保存当前有效的 jti -> 任意值（一般是时间戳）。
// auth 中间件在校验 JWT 签名/过期之外，再校验 jti 是否存在于桶内，
// 实现：挤下线、主动登出、跨服务同步吊销 等场景。
type TokenStore interface {
	// Save 记录一个新签发的 token。
	Save(ctx context.Context, userID int64, jti string, ttl time.Duration) error
	// Exists 判断 jti 是否仍然有效（未被主动吊销）。
	Exists(ctx context.Context, userID int64, jti string) (bool, error)
	// Revoke 吊销单个 jti（登出、刷新旧 token 等）。
	Revoke(ctx context.Context, userID int64, jti string) error
	// RevokeAll 吊销该用户全部 jti（改密码、安全事件）。
	RevokeAll(ctx context.Context, userID int64) error
}

// RedisTokenStore 基于 Redis hash 的 TokenStore 实现。
// 约定 keyPattern 为形如 "user:auth:tokens:%s" 的模板，
// 使用 userID（字符串化）作为 hash key，jti 作为 field，value 记录签发时间戳（秒）。
// 用 %s 占位符是为了和 consts.USER_AUTH_USER_TOKENS_KEY 对齐，
// 同时兼容游客场景下 key 为 guestDeviceTicket 的情况。
//
// 备注：
//   - 这里没有直接给每个 jti 设置独立 TTL（Redis hash field 不支持 per-field TTL），
//     而是把整桶的过期时间设置为"最长 token TTL"；短期内桶里可能有已过期 field，
//     JWT 自身的 exp 校验会拦住它们，不会造成安全问题。
//   - 若需要严格 per-field TTL，可把每个 jti 独立成一个 key
//     （如 "user:auth:tokens:<uid>:<jti>"），调用方换用 Save/Exists 的实现即可。
type RedisTokenStore struct {
	client     *Client
	keyPattern string
}

// NewRedisTokenStore 创建一个 redis 实现。
// keyPattern 必须包含一个 %s 占位符，默认 "user:auth:tokens:%s"。
func NewRedisTokenStore(client *Client, keyPattern string) *RedisTokenStore {
	if keyPattern == "" {
		keyPattern = "user:auth:tokens:%s"
	}
	return &RedisTokenStore{client: client, keyPattern: keyPattern}
}

func (s *RedisTokenStore) userKey(userID int64) string {
	return fmt.Sprintf(s.keyPattern, strconv.FormatInt(userID, 10))
}

// Save 写入 jti 并尝试把整桶的过期时间抬高到 ttl。
func (s *RedisTokenStore) Save(ctx context.Context, userID int64, jti string, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return nil
	}
	if jti == "" {
		return nil
	}
	key := s.userKey(userID)
	now := strconv.FormatInt(time.Now().Unix(), 10)
	if _, err := s.client.HSet(ctx, key, jti, now); err != nil {
		return err
	}
	if ttl > 0 {
		if _, err := s.client.Expire(ctx, key, ttl); err != nil {
			return err
		}
	}
	return nil
}

// Exists 判断 jti 是否仍在用户桶里。
func (s *RedisTokenStore) Exists(ctx context.Context, userID int64, jti string) (bool, error) {
	if s == nil || s.client == nil {
		// 无 store 视为不启用校验，让上层只认 JWT 签名/过期即可。
		return true, nil
	}
	if jti == "" {
		return false, nil
	}
	return s.client.HExists(ctx, s.userKey(userID), jti)
}

// Revoke 从用户桶中删除指定 jti。
func (s *RedisTokenStore) Revoke(ctx context.Context, userID int64, jti string) error {
	if s == nil || s.client == nil || jti == "" {
		return nil
	}
	_, err := s.client.HDel(ctx, s.userKey(userID), jti)
	return err
}

// RevokeAll 整桶清空，等效于把该用户所有 token 全部踢下线。
func (s *RedisTokenStore) RevokeAll(ctx context.Context, userID int64) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.Del(ctx, s.userKey(userID))
	return err
}
