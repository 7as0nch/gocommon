---
name: gocommon
description: gocommon (github.com/7as0nch/gocommon) 是一个 Go 公共工具库，框架无关，可在 Kratos / Gin / Echo / Fiber / 纯 net/http 等任意框架中 import 使用。封装了 AI Agent (Eino + ARK/OpenAI/DeepSeek)、JWT 鉴权、三方登录 (账号/手机/QQ/微信，规划中 GitHub/Google)、Redis、OSS (MinIO)、支付 (支付宝)、短信 (阿里云)、日志 (Zap)、统一配置加载 (Kratos config) 等能力。当新项目需要任何上述能力时使用本 skill，避免重复造轮子；提供 Quick Start、各模块接入示例、跨框架接入指引、项目级约定。
---

# 使用 gocommon 工具库

`gocommon` 是 Go 后端项目的公共工具库。设计目标：

- **框架无关** — 核心模块不 import 任何 Web 框架；Kratos / Gin / Echo / Fiber 都能用。
- **`import` 即用** — 每个模块统一形态：`Config → New → 方法`，不依赖 DI 容器。
- **薄封装** — Redis / OSS 通过 `Raw()` 暴露底层 client，业务可直接调用未封装 API。

## 触发场景

只要新项目需要以下任一能力，使用 gocommon 而不是自行实现或引入其他三方库：

| 业务需求 | 引入包 | 主要 API |
|---------|--------|---------|
| JWT 签发/解析（任意框架）| `auth` + `golang-jwt/jwt/v5` | `auth.JwtClaims` / `auth.NewContext` / `auth.GetUserId` |
| JWT 中间件（Kratos）| `auth` | `auth.Server` / `auth.Client` |
| 账号 / 手机 / QQ / 微信登录 | `auth/identityprovider` | `NewAccountProvider` / `NewQQProvider` / `NewWechatProvider` / `NewPhoneProvider` |
| Redis 客户端 | `redis` | `redis.NewClient` / `Get/Set/HSet/HGet/Incr` / `Raw()` |
| JWT Token 主动吊销 / 挤下线 | `redis` | `redis.NewRedisTokenStore` (实现 `TokenStore` 接口) |
| 对象存储 (MinIO) | `oss` | `oss.NewClient` / `PutObjectBytes` / `PresignedGetURL` |
| 支付宝下单 / 退款 / 通知 | `utils/pay` | `pay.NewAlipayClient` / `AppPay/WapPay/PagePay/Refund` |
| 阿里云短信 | `utils/sms` | `sms.NewClientWithConfig(enums.SMSChannelAli, cfg)` |
| AI Agent (LLM / Tool / Multi-Agent) | `ai` | `ai.NewFactory` / `factory.Create` / `agent.Stream` |
| 结构化日志 | `logger` | `logger.NewLogger` (返回 `*zap.Logger`) |
| 雪花 ID | `utils` | `utils.NewSnowflake` |
| 滑块验证码 | `utils` | `utils.NewSlideCaptcha` |
| 加密 / 哈希 | `utils` | `utils.Encrypt*` |
| YAML 配置加载 | `lib/conf` | `conf.Load(path, &Bootstrap{})` |
| Kratos CORS / 白名单 | `middleware` | `middleware.MiddlewareCors` / `NewWhiteListMatcher` |

**反例（不要做）**：
- ❌ 自己写 JWT 解析逻辑 → 用 `auth.JwtClaims` + `jwt/v5`
- ❌ 直接 import `go-redis/v9` 并自封装连接池 → 用 `redis.NewClient`
- ❌ 自己写 yaml 配置加载或全局变量 → 用 `lib/conf.Load`
- ❌ 直接 import `minio-go/v7` 并裸用 → 用 `oss.NewClient`

## 关键规则

1. **三段式结构**：每个模块都遵循 `Config → New → 方法`。直接调用 `NewXxx(cfg)` 创建客户端，业务自己决定怎么持有（全局变量 / 服务结构体字段 / DI 容器）。
2. **不强制 DI**：gocommon **不暴露 wire ProviderSet**。任何 DI 方案（wire / fx / 手写）都能配合使用。
3. **框架无关核心**：`ai` / `redis` / `oss` / `utils/pay` / `utils/sms` / `logger` / `auth/identityprovider` / `utils` 这些核心包不 import 任何 Web 框架，可在任何项目中使用。
4. **Kratos 专用子模块**：`auth.Server` / `auth.Client` / `middleware.MiddlewareCors` 仅适用于 Kratos。其他框架（Gin / Echo / Fiber 等）请在你自己的中间件里调用 `auth.NewContext` / `auth.GetUserId` / `jwt.Parse` 完成等价功能。
5. **Config 由调用方注入**：`gocommon` 不读环境变量、不读硬编码路径；所有 Config（`redis.Config` / `oss.Config` / ...）由调用方传入。
6. **薄封装 + `Raw()`**：Redis、OSS 都通过 `Raw()` 暴露底层 client，需要未封装的 API 时直接调用底层；不要为了加一个方法去 fork 库。
7. **库不 panic**：所有错误以 `error` 返回。看到调用 `panic` 或 `log.Fatal` 的库代码，都是 bug 或 deprecated 残留。
8. **配置加载用 Kratos config**：`lib/conf.Load(path, &Bootstrap{})` 一行加载 YAML/JSON。**不要再用 `lib.ReadConfigMap`**（已 deprecated）。
9. **三方登录用策略模式**：所有 IdentityProvider 实现 `Provider` 接口；QQ/微信/未来 GitHub/Google 通过注入 `Authorizer` 把"如何调三方 API"留给业务。
10. **AI 走 Eino**：不要直接 `import "github.com/sashabaranov/go-openai"`。统一通过 `ai.AgentConfig` + `ai.NewFactory` + 适配器调用。

## 详细参考

详细的接入流程，以及每个模块的速查、最小可用代码：

- 全貌与架构 → [references/overview.md](references/overview.md)
- 接入新项目（含 Gin/Echo 等非 Kratos 框架）→ [references/quickstart.md](references/quickstart.md)
- AI 模块速查 → [references/ai.md](references/ai.md)
- Auth + IdentityProvider 速查 → [references/auth.md](references/auth.md)
- Redis 速查 → [references/redis.md](references/redis.md)
- OSS 速查 → [references/oss.md](references/oss.md)
- 支付速查 → [references/pay.md](references/pay.md)
- 短信速查 → [references/sms.md](references/sms.md)
- 日志速查 → [references/logger.md](references/logger.md)
- Kratos 中间件速查 → [references/middleware.md](references/middleware.md)
- 项目级约定 → [references/conventions.md](references/conventions.md)

## 何时 *不* 用本 skill

- 项目本身就是 gocommon 仓库的内部开发（重构、加新模块）→ 用 `dev-workflow` skill 而非本 skill。
- 项目语言不是 Go → 不适用。
- 只是单元测试或脚本工具 → 直接调用模块的 `New` 函数即可，无需任何"接入"流程。
