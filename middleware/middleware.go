// Package middleware 提供项目通用的 Kratos 中间件，包含路由白名单与跨域处理。
// 设计原则：白名单与允许跨域路径由调用方通过参数注入，不在库内硬编码。
package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewWhiteListMatcher 构造路径白名单匹配器，命中白名单的路径会跳过该中间件。
// 常用于把鉴权中间件包到 selector.Server(...).Match(NewWhiteListMatcher(...)).Build()。
func NewWhiteListMatcher(whiteList map[string]bool) selector.MatchFunc {
	return func(ctx context.Context, optUrl string) bool {
		if _, ok := whiteList[optUrl]; ok {
			return false
		}
		return true
	}
}

// CorsOptions CORS 中间件配置；零值时使用通配 "*" + 常用方法/头。
type CorsOptions struct {
	// AllowedPaths 仅对这些路径放开 CORS；为 nil 或空时所有路径都放开。
	AllowedPaths map[string]bool
	// AllowOrigin 允许的来源，默认 "*"。
	AllowOrigin string
	// AllowMethods 允许的方法，默认 "GET,POST,OPTIONS,PUT,PATCH,DELETE"。
	AllowMethods string
	// AllowHeaders 允许的请求头，默认包含 Content-Type/Token/Authorization 等常见项。
	AllowHeaders string
	// AllowCredentials 是否允许携带凭证，默认 true。
	AllowCredentials bool
}

// MiddlewareCors 跨域中间件。若 opts.AllowedPaths 非空，则仅对其内路径以及 OPTIONS 预检放行。
// 不传 opts 等价于零值，对所有路径放开通配 CORS。
func MiddlewareCors(opts ...CorsOptions) middleware.Middleware {
	cfg := CorsOptions{
		AllowOrigin:      "*",
		AllowMethods:     "GET,POST,OPTIONS,PUT,PATCH,DELETE",
		AllowHeaders:     "Content-Type,Token,X-Requested-With,Access-Control-Allow-Credentials,User-Agent,Content-Length,Authorization,Accept,Accept-Language,Content-Language,Origin",
		AllowCredentials: true,
	}
	if len(opts) > 0 {
		in := opts[0]
		cfg.AllowedPaths = in.AllowedPaths
		if in.AllowOrigin != "" {
			cfg.AllowOrigin = in.AllowOrigin
		}
		if in.AllowMethods != "" {
			cfg.AllowMethods = in.AllowMethods
		}
		if in.AllowHeaders != "" {
			cfg.AllowHeaders = in.AllowHeaders
		}
		cfg.AllowCredentials = in.AllowCredentials
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if ts, ok := transport.FromServerContext(ctx); ok {
				if ht, ok := ts.(http.Transporter); ok {
					path := ht.RequestHeader().Get(":path")
					method := ht.RequestHeader().Get("x-method")
					isOptions := method == "OPTIONS"

					// 命中白名单或未配置白名单时放行
					pass := len(cfg.AllowedPaths) == 0 ||
						cfg.AllowedPaths[path] ||
						(isOptions && cfg.AllowedPaths[ht.RequestHeader().Get("x-path")])
					if pass {
						h := ht.ReplyHeader()
						h.Set("Access-Control-Allow-Origin", cfg.AllowOrigin)
						h.Set("Access-Control-Allow-Methods", cfg.AllowMethods)
						h.Set("Access-Control-Allow-Headers", cfg.AllowHeaders)
						if cfg.AllowCredentials {
							h.Set("Access-Control-Allow-Credentials", "true")
						}
					}
				}
			}
			return handler(ctx, req)
		}
	}
}
