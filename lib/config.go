// 历史的配置加载方式：开发环境 YAML ConfigMap，生产环境 key=value 文本。
// 该实现存在包级全局变量、panic、ioutil 等问题，不适合作为公共库被多项目复用。
//
// Deprecated: 新项目请改用 github.com/7as0nch/gocommon/lib/conf，基于 Kratos config，
// 支持 YAML/JSON/Env/Apollo/Nacos 等多源，且无全局状态。
// 本文件保留 ReadConfigMap / Config 接口的转发实现，以便老项目无痛升级；
// 在下一个 minor 版本会移除。

package lib

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

const (
	dev  = "dev"
	prod = "prod"
)

// Config 旧版配置接口。
//
// Deprecated: 改用 conf.Loader.Scan(&Bootstrap{})。
type Config interface {
	Get(string) string
	GetInt(string) int
	GetBool(string) bool
}

// ConfigMap 旧版配置在内存中的存储结构。
//
// Deprecated: 改用 conf.Load。
type ConfigMap map[string]map[string]string

type metaData struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type configMapDesc struct {
	Kind       string                 `yaml:"kind"`
	ApiVersion string                 `yaml:"apiVersion"`
	MetaData   metaData               `yaml:"metadata"`
	Data       map[string]interface{} `yaml:"data"`
}

// 包级状态用 sync 保护，避免多 goroutine 同时加载时 panic。
// 注意：包级状态本身就是旧设计的缺陷；保留仅为向后兼容。
var (
	legacyMu        sync.Mutex
	legacyDataStore = ConfigMap{}
	legacyFileRead  = map[string]bool{}
	deprecationOnce sync.Once
)

func warnDeprecated(api string) {
	deprecationOnce.Do(func() {
		log.Printf("[gocommon][DEPRECATED] lib.%s 已弃用，请改用 github.com/7as0nch/gocommon/lib/conf（基于 Kratos config）", api)
	})
}

// ReadConfigMap 兼容老调用方：按 IsDev() 走 dev / prod 分支。
//
// Deprecated: 改用 conf.Load。
func ReadConfigMap(filePath string) Config {
	warnDeprecated("ReadConfigMap")
	if IsDev() {
		return ReadConfigMapDev(filePath)
	}
	return ReadConfigMapProd(filePath)
}

// ReadConfigMapDev 读取开发环境的 ConfigMap YAML 文件。
//
// Deprecated: 改用 conf.Load。
func ReadConfigMapDev(filePath string) Config {
	warnDeprecated("ReadConfigMapDev")
	legacyMu.Lock()
	defer legacyMu.Unlock()

	if legacyFileRead[filePath] {
		return &legacyDataStore
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("[gocommon] ReadConfigMapDev open %q failed: %v", filePath, err)
		return &legacyDataStore
	}

	cfg := configMapDesc{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Printf("[gocommon] ReadConfigMapDev unmarshal %q failed: %v", filePath, err)
		return &legacyDataStore
	}

	for key, values := range cfg.Data {
		text, ok := values.(string)
		if !ok {
			continue
		}
		settingsMap := map[string]string{}
		for _, line := range strings.Split(text, "\n") {
			if line == "" {
				continue
			}
			items := strings.SplitN(line, "=", 2)
			if len(items) == 2 {
				settingsMap[items[0]] = items[1]
			}
		}
		legacyDataStore[key] = settingsMap
	}
	legacyFileRead[filePath] = true
	return &legacyDataStore
}

// ReadConfigMapProd 读取生产环境的 key=value 文本配置。
//
// Deprecated: 改用 conf.Load。
func ReadConfigMapProd(filePath string) Config {
	warnDeprecated("ReadConfigMapProd")
	legacyMu.Lock()
	defer legacyMu.Unlock()

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[gocommon] ReadConfigMapProd open %q failed: %v", filePath, err)
		return &legacyDataStore
	}
	defer file.Close()

	settings := map[string]string{}
	br := bufio.NewReader(file)
	for {
		line, _, e := br.ReadLine()
		if e == io.EOF {
			break
		}
		items := strings.SplitN(string(line), "=", 2)
		if len(items) == 2 {
			settings[items[0]] = items[1]
		}
	}
	legacyDataStore[filepath.Base(filePath)] = settings
	return &legacyDataStore
}

// Get 通过 "module.key" 形式取值；env 后缀按 IsDev() 自动选择。
//
// Deprecated: 改用 conf.Loader.Value(...).String()。
func (c *ConfigMap) Get(key string) string {
	keyArray := strings.SplitN(key, ".", 2)
	if len(keyArray) < 2 {
		log.Printf("[gocommon] ConfigMap.Get: key 必须形如 module.key，得到 %q", key)
		return ""
	}
	module, subKey := keyArray[0], keyArray[1]

	suffix := "-prod"
	if IsDev() {
		suffix = "-dev"
	}
	settingsMap := (map[string]map[string]string)(*c)
	settings, ok := settingsMap[module]
	if !ok {
		return ""
	}
	if v, ok := settings[subKey+suffix]; ok {
		return v
	}
	return settings[subKey]
}

// GetInt 取 int；解析失败返回 0 + 日志。
//
// Deprecated: 改用 conf.Loader.Value(...).Int()。
func (c *ConfigMap) GetInt(key string) int {
	v := c.Get(key)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("[gocommon] ConfigMap.GetInt %q: %v", key, err)
		return 0
	}
	return n
}

// GetBool 取 bool；解析失败返回 false + 日志。
//
// Deprecated: 改用 conf.Loader.Value(...).Bool()。
func (c *ConfigMap) GetBool(key string) bool {
	v := c.Get(key)
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Printf("[gocommon] ConfigMap.GetBool %q: %v", key, err)
		return false
	}
	return b
}

// 保留 fmt 引用占位，避免后续若添加错误消息时再引入。
var _ = fmt.Sprintf
