# AI 模块速查

Package：`github.com/7as0nch/gocommon/ai`
框架依赖：**无**（任何 Go 项目可用）
底层：CloudWeGo Eino

## 核心概念

- **Agent**：对话/任务执行单元，统一接口 `Stream(ctx, req) (<-chan Response, error)`
- **Factory**：工厂 + 适配器注册中心；每种 `AdapterType` 注册一个 `AdapterCreator`
- **AdapterType**：当前支持 `AdapterTypeEino`（adk）、`AdapterTypeDeepAdk`、`AdapterTypeHost`、`AdapterTypeGraph`
- **Model**：底层 LLM 后端，支持 `ark`、`openai`、`deepseek`

## 创建 Agent

```go
import "github.com/7as0nch/gocommon/ai"
import _ "github.com/7as0nch/gocommon/ai/adapter"  // init() 时注册所有适配器（推荐）

factory := ai.NewFactory()

cfg := &ai.AgentConfig{
    Name:        "chatbot",
    Description: "通用对话助手",
    AdapterType: ai.AdapterTypeEino,
    ModelConfig: ai.ModelConfig{
        ModelType: "ark",                  // ark | openai | deepseek
        ModelName: "doubao-pro-32k",
        APIKey:    bc.AI.ArkKey,
        BaseURL:   "https://ark.cn-beijing.volces.com/api/v3",
    },
    MaxIteration: 10,
}

agent, err := factory.Create(ctx, cfg)
if err != nil { return err }
defer agent.Close()
```

## 流式对话

```go
ch, err := agent.Stream(ctx, ai.Request{
    Message: &ai.Message{Role: ai.RoleUser, Content: "你好"},
    History: history,  // 可选：上下文消息
})
if err != nil { return err }

for resp := range ch {
    fmt.Print(resp.Content)
}
```

## Multi-Agent（Host 模式）

主 Agent 调度多个子 Agent：

```go
masterCfg := &ai.AgentConfig{
    AdapterType: ai.AdapterTypeHost,
    Name:        "host",
    IsMaster:    true,
    ModelConfig: ai.ModelConfig{ModelType: "ark", ...},
}
subCfgs := []*ai.AgentConfig{
    {Name: "writer", AdapterType: ai.AdapterTypeEino, ModelConfig: ...},
    {Name: "reviewer", AdapterType: ai.AdapterTypeEino, ModelConfig: ...},
}
agent, _ := factory.CreateWithSubAgents(ctx, masterCfg, subCfgs)
```

## 工具调用（Function Calling）

```go
import "github.com/7as0nch/gocommon/ai/tool"

// 注册自定义工具（实现 Eino tool.InvokableTool 接口）
myTool := tool.NewMyTool(...)
cfg.Tools = []tool.Tool{myTool}

// 内置工具：get_current_time、web_search（DuckDuckGo）
cfg.WithWebSearchAgent = true
```

## 提示词管理

```go
import "github.com/7as0nch/gocommon/ai"

p := ai.NewPromptTemplate("你是 {{.role}}，请用 {{.lang}} 回答。")
rendered, _ := p.Render(map[string]any{"role": "助手", "lang": "中文"})
cfg.SystemPrompt = rendered
```

## 配置持久化

`AgentConfig` 字段已带 GORM tag，可直接落表：

```go
db.AutoMigrate(&ai.AgentConfig{})
db.Create(cfg)
```

`Repository` 接口（CRUD 抽象）：

```go
type Repository interface {
    Save(ctx, cfg) error
    Get(ctx, id) (*AgentConfig, error)
    List(ctx, filter) ([]*AgentConfig, error)
    Delete(ctx, id) error
}
```

业务侧用 GORM 实现接口，注入到 Factory 实现"动态创建 Agent"场景。

## 不要做的事

- ❌ 直接 `import "github.com/sashabaranov/go-openai"` — 用 `ai.ModelConfig{ModelType: "openai"}`
- ❌ 直接 `import "github.com/cloudwego/eino/components/model/openai"` — 通过 Factory + AdapterType 间接使用
- ❌ 在业务代码里手写 system prompt 字符串拼接 — 用 `ai.PromptTemplate`
- ❌ 把 `APIKey` 写到代码里 — 通过 `ModelConfig` 由 Bootstrap 注入

## 已知约束

- Eino 版本：`v0.8.x`，部分高级 API（如 Workflow / Graph）仍在演进
- Stream 通道关闭由 adapter 负责；业务侧不应主动 close
- `AgentConfig.MaxIteration` 默认 10；过大会增加 LLM 调用次数与延迟
