# Logger 模块速查

Package：`github.com/7as0nch/gocommon/logger`
框架依赖：**无**（任何 Go 项目可用）
底层：Uber Zap + 自研 lumberjack

## Config

```go
type LoggerConfig struct {
    Path     string  // 日志目录，如 "./logs"
    FileName string  // 主文件名，如 "app.log"
    Level    string  // "debug" | "info" | "warn" | "error"，默认 debug
}
```

## 创建

```go
import "github.com/7as0nch/gocommon/logger"

l := logger.NewLogger(logger.LoggerConfig{
    Path: "./logs", FileName: "app.log", Level: "info",
})
defer l.Sync()
```

特点：
- **同时输出到控制台与文件**
- **自动日期切分**：每天 0 点轮转
- **压缩备份**：旧日志放入 `./logs/backup/` 并 gzip 压缩
- **附带 caller + stacktrace**（warn 及以上自动带堆栈）

## 与 Kratos log.Helper 集成

```go
import (
    kratoszap "github.com/go-kratos/kratos/contrib/log/zap/v2"
    "github.com/go-kratos/kratos/v2/log"
)

zl := logger.NewLogger(cfg)
kl := kratoszap.NewLogger(zl)
helper := log.NewHelper(kl)

helper.Infof("user %d login", uid)
```

## 在 Gin / 其他框架里用

`logger.NewLogger` 返回标准 `*zap.Logger`，任何框架都能直接用：

```go
zl := logger.NewLogger(cfg)
defer zl.Sync()

// Gin：把 zap 当 access log
r := gin.New()
r.Use(ginzap.Ginzap(zl, time.RFC3339, true))   // gin-contrib/zap
```

## 注意事项

- **不要在热路径上 `.With()` 大量字段**：每次 With 都会拷贝字段切片
- **结构化日志**：用 `l.Info("msg", zap.Int64("uid", uid))`，不要 `Sprintf` 拼接
- **traceID 注入**：与 OTel 集成后，通过 ctx 提取 traceID 并 `zap.String("trace_id", tid)`（P1 完成后会有 helper）
