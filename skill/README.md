# gocommon skill — 给 AI 编程助手用的接入说明书

本目录是一个 **Claude Code skill**，让其他 AI（在新项目里）能够快速理解并正确接入 `gocommon` 公共组件库。

## 它是什么

- `SKILL.md` 是 skill 的入口文件，带 frontmatter，描述触发场景和关键规则。
- `references/*.md` 是各模块的速查与最小可用示例，按需引用。

文件结构：

```
skill/
├── SKILL.md           # 入口；frontmatter 里的 description 决定 AI 何时触发本 skill
├── README.md          # 你正在看的这份
└── references/
    ├── overview.md
    ├── quickstart.md
    ├── ai.md
    ├── auth.md
    ├── redis.md
    ├── oss.md
    ├── pay.md
    ├── sms.md
    ├── logger.md
    ├── middleware.md
    └── conventions.md
```

## 如何在新项目里使用

两条路径（任选）：

### 方式 1：复制到目标项目（最简单）

把整个 `skill/` 目录复制到目标项目的 `.claude/skills/gocommon/`：

```bash
# 在目标项目根目录
mkdir -p .claude/skills
cp -r path/to/gocommon/skill .claude/skills/gocommon
```

Claude Code 启动会自动发现并加载该 skill。当用户提到任何与 gocommon 模块相关的需求（JWT、Redis、OSS、支付、短信、AI Agent 等），AI 会自动触发本 skill 并按其中的规则进行。

### 方式 2：作为 Claude Code plugin 发布

如果你希望多个项目共享一份 skill 且统一更新：

1. 在 gocommon 根目录新建 `.claude-plugin/plugin.json`（未来 P1 完成后会自动生成）。
2. 把仓库 push 到一个 Git 服务。
3. 在目标项目里：

```bash
claude plugin install <gocommon repo url>
```

然后在目标项目的 `.claude/settings.local.json` 里启用：

```json
{ "enabledPlugins": { "gocommon": true } }
```

## 验证 skill 是否生效

在目标项目里启动 Claude Code，问：

> 如何接入 gocommon 的 JWT + Redis TokenStore？

如果 skill 生效，AI 会基于 `SKILL.md` 和 `references/auth.md` + `references/redis.md` 给出 3-5 行可运行代码，且 import path 正确指向 `github.com/7as0nch/gocommon/...`。

如果 AI 给出了不正确的 import 路径或自行造轮子，请确认 skill 是否真的被加载（运行 `/skills` 命令查看）。

## 维护原则

1. **跟随 gocommon API 演进**。每次 gocommon 暴露的公共 API 变化，必须同步更新 `references/`。
2. **代码片段必须可编译**。所有示例都应基于当前最新的 gocommon API；建议 CI 加一个 `skill_test` 编译标签来验证。
3. **简短优先**。AI 不读太长的文档；每个 reference 单文件控制在 200 行以内，超过则拆分。
4. **不要在 skill 里写业务逻辑**。skill 只描述"如何接入"，不描述"如何做业务"。
