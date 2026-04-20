package redis

import "errors"

// ErrNil 统一的 "key 不存在" 错误；屏蔽掉 go-redis.Nil，
// 业务侧通过 errors.Is(err, redis.ErrNil) 判断。
var ErrNil = errors.New("redis: key not found")
