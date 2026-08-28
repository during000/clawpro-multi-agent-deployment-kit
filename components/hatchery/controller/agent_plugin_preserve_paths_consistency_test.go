package controller

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"testing"
)

// preservePathEntry 对应 config/agent_plugin_preserve_paths.json 中的一条声明。
type preservePathEntry struct {
	AgentType string   `json:"agent_type"`
	Plugin    string   `json:"plugin"`
	Reason    string   `json:"reason"`
	Paths     []string `json:"paths"`
}

// preservePathsArrayRe 匹配 shell 脚本中 `declare -a PRESERVE_PATHS=( ... )` 块，
// 用于从 backup_pre_reinstall_hermes.sh 提取实际生效的硬编码路径列表。
var preservePathsArrayRe = regexp.MustCompile(`(?s)declare -a PRESERVE_PATHS=\((.*?)\)`)

// preservePathLineRe 匹配数组内的一条路径（去除引号和注释）。
var preservePathLineRe = regexp.MustCompile(`"([^"]+)"`)

// TestAgentPluginPreservePathsConsistency 校验
// config/agent_plugin_preserve_paths.json（声明性配置）与
// scripts/backup_pre_reinstall_hermes.sh 中硬编码的 PRESERVE_PATHS 数组（实际生效配置）
// 是否保持一致。
//
// 背景：config/agent_plugin_preserve_paths.json 未被任何 Go 代码在运行期读取，
// 纯粹作为"重装后必须保留路径"的声明性文档；脚本内的 PRESERVE_PATHS 才是真正
// 在备份阶段生效的路径清单。两处需要人工同步维护，容易产生配置漂移
// （声明了但脚本没保留 / 脚本保留了但未声明），因此用本测试兜底校验。
//
// 若未来改为 Go 侧动态读取 JSON 并作为脚本参数传入，可删除本测试。
func TestAgentPluginPreservePathsConsistency(t *testing.T) {
	jsonData, err := os.ReadFile("../config/agent_plugin_preserve_paths.json")
	if err != nil {
		t.Fatalf("读取 config/agent_plugin_preserve_paths.json 失败: %v", err)
	}

	var entries []preservePathEntry
	if err := json.Unmarshal(jsonData, &entries); err != nil {
		t.Fatalf("解析 config/agent_plugin_preserve_paths.json 失败: %v", err)
	}

	// 汇总 JSON 中声明的所有路径（按 agent_type 分组，当前仅有 hermes）。
	declaredByAgentType := make(map[string][]string)
	for _, e := range entries {
		if e.AgentType == "" {
			t.Fatalf("agent_plugin_preserve_paths.json 存在缺少 agent_type 的条目: %+v", e)
		}
		if len(e.Paths) == 0 {
			t.Fatalf("agent_plugin_preserve_paths.json 中 plugin=%q 未声明任何 paths", e.Plugin)
		}
		declaredByAgentType[e.AgentType] = append(declaredByAgentType[e.AgentType], e.Paths...)
	}

	declaredHermesPaths := declaredByAgentType["hermes"]
	if len(declaredHermesPaths) == 0 {
		t.Fatal("agent_plugin_preserve_paths.json 未声明任何 hermes 路径，请检查配置是否被误删")
	}

	scriptData, err := os.ReadFile("../scripts/backup_pre_reinstall_hermes.sh")
	if err != nil {
		t.Fatalf("读取 scripts/backup_pre_reinstall_hermes.sh 失败: %v", err)
	}

	arrayMatch := preservePathsArrayRe.FindStringSubmatch(string(scriptData))
	if arrayMatch == nil {
		t.Fatal("在 backup_pre_reinstall_hermes.sh 中未找到 `declare -a PRESERVE_PATHS=(...)` 数组，" +
			"若脚本重命名了该变量，请同步更新本测试的匹配正则")
	}

	var scriptPaths []string
	for _, lm := range preservePathLineRe.FindAllStringSubmatch(arrayMatch[1], -1) {
		scriptPaths = append(scriptPaths, lm[1])
	}
	if len(scriptPaths) == 0 {
		t.Fatal("backup_pre_reinstall_hermes.sh 的 PRESERVE_PATHS 数组为空")
	}

	// 双向比较：JSON 声明的每条路径都必须在脚本中出现，脚本中的每条路径也必须在 JSON 中声明。
	sortedDeclared := append([]string{}, declaredHermesPaths...)
	sortedScript := append([]string{}, scriptPaths...)
	sort.Strings(sortedDeclared)
	sort.Strings(sortedScript)

	declaredSet := toSet(sortedDeclared)
	scriptSet := toSet(sortedScript)

	for p := range declaredSet {
		if !scriptSet[p] {
			t.Errorf("config/agent_plugin_preserve_paths.json 声明了路径 %q，"+
				"但 scripts/backup_pre_reinstall_hermes.sh 的 PRESERVE_PATHS 中未包含，"+
				"备份时该路径会被漏掉，请同步更新脚本", p)
		}
	}
	for p := range scriptSet {
		if !declaredSet[p] {
			t.Errorf("scripts/backup_pre_reinstall_hermes.sh 的 PRESERVE_PATHS 包含路径 %q，"+
				"但未在 config/agent_plugin_preserve_paths.json 中声明，"+
				"请补充声明说明该路径归属的插件与保留原因，避免文档与实现脱节", p)
		}
	}
}

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}
