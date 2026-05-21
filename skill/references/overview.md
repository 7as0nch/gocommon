# gocommon 全貌

## 定位

`gocommon` 是 Go 后端项目的公共工具库。设计目标：

- 新项目通过 `import` 直接复用，**不需要重复造轮子**。
- **框架无关**：核心能力（AI / Redis / OSS / Pay / SMS / Logger / IdentityProvider / Utils）不绑定 Web 框架，Kratos / Gin / Echo / Fiber / 纯 net/http 都能用。
- 所有模块统一形态：`Config → New → 方法`。
- 薄封装哲学：暴露 `Raw()` 让业务直接调用底层 SDK。
- 库不 panic，不读环境变量，所有外部状态由调用方注入。
- **不强制 DI**：不暴露 wire ProviderSet 等约束，使用方自由选择 DI 方案。

## 技术栈

| 类别 | 选型 | 框架绑定 |
|------|------|---------|
| AI 框架 | Eino (CloudWeGo) | 无 |
| LLM 后端 | ARK / OpenAI / DeepSeek | 无 |
| JWT | golang-jwt/v5 | 无 |
| Redis | go-redis/v9 | 无 |
| OSS | minio-go/v7 | 无 |
| 支付 | smartwalle/alipay/v3 | 无 |
| 短信 | alibabacloud-go/dysmsapi | 无 |
| 日志 | uber/zap + lumberjack | 无 |
| 配置加载 | go-kratos/v2/config | 仅当作 YAML/JSON 解析器 |
| Kratos 中间件（可选）| go-kratos/kratos/v2 | 仅 `auth.Server` / `middleware.MiddlewareCors` |
| DI | 无 — 任何方案（wire / fx / 手写）均可 | — |

Go 版本最低：`1.25`。

## 模块依赖关系

```
                   ┌──────────────┐
                   │   business   │  (Kratos / Gin / Echo / 纯 net/http)
                   └──────┬───────┘
                          │ 直接 import + New
            ┌─────────────┼─────────────┐
            │             │             │
       ┌────▼──┐     ┌────▼──┐     ┌────▼───┐
       │  ai   │     │ auth  │     │ redis  │
       └───────┘     └───┬───┘     └────┬───┘
                         │              │
                  ┌──────▼───┐     ┌────▼─────────┐
                  │ identity │     │ TokenStore   │
                  │ provider │     │ (interface)  │
                  └──────────┘     └──────┬───────┘
                                          │
                                  ┌───────▼────────┐
                                  │RedisTokenStore │
                                  └────────────────┘

       ┌───────┐     ┌───────┐     ┌────────┐     ┌────────┐
       │  oss  │     │  pay  │     │  sms   │     │ logger │
       └───────┘     └───────┘     └────────┘     └────────┘

       ┌───────┐
       │ lib   │  → conf (Kratos config 封装) / utils / zip
       └───────┘
```

## 模块清单

每个模块的详细说明见同目录其它文件：

- `ai.md` — Eino Agent 工厂 + 适配器（adk / deepadk / host / graph）+ ModelConfig（ARK / OpenAI / DeepSeek）
- `auth.md` — JWT 中间件 + 多种 IdentityProvider（账号 / 手机 / QQ / 微信）
- `redis.md` — Redis 客户端薄封装 + TokenStore 接口与 Redis 实现
- `oss.md` — MinIO 上传 / 下载 / 预签 URL
- `pay.md` — 支付宝 AppPay / WapPay / PagePay / Refund / Query / 通知
- `sms.md` — 阿里云短信
- `logger.md` — Zap + 按级别分文件 + 自动轮转
- `middleware.md` — CORS / 白名单

## 关键约定

1. **错误处理**：库内统一使用 `error` 返回；标准库错误用 `fmt.Errorf("...: %w", err)` 包装；Kratos 业务错误用 `kratos errors`（如 `auth` 模块）。
2. **Context 贯穿**：所有 IO 方法第一个参数是 `context.Context`，不接受 `nil`。
3. **配置注入**：所有 Config 是值类型 struct；不在库内提供"默认 Config"。业务侧从 Bootstrap 拆解。
4. **资源释放**：`*redis.Client` 等需要释放的对象由调用方持有并在程序退出时 `Close()`；放全局变量、放服务结构体字段、或交给任意 DI 容器均可。
5. **日志**：库内仅在初始化和 deprecated API 触发时打日志；不在热路径上打日志。

## 不在范围内

- 不做 ORM（GORM 集成在 P1 路线图中）。
- 不做 RBAC / 权限模型（业务级，应在业务层做）。
- 不做服务发现 / RPC 路由（用 Kratos 自身的能力）。
- 不做项目脚手架 CLI（gocommon 是纯 library，复制 `skill/` 目录即可让 AI 帮忙生成骨架）。
