//go:build !release

package main

import (
	"os"

	"hatchery/controller"
)

func loadDefaultRoles() []byte {
	data, err := os.ReadFile("config/default_roles.json")
	if err != nil {
		return []byte{}
	}
	return data
}

func loadClawproRequiredSGRules() []byte {
	data, err := os.ReadFile("config/clawpro_required_sg_rules.json")
	if err != nil {
		return []byte{}
	}
	return data
}

func loadPluginVersionMap() []byte {
	data, err := os.ReadFile("config/plugin_version_map.json")
	if err != nil {
		return []byte{}
	}
	return data
}

func loadScript(name string) (string, error) {
	// 先查内联脚本注册表（用于 Go 侧参数替换后的临时脚本）
	if content, ok := controller.LookupInlineScript(name); ok {
		return content, nil
	}
	data, err := os.ReadFile("scripts/" + name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
