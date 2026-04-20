// Package auth
/* *
 * @Author: chengjiang
 * @Date: 2026-04-20 14:13:57
 * @Description:
**/
// @author chengjiang
// @note  Kratos 服务端/客户端 JWT 中间件，已升级到 jwt/v5。
package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewWhiteListMatcher 白名单；命中白名单的路径不走鉴权。
func NewWhiteListMatcher(whiteList map[string]bool) selector.MatchFunc {
	return func(ctx context.Context, optUrl string) bool {
		if _, ok := whiteList[optUrl]; ok {
			return false
		}
		return true
	}
}

// allowedPaths 需要单独放开跨域的接口白名单。
// 业务侧如果需要扩展，可以改成通过参数注入。
var allowedPaths = map[string]bool{
	"/tracker/batch": true,
}

// MiddlewareCors 对跨域做过滤，仅对 allowedPaths 及其 OPTIONS 预检放行。
func MiddlewareCors() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if ts, ok := transport.FromServerContext(ctx); ok {
				if ht, ok := ts.(http.Transporter); ok {
					path := ht.RequestHeader().Get(":path")
					method := ht.RequestHeader().Get("x-method")
					isOptions := method == "OPTIONS"

					if allowedPaths[path] || (isOptions && allowedPaths[ht.RequestHeader().Get("x-path")]) {
						ht.ReplyHeader().Set("Access-Control-Allow-Origin", "*")
						ht.ReplyHeader().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS,PUT,PATCH,DELETE")
						ht.ReplyHeader().Set("Access-Control-Allow-Credentials", "true")
						ht.ReplyHeader().Set("Access-Control-Allow-Headers", "Content-Type,Token,"+
							"X-Requested-With,Access-Control-Allow-Credentials,User-Agent,Content-Length,Authorization,Accept,Accept-Language,Content-Language,Origin")
					}
				}
			}
			return handler(ctx, req)
		}
	}
}
