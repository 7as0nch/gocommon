# Middleware 速查

Package：`github.com/7as0nch/gocommon/middleware`

提供项目通用的 Kratos 中间件。当前包含 **CORS** 与 **路由白名单**；未来 P1 会补 recovery / trace / metrics / ratelimit / validate。

## CORS

```go
import "github.com/7as0nch/gocommon/middleware"

// 简易用法：所有路径放开 *
opts := []http.ServerOption{
    http.Middleware(middleware.MiddlewareCors()),
}

// 进阶用法：仅对白名单路径放开
opts := []http.ServerOption{
    http.Middleware(middleware.MiddlewareCors(middleware.CorsOptions{
        AllowedPaths: map[string]bool{
            "/api/upload":  true,
            "/api/webhook": true,
        },
        AllowOrigin:      "https://your-frontend.com",
        AllowMethods:     "GET,POST,OPTIONS",
        AllowCredentials: true,
    })),
}
```

`CorsOptions` 字段：

```go
type CorsOptions struct {
    AllowedPaths     map[string]bool  // 为空时所有路径都放开
    AllowOrigin      string           // 默认 "*"
    AllowMethods     string           // 默认 GET,POST,OPTIONS,PUT,PATCH,DELETE
    AllowHeaders     string           // 默认含 Content-Type/Token/Authorization 等
    AllowCredentials bool             // 默认 true
}
```

## 路由白名单（跳过鉴权）

`NewWhiteListMatcher` 配合 Kratos `selector.Server` 使用：

```go
import (
    "github.com/7as0nch/gocommon/auth"
    "github.com/7as0nch/gocommon/middleware"
    "github.com/go-kratos/kratos/v2/middleware/selector"
)

whiteList := map[string]bool{
    "/api.user.UserService/Login":    true,
    "/api.user.UserService/Register": true,
    "/api.health.HealthService/Ping": true,
}

opts := []http.ServerOption{
    http.Middleware(
        selector.Server(auth.Server(keyFn)).
            Match(middleware.NewWhiteListMatcher(whiteList)).
            Build(),
    ),
}
```

逻辑：命中白名单 → `MatchFunc` 返回 `false` → selector 跳过本中间件。

## 即将到来（P1）

- `middleware.Recovery()` — panic recover + zap 日志
- `middleware.Tracing()` — OpenTelemetry trace
- `middleware.Metrics()` — Prometheus 指标
- `middleware.RateLimit(rdb, opts)` — Redis 令牌桶限流
- `middleware.Validate()` — go-playground/validator/v10 请求参数校验

跟踪进度见 [README.md](../../README.md) 的路线图。
