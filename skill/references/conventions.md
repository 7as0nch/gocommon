# 项目级约定

接入 gocommon 的新项目应遵循以下约定，与库的设计保持一致。

## 1. 配置加载

✅ **DO**：用 `lib/conf.Load` + 业务 Bootstrap struct
```go
bc := &Bootstrap{}
cleanup, _ := conf.Load("./configs", bc)
defer cleanup()
```

❌ **DON'T**：
- 不要用 `lib.ReadConfigMap` / `lib.ReadConfigMapDev` / `lib.ReadConfigMapProd`（已 deprecated）
- 不要在库内读环境变量；环境变量由 Kratos config 的 `env.NewSource` 提供
- 不要硬编码配置路径，启动脚本通过参数传 `-conf ./configs`

## 2. 错误处理

✅ **DO**：
- 库间错误：`fmt.Errorf("xxx: %w", err)` 保留链路
- 业务向 HTTP / gRPC 透出：`github.com/go-kratos/kratos/v2/errors`，与 auth 模块对齐
- 用 `errors.Is` / `errors.As` 判断错误类型

❌ **DON'T**：
- 不要 `panic`（除了 main 中无法继续的初始化失败）
- 不要 `log.Fatal`（库代码尤其不行）
- 不要裸 `_ = someCall()` 忽略错误（除非确定可忽略，必须加注释说明）

## 3. Context 贯穿

✅ **DO**：所有 IO / 跨层调用第一个参数是 `context.Context`
❌ **DON'T**：
- 不要使用 `context.TODO()` 在生产代码（仅测试可用）
- 不要在 struct 字段里持有 `context.Context`
- 不要 `context.WithCancel` 后忘记 cancel

## 4. 命名

- 包名：小写、短、单数（`pay` 而非 `payments`）
- 接口：动词 + er（`TokenStore`、`Provider`、`Authorizer`）
- 工厂函数：`NewXxx(cfg Config) (*Xxx, error)`

## 5. 注释

- 所有导出 API 必须有注释；用中文，描述意图与约束
- 不写无意义注释（`// Set sets value`）
- 复杂内部逻辑用 1-2 行说明 _为什么_，不解释 _做了什么_
- Deprecated API 用 `// Deprecated: 改用 xxx` 标记

## 6. 测试

- 单测：`xxx_test.go` 与被测文件同包；表驱动优先
- 集成测试：放 `internal/integration/` 或 `test/integration/`，用 `testcontainers-go` 起真实 Redis / Postgres / MinIO
- 测试不依赖外部 mock 库 → 用 `testify/mock` 或手写
- **不要 mock gocommon 的组件**：直接用真实 Redis（testcontainers）即可，gocommon 的 API 已经是最薄一层

## 7. 提交规范

- 新功能：`feat(redis): add ratelimit module`
- Bug 修复：`fix(auth): correct cors hardcoded path`
- 重构：`refactor(lib): replace ioutil with os`
- 文档：`docs(skill): add quickstart guide`
- 破坏性变更：在提交体里写 `BREAKING CHANGE: xxx`

## 8. 安全

- ✅ 所有密钥（JWT signingKey、阿里云 AK/SK、支付宝私钥）通过 config 注入
- ❌ 不要把密钥写到 git 里；`.env` 加入 `.gitignore`
- ❌ 不要在日志里打印完整 token / 密钥；可打前 8 位 + `***`
- ✅ JWT 必须设 `exp` 与 `jti`；jti 配合 `redis.TokenStore` 支持主动吊销
- ✅ 支付回调必须验签（`pay.ParseTradeNotification` 已内置）

## 9. 性能

- Redis 连接池：业务侧通过 `redis.Config.PoolSize` 调整，默认 10
- OSS 上传 ≥ 5MB 用 `PutObjectStream` 而非 `PutObjectBytes`
- AI Stream 使用 channel 接收，避免一次性 `ReadAll`
- 大对象禁止跨 goroutine 共享 `[]byte`，用 `sync.Pool` 复用

## 10. 不在 gocommon 范围

以下能力不应该加入 gocommon，请在业务层实现：

- 业务 Repo / UseCase / Service（Kratos DDD 分层）
- 业务实体（User、Order 等 GORM model）
- 业务路由 / proto 定义
- 业务级权限模型（RBAC / Casbin）
- 第三方业务平台对接（微信公众号、抖音开放平台等业务相关 API）
