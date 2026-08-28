package model

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// PluginEntry 单个插件的版本信息
type PluginEntry struct {
	Version     string `json:"version"`
	DisplayName string `json:"display_name"`
	NpmPkg      string `json:"npm_pkg"`
}

// OpenclawVersionEntry 某个 openclaw 版本对应的插件映射表
type OpenclawVersionEntry struct {
	Plugins map[string]PluginEntry `json:"plugins"`
}

// PluginVersionMapJSON 由外部注入（main.go 启动时赋值）。
// release 模式通过 go:embed 嵌入 config/plugin_version_map.json，
// dev 模式从磁盘读取 config/plugin_version_map.json。
var PluginVersionMapJSON []byte

var (
	// pluginVersionMap key: openclaw 版本号，value: 该版本下的插件映射
	pluginVersionMap     map[string]OpenclawVersionEntry
	pluginVersionMapOnce sync.Once
)

// GetPluginVersionMap 懒加载并返回插件版本映射表（openclaw版本 → OpenclawVersionEntry）。
// 进程生命周期内只解析一次，线程安全。
func GetPluginVersionMap() map[string]OpenclawVersionEntry {
	pluginVersionMapOnce.Do(func() {
		data := PluginVersionMapJSON
		if len(data) == 0 {
			slog.Warn("插件版本映射表未注入，跳过加载")
			pluginVersionMap = make(map[string]OpenclawVersionEntry)
			return
		}
		// JSON 中含有 _comment 字段，使用 map[string]json.RawMessage 先解析再过滤
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			slog.Error("解析插件版本映射表失败", "error", err)
			pluginVersionMap = make(map[string]OpenclawVersionEntry)
			return
		}
		pluginVersionMap = make(map[string]OpenclawVersionEntry, len(raw))
		for k, v := range raw {
			if k == "_comment" {
				continue
			}
			var entry OpenclawVersionEntry
			if err := json.Unmarshal(v, &entry); err != nil {
				slog.Warn("解析插件版本映射表条目失败", "openclaw_version", k, "error", err)
				continue
			}
			pluginVersionMap[k] = entry
		}
	})
	return pluginVersionMap
}

// GetPluginsForVersion 根据 openclaw 版本返回对应的插件映射表。
// 若版本不在映射表中，返回 nil。
func GetPluginsForVersion(openclawVersion string) map[string]PluginEntry {
	m := GetPluginVersionMap()
	if entry, ok := m[openclawVersion]; ok {
		return entry.Plugins
	}
	return nil
}

// PluginVersionStatus 单个插件的版本对比结果
type PluginVersionStatus struct {
	Slug             string `json:"slug"`
	DisplayName      string `json:"display_name"`
	InstalledVersion string `json:"installed_version"` // 实例上已安装的版本，空表示未安装
	LatestVersion    string `json:"latest_version"`    // 映射表中的最新版本
	NeedUpgrade      bool   `json:"need_upgrade"`      // 已安装但版本落后
	NotInstalled     bool   `json:"not_installed"`     // 映射表中有但实例未安装
}

// BuildPluginVersionStatus 对比实例已安装插件（pluginVersionsJSON）与 agent 版本对应的映射表，
// 返回每个插件的版本状态列表序列化后的 JSON 字符串。
// pluginVersionsJSON 格式：{"slug": "version", ...}
// agentVersion 为空或映射表中不存在时，返回 "[]"。
func BuildPluginVersionStatus(agentVersion, pluginVersionsJSON string) string {
	expected := GetPluginsForVersion(agentVersion)
	if len(expected) == 0 {
		return "[]"
	}

	// 解析实例已安装的插件版本
	installed := make(map[string]string)
	if pluginVersionsJSON != "" && pluginVersionsJSON != "{}" {
		_ = json.Unmarshal([]byte(pluginVersionsJSON), &installed)
	}

	result := make([]PluginVersionStatus, 0, len(expected))
	for slug, entry := range expected {
		status := PluginVersionStatus{
			Slug:          slug,
			DisplayName:   entry.DisplayName,
			LatestVersion: entry.Version,
		}
		if installedVer, ok := installed[slug]; ok {
			status.InstalledVersion = installedVer
			status.NeedUpgrade = installedVer != entry.Version
		} else {
			status.NotInstalled = true
		}
		result = append(result, status)
	}
	b, err := json.Marshal(result)
	if err != nil {
		return "[]"
	}
	return string(b)
}
