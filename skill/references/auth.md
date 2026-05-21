# Auth 模块速查

Package：`github.com/7as0nch/gocommon/auth`
子包：`github.com/7as0nch/gocommon/auth/identityprovider`
框架依赖：
- `JwtClaims` / `NewContext` / `FromContext` / `GetUserId` — **框架无关**，任何项目可用
- `auth.Server` / `auth.Client` — **Kratos 专用**中间件；其他框架请在自家中间件里包装等价逻辑（见 quickstart.md 场景 B）
- `identityprovider/*` — **框架无关**

## JWT 中间件（Kratos）

### Server 端

```go
import (
    "github.com/7as0nch/gocommon/auth"
    "github.com/7as0nch/gocommon/middleware"
    "github.com/go-kratos/kratos/v2/middleware/selector"
    "github.com/go-kratos/kratos/v2/transport/http"
    "github.com/golang-jwt/jwt/v5"
)

keyFn := func(t *jwt.Token) (any, error) {
    return []byte(bc.JWT.SigningKey), nil
}

// 白名单：登录/注册接口跳过鉴权
whiteList := map[string]bool{
    "/api.user.UserService/Login":    true,
    "/api.user.UserService/Register": true,
}

opts := []http.ServerOption{
    http.Middleware(
        selector.Server(auth.Server(keyFn, auth.WithClaims(func() jwt.Claims { return &auth.JwtClaims{} }))).
            Match(middleware.NewWhiteListMatcher(whiteList)).Build(),
    ),
}
```

### 在 handler 里读用户

```go
import "github.com/7as0nch/gocommon/auth"

func (s *UserService) Profile(ctx context.Context, req *pb.ProfileReq) (*pb.ProfileResp, error) {
    uid := auth.GetUserId(ctx)
    if uid == 0 {
        return nil, auth.ErrMissingJwtToken
    }
    // ...
}
```

### 签发 token

```go
claims := auth.JwtClaims{
    UserId:    user.ID,
    UserName:  user.Name,
    UserPhone: user.Phone,
    RegisteredClaims: jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
        ID:        utils.NewSnowflake().String(),  // jti
    },
}
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
str, _ := token.SignedString([]byte(bc.JWT.SigningKey))

// 配合 TokenStore（如 RedisTokenStore）记录 jti
_ = store.Save(ctx, user.ID, claims.ID, 24*time.Hour)
```

## IdentityProvider（策略模式）

`Provider` 统一接口：

```go
type Provider interface {
    Type() enums.IdentityType
    PrepareRegister(ctx context.Context, input any) (*RegisterResult, error)
    VerifyLogin(ctx context.Context, input any, user any, auth any) error
}
```

### 账号密码

```go
import "github.com/7as0nch/gocommon/auth/identityprovider"

p := identityprovider.NewAccountProvider("")  // 默认 8-32 位至少一字母一数字；传 ""→默认正则

hash, _ := identityprovider.HashPassword("Abcd1234")
ok := identityprovider.CheckPassword("Abcd1234", hash)
```

### 手机号 + SMS 验证码

```go
p := identityprovider.NewPhoneProvider(/* deps */)
// 配合 utils/sms 发送验证码，存到 redis（用 consts.USER_AUTH_SMS_CAPTCHA_KEY）
```

### QQ / 微信（OAuth）

需要业务侧实现 `Authorizer` 接口（封装"如何调 QQ/微信 API"）：

```go
type QQAuthorizer interface {
    GetOpenID(ctx context.Context, accessToken string) (string, error)
}

identityprovider.SetQQAuthorizer(myQQAuthorizer)
p := identityprovider.NewQQProvider(myQQAuthorizer)
```

`oauth_authorizer_http.go` 提供了基于 HTTP 的默认实现，业务侧也可直接用：

```go
auth := identityprovider.NewHTTPOAuthAuthorizer(/* config */)
```

## 错误码

所有错误都是 `kratos errors.Unauthorized` 类型，HTTP 状态自动 401：

```go
auth.ErrMissingJwtToken
auth.ErrTokenInvalid
auth.ErrTokenExpired
auth.ErrTokenParseFail
auth.ErrUnSupportSigningMethod
auth.ErrWrongContext
auth.ErrNeedTokenProvider
auth.ErrSignToken
auth.ErrGetKey
```

## Context Key

```go
const (
    UserId    // ctx.Value(auth.UserId).(int64)
    UserName  // ctx.Value(auth.UserName).(string)
    UserPhone // ctx.Value(auth.UserPhone).(string)
    ClientID
)
```

便捷读取：

```go
uid := auth.GetUserId(ctx)
claims, ok := auth.FromContext(ctx)
```

## 与 Redis TokenStore 集成

参考 [redis.md](redis.md) 中的 TokenStore 部分。auth 中间件本身不强制要求 TokenStore（默认只校验签名 + exp）；
如果需要"挤下线 / 主动登出 / 改密码即时生效"，把 TokenStore 注入业务的 LoginUsecase 即可。

## 不支持的能力

- GitHub OAuth → 路线图 P1
- Google OAuth → 路线图 P1
- WebAuthn / Passkey → 暂不支持
- RBAC / Casbin 集成 → 业务层做，不在 gocommon 范围内
