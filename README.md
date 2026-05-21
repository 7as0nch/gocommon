# gocommon

> Go 公共工具库 — `import` 即用，**框架无关**，不再重复造轮子。

`gocommon` 把 AI Agent、JWT、三方登录、Redis、OSS、支付、短信等常用能力做了统一封装。每个模块只做一件事：暴露 `Config → New → 方法` 的最小 API，**不绑定任何 Web 框架**。无论你用 Kratos、Gin、Echo、Fiber、Chi 还是纯 `net/http`，都能引入使用。

```go
import "github.com/7as0nch/gocommon/redis"

cli, _ := redis.NewClient(ctx, redis.Config{Addr: "127.0.0.1:6379"})
defer cli.Close()
_ = cli.Set(ctx, "k", "v", time.Hour)
```

---

## 模块矩阵

| 模块 | Import 路径 | 主要能力 | 框架依赖 |
|------|-------------|---------|---------|
| AI | `github.com/7as0nch/gocommon/ai` | Eino Agent 工厂 / ARK / OpenAI / DeepSeek / Multi-Agent | 无 |
| 鉴权核心 | `github.com/7as0nch/gocommon/auth` | JWT 签发 / 解析 / Context 读写 | Kratos errors（仅错误码） |
| 身份提供者 | `github.com/7as0nch/gocommon/auth/identityprovider` | 账号 / 手机 / QQ / 微信 OAuth | 无 |
| Redis | `github.com/7as0nch/gocommon/redis` | KV / Hash / TokenStore (吊销 JWT) | 无 |
| OSS | `github.com/7as0nch/gocommon/oss` | MinIO 上传 / 下载 / 预签 URL | 无 |
| 支付 | `github.com/7as0nch/gocommon/utils/pay` | 支付宝 App/H5/Page Pay / 退款 / 通知 | 无 |
| 短信 | `github.com/7as0nch/gocommon/utils/sms` | 阿里云短信 | 无 |
| 日志 | `github.com/7as0nch/gocommon/logger` | Zap + 自动轮转 | 无 |
| 配置 | `github.com/7as0nch/gocommon/lib/conf` | Kratos config 封装（统一 YAML/JSON 加载） | Kratos config（仅作为 YAML 解析） |
| 中间件 | `github.com/7as0nch/gocommon/middleware` | CORS / 白名单（Kratos 版本） | Kratos middleware |
| 常量 | `github.com/7as0nch/gocommon/consts` | Redis key 模式 | 无 |
| 枚举 | `github.com/7as0nch/gocommon/enums` | PayChannel / SMSChannel / IdentityType | 无 |
| 工具 | `github.com/7as0nch/gocommon/utils` | snowflake / encrypt / safego / 滑块验证码 | 无 |

**核心能力（AI / Redis / OSS / Pay / SMS / Logger / Utils / IdentityProvider）全部框架无关**；
仅 `auth/middleware.go`、`middleware/middleware.go` 是 Kratos 专用，其他框架请自行包装等价中间件（参考 [docs/non-kratos.md](docs/non-kratos.md) 后续补齐）。

Go 版本：`1.25+`。

---

## Quick Start

### 安装

```bash
go get github.com/7as0nch/gocommon
```

### Redis

```go
import "github.com/7as0nch/gocommon/redis"

cli, err := redis.NewClient(ctx, redis.Config{Addr: "127.0.0.1:6379"})
defer cli.Close()
_ = cli.Set(ctx, "key", "value", time.Hour)
val, _ := cli.Get(ctx, "key")

// 需要未封装 API
rdb := cli.Raw()  // *go-redis/v9 Client
```

### JWT（任何框架）

```go
import (
    "github.com/7as0nch/gocommon/auth"
    "github.com/golang-jwt/jwt/v5"
)

// 签发
claims := auth.JwtClaims{
    UserId: 42, UserName: "alice",
    RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour))},
}
tk := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
tokenStr, _ := tk.SignedString([]byte("secret"))

// 解析（在你的框架中间件里）
parsed, _ := jwt.ParseWithClaims(tokenStr, &auth.JwtClaims{}, func(t *jwt.Token) (any, error) {
    return []byte("secret"), nil
})
c := parsed.Claims.(*auth.JwtClaims)
ctx = auth.NewContext(ctx, c)
uid := auth.GetUserId(ctx)
```

Kratos 用户可直接用 `auth.Server(keyFn, ...)`；Gin / Echo / Fiber 用户自行在 handler 前包装一层。

### OSS

```go
import "github.com/7as0nch/gocommon/oss"

cli, _ := oss.NewClient(oss.Config{
    Endpoint: "127.0.0.1:9000", AccessKeyID: "minioadmin", SecretAccessKey: "minioadmin",
    DefaultBucket: "app",
})
_, _ = cli.PutObjectBytes(ctx, "", "file.txt", []byte("hi"), oss.PutObjectOptions{})
url, _ := cli.PresignedGetURL(ctx, "", "file.txt", 10*time.Minute, nil)
```

### 支付宝

```go
import "github.com/7as0nch/gocommon/utils/pay"

cli, _ := pay.NewAlipayClient(pay.AlipayConfig{AppID:"...", PrivateKey:"...", AliPayPublicKey:"...", NotifyURL:"https://..."})
orderStr, _ := cli.AppPay(pay.AppPayOrder{OutTradeNo:"O1", Subject:"VIP", TotalAmount:"9.90"})
```

### 阿里云短信

```go
import (
    "github.com/7as0nch/gocommon/utils/sms"
    "github.com/7as0nch/gocommon/enums"
)

cli, _ := sms.NewClientWithConfig(enums.SMSChannelAli, sms.AliyunConfig{
    AccessKeyID:"...", AccessKeySecret:"...", SignName:"应用名", TemplateCode:"SMS_xxx",
})
_ = cli.Send(ctx, &sms.Request{PhoneNumber:"138...", TemplateParams:`{"code":"1234"}`})
```

### AI Agent

```go
import "github.com/7as0nch/gocommon/ai"

factory := ai.NewFactory()
agent, _ := factory.Create(ctx, &ai.AgentConfig{
    Name:        "chat",
    AdapterType: ai.AdapterTypeEino,
    ModelConfig: ai.ModelConfig{ModelType:"ark", ModelName:"doubao-pro-32k", APIKey: bc.ArkKey},
})
ch, _ := agent.Stream(ctx, ai.Request{Message: &ai.Message{Role: ai.RoleUser, Content: "你好"}})
for resp := range ch { fmt.Print(resp.Content) }
```

---

## 给 AI 的 Skill

仓库自带 [`skill/`](skill/) 目录，是给 Claude Code 等 AI 编程助手看的"接入说明书"。
复制到目标项目的 `.claude/skills/gocommon/`，新项目里的 AI 就会自动遵循本库的接入规范。详见 [skill/README.md](skill/README.md)。

---

## 设计原则

1. **框架无关**。核心模块（AI / Redis / OSS / Pay / SMS / Logger / IdentityProvider）不 import 任何 Web 框架。
2. **薄封装 + `Raw()`**。Redis / OSS 都通过 `Raw()` 暴露底层 client，业务可直接调用未封装 API。
3. **`Config → New → 方法` 三段式**。每个模块的接入方式一致，不引入 DI 框架约束。
4. **库不 panic**。所有错误以 `error` 返回。
5. **配置不内嵌**。所有 Config 由调用方注入；库不读环境变量，不依赖配置中心。
6. **不强制 DI**。删除了早期的 wire ProviderSet——任何 DI 方案（wire / fx / 手写）都能用，库不掺和。

---

## 关于 Kratos / 其他框架

- **Kratos 项目**：可以直接用 `auth.Server` / `middleware.MiddlewareCors`，无需任何额外包装。
- **Gin / Echo / Fiber / Chi 项目**：核心模块直接 `import` 即可使用；中间件需要在你的框架里自行写薄薄一层 wrapper（10 行以内），把 `auth.GetUserId` 等放进自己的 context。

未来路线图考虑提供 `auth/ginmw/`、`middleware/ginmw/` 等子包，按需而非默认引入。

---

## 路线图

- [ ] **db/**：GORM + PostgreSQL（轻量封装，连接池 + zap logger 桥接）
- [ ] **errors/**：统一错误码（标准 error，不强绑 kratos）
- [ ] **observability/**：OTel trace + Prometheus metrics（无框架绑定）
- [ ] **redis/ratelimit.go**：令牌桶限流
- [ ] **auth/identityprovider/**：GitHub / Google OAuth
- [ ] **多框架中间件**：`auth/ginmw/`、`middleware/ginmw/` 等
- [ ] **目录重组**：`utils/pay` → `pay/`，`utils/sms` → `sms/`
- [ ] **工程基线**：Makefile / Dockerfile / docker-compose / golangci.yaml / GitHub Actions

---

## License

MIT — 见 [LICENSE](LICENSE)。
