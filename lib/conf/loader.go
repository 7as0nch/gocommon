// Package conf 提供基于 Kratos config 的统一配置加载入口。
// 设计目标：让所有接入 gocommon 的新项目使用同一套配置加载方式，避免每个项目自研 yaml/env 解析逻辑。
//
// 典型用法（在业务侧 main.go 中）：
//
//	import "github.com/7as0nch/gocommon/lib/conf"
//
//	type Bootstrap struct {
//	    Server  *Server  `yaml:"server"`
//	    Data    *Data    `yaml:"data"`
//	    JWT     *JWT     `yaml:"jwt"`
//	}
//
//	bc := &Bootstrap{}
//	cleanup, err := conf.Load("./configs", bc)
//	defer cleanup()
package conf

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
)

// Source 配置源抽象，便于业务侧扩展（例如远程 Apollo/Nacos）。
type Source = config.Source

// Loader 基于 Kratos config 的薄封装，统一暴露 Load/Watch/Close 三个方法。
type Loader struct {
	cfg config.Config
}

// New 用一组 Source 创建 Loader 并立即 Load。
// 任意一个 Source 出错都会立即返回。
func New(sources ...Source) (*Loader, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("conf: at least one source is required")
	}
	c := config.New(config.WithSource(sources...))
	if err := c.Load(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("conf: load failed: %w", err)
	}
	return &Loader{cfg: c}, nil
}

// Scan 把整份配置反序列化到 v 指向的结构体。
func (l *Loader) Scan(v any) error {
	if l == nil || l.cfg == nil {
		return fmt.Errorf("conf: loader is nil")
	}
	return l.cfg.Scan(v)
}

// Value 读取单个键值，使用 Kratos config Value 接口（支持 Bool/Int/String/Duration/Slice 等）。
func (l *Loader) Value(key string) config.Value {
	return l.cfg.Value(key)
}

// Watch 监听 key 变化，回调里收到新的 config.Value。
func (l *Loader) Watch(key string, fn func(string, config.Value)) error {
	if l == nil || l.cfg == nil {
		return fmt.Errorf("conf: loader is nil")
	}
	return l.cfg.Watch(key, config.Observer(fn))
}

// Close 释放底层 watcher。
func (l *Loader) Close() error {
	if l == nil || l.cfg == nil {
		return nil
	}
	return l.cfg.Close()
}

// Load 便捷函数：从一个目录或单文件加载 YAML/JSON 配置并 Scan 到 v。
// 返回的 cleanup 应在 main 退出时调用。
func Load(path string, v any) (cleanup func() error, err error) {
	l, err := New(file.NewSource(path))
	if err != nil {
		return nil, err
	}
	if err := l.Scan(v); err != nil {
		_ = l.Close()
		return nil, fmt.Errorf("conf: scan failed: %w", err)
	}
	return l.Close, nil
}
