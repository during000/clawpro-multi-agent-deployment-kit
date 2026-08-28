//go:build release

package main

import (
	"embed"

	"hatchery/controller"
)

//go:embed scripts/*
var scriptsFS embed.FS

//go:embed config/default_roles.json
var defaultRolesJSON []byte

//go:embed config/clawpro_required_sg_rules.json
var clawproRequiredSGRulesJSON []byte

//go:embed config/plugin_version_map.json
var pluginVersionMapJSON []byte

func loadDefaultRoles() []byte {
	return defaultRolesJSON
}

func loadClawproRequiredSGRules() []byte {
	return clawproRequiredSGRulesJSON
}

func loadPluginVersionMap() []byte {
	return pluginVersionMapJSON
}

func loadScript(name string) (string, error) {
	// 先查内联脚本注册表（用于 Go 侧参数替换后的临时脚本）
	if content, ok := controller.LookupInlineScript(name); ok {
		return content, nil
	}
	data, err := scriptsFS.ReadFile("scripts/" + name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
