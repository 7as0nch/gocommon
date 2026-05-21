# Quick Start — 把 gocommon 接入新项目

gocommon 是**纯工具库**，没有 DI 约束。下面分两种典型场景给出接入方式。

## 通用步骤（任何框架）

### 1. 添加依赖

```bash
go get github.com/7as0nch/gocommon
```

### 2. 加载配置（可选，推荐）

如果项目已经有配置加载方案（viper / kratos config / 手动），跳过这步。
否则用 gocommon 提供的统一入口：

`configs/config.yaml`：

```yaml
data:
  redis:
    addr: 127.0.0.1:6379
oss:
  endpoint: 127.0.0.1:9000
  access_key_id: minioadmin
  secret_access_key: minioadmin
  default_bucket: app
jwt:
  signing_key: change-me
  token_ttl: 24h
```

代码：

```go
import "github.com/7as0nch/gocommon/lib/conf"

type Bootstrap struct {
    Data *Data `yaml:"data"`
    OSS  *OSS  `yaml:"oss"`
    JWT  *JWT  `yaml:"jwt"`
}
type Data struct { Redis *Redis `yaml:"redis"` }
type Redis struct { Addr string `yaml:"addr"` }
// ...

bc := &Bootstrap{}
cleanup, err := conf.Load("./configs", bc)
if err != nil { log.Fatal(err) }
defer cleanup()
```

### 3. 各模块直接 `New`

```go
import (
    "github.com/7as0nch/gocommon/redis"
    "github.com/7as0nch/gocommon/oss"
    "github.com/7as0nch/gocommon/logger"
)

rdb, _   := redis.NewClient(ctx, redis.Config{Addr: bc.Data.Redis.Addr})
ossCli, _ := oss.NewClient(oss.Config{
    Endpoint: bc.OSS.Endpoint, AccessKeyID: bc.OSS.AccessKeyID,
    SecretAccessKey: bc.OSS.SecretAccessKey, DefaultBucket: bc.OSS.DefaultBucket,
})
zl := logger.NewLogger(logger.LoggerConfig{Path:"./logs", FileName:"app.log", Level:"info"})
```

挂到你的应用上下文（全局变量 / 服务结构体字段 / DI 容器 任选）：

```go
type App struct {
    RDB    *redis.Client
    OSS    *oss.Client
    Logger *zap.Logger
}
```

---

## 场景 A：Kratos 项目

JWT 中间件直接用 `auth.Server`：

```go
import (
    "github.com/7as0nch/gocommon/auth"
    "github.com/7as0nch/gocommon/middleware"
    khttp "github.com/go-kratos/kratos/v2/transport/http"
    "github.com/go-kratos/kratos/v2/middleware/selector"
    "github.com/golang-jwt/jwt/v5"
)

keyFn := func(t *jwt.Token) (any, error) { return []byte(bc.JWT.SigningKey), nil }

whiteList := map[string]bool{
    "/api.user.UserService/Login":    true,
    "/api.user.UserService/Register": true,
}

opts := []khttp.ServerOption{
    khttp.Middleware(
        middleware.MiddlewareCors(),
        selector.Server(auth.Server(keyFn, auth.WithClaims(func() jwt.Claims { return &auth.JwtClaims{} }))).
            Match(middleware.NewWhiteListMatcher(whiteList)).
            Build(),
    ),
}
srv := khttp.NewServer(opts...)
```

---

## 场景 B：Gin 项目（或 Echo / Fiber / Chi / net/http）

gocommon 的核心模块（Redis / OSS / Pay / SMS / AI / IdentityProvider）可直接用。
JWT 中间件需要在你的框架里写一层薄薄的 wrapper（10 行以内）。

示例：Gin 的 JWT 中间件：

```go
import (
    "strings"
    "github.com/gin-gonic/gin"
    "github.com/7as0nch/gocommon/auth"
    "github.com/golang-jwt/jwt/v5"
)

func JWTAuth(signingKey []byte) gin.HandlerFunc {
    return func(c *gin.Context) {
        h := c.GetHeader("Authorization")
        parts := strings.SplitN(h, " ", 2)
        if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
            c.AbortWithStatusJSON(401, gin.H{"error": "missing token"})
            return
        }

        parsed, err := jwt.ParseWithClaims(parts[1], &auth.JwtClaims{}, func(t *jwt.Token) (any, error) {
            return signingKey, nil
        })
        if err != nil || !parsed.Valid {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }
        // 写入 context，handler 里用 auth.GetUserId(c.Request.Context()) 读取
        ctx := auth.NewContext(c.Request.Context(), parsed.Claims)
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}

// 用法
r := gin.New()
r.Use(JWTAuth([]byte(bc.JWT.SigningKey)))
r.GET("/me", func(c *gin.Context) {
    uid := auth.GetUserId(c.Request.Context())
    c.JSON(200, gin.H{"uid": uid})
})
```

签发 JWT（业务无关代码）：

```go
claims := auth.JwtClaims{
    UserId: user.ID, UserName: user.Name,
    RegisteredClaims: jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(bc.JWT.TokenTTL)),
        ID:        uuid.New().String(),  // jti
    },
}
tk := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
tokenStr, _ := tk.SignedString([]byte(bc.JWT.SigningKey))
```

CORS 中间件 Gin 用户用 `gin-contrib/cors` 即可；gocommon 的 `middleware.MiddlewareCors` 是 Kratos 专用的，不要在 Gin 里用。

---

## 场景 C：纯 net/http

```go
import (
    "net/http"
    "github.com/7as0nch/gocommon/auth"
    "github.com/golang-jwt/jwt/v5"
)

func jwtMiddleware(next http.Handler, key []byte) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        h := r.Header.Get("Authorization")
        // ... 解析逻辑同 Gin 版本 ...
        ctx := auth.NewContext(r.Context(), parsed.Claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## 配合 Redis TokenStore（任何框架）

无论哪个框架，token 主动吊销逻辑都一样：

```go
store := redis.NewRedisTokenStore(rdb, "")

// 登录成功后保存 jti
_ = store.Save(ctx, user.ID, claims.ID, bc.JWT.TokenTTL)

// 中间件里在解析完 JWT 后再校验 jti
ok, _ := store.Exists(ctx, claims.UserId, claims.ID)
if !ok { /* 已吊销 */ }

// 登出
_ = store.Revoke(ctx, uid, jti)

// 改密码 → 全部吊销
_ = store.RevokeAll(ctx, uid)
```
