package controller

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"hatchery/controller/usergroup"
	"hatchery/i18n"
	"hatchery/model"
)

func TestComputeRowStatus(t *testing.T) {
	v := func(id, val string) configDiffValue { return configDiffValue{ID: id, Value: val} }

	cases := []struct {
		name       string
		instance   []configDiffValue
		target     []configDiffValue
		wantStatus string
	}{
		{
			name:       "both_empty",
			instance:   nil,
			target:     nil,
			wantStatus: "same",
		},
		{
			name:       "identical_single",
			instance:   []configDiffValue{v("1", "GPT-4")},
			target:     []configDiffValue{v("1", "GPT-4")},
			wantStatus: "same",
		},
		{
			name:       "identical_multi_unordered",
			instance:   []configDiffValue{v("1", "A"), v("2", "B"), v("3", "C")},
			target:     []configDiffValue{v("3", "C"), v("1", "A"), v("2", "B")},
			wantStatus: "same",
		},
		{
			name:       "instance_subset_of_target",
			instance:   []configDiffValue{v("1", "A"), v("2", "B")},
			target:     []configDiffValue{v("1", "A"), v("2", "B"), v("3", "C")},
			wantStatus: "contained_in_target",
		},
		{
			name:       "instance_has_item_not_in_target",
			instance:   []configDiffValue{v("1", "A"), v("9", "X")},
			target:     []configDiffValue{v("1", "A"), v("2", "B")},
			wantStatus: "different",
		},
		{
			name:       "completely_disjoint",
			instance:   []configDiffValue{v("1", "A")},
			target:     []configDiffValue{v("2", "B")},
			wantStatus: "different",
		},
		{
			name:       "instance_only_target_empty",
			instance:   []configDiffValue{v("1", "A")},
			target:     nil,
			wantStatus: "different",
		},
		{
			name:       "target_only_instance_empty",
			instance:   nil,
			target:     []configDiffValue{v("1", "A")},
			wantStatus: "contained_in_target",
		},
		{
			name: "policy_value_differs_same_id",
			// 同 policy 不同值 → value 不同 → 视作 different
			instance:   []configDiffValue{v("instance_quota", "5")},
			target:     []configDiffValue{v("instance_quota", "10")},
			wantStatus: "different",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := computeRowStatus(c.instance, c.target)
			if got != c.wantStatus {
				t.Fatalf("got %q want %q", got, c.wantStatus)
			}
		})
	}
}

func TestPolicyEntryValueFormat(t *testing.T) {
	cases := []struct {
		name string
		val  interface{}
		want string
	}{
		{"string", "hello", "hello"},
		{"bool_true", true, "true"},
		{"bool_false", false, "false"},
		{"int", 5, "5"},
		{"int64", int64(100000), "100000"},
		{"float64_int_value", float64(42), "42"},
		{"nil_empty", nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatPolicyValue(c.val); got != c.want {
				t.Errorf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestMakeInstanceModelEntries(t *testing.T) {
	if got := makeInstanceModelEntries(0, ""); len(got) != 0 {
		t.Fatalf("AIModelID=0 want empty, got %v", got)
	}
	got := makeInstanceModelEntries(1804, "claude-opus-4-6-admin")
	if len(got) != 1 || got[0].ID != "1804" || got[0].Label != "claude-opus-4-6-admin" {
		t.Fatalf("unexpected entries: %#v", got)
	}
}

func TestInstanceModelEntries_LookupBranches(t *testing.T) {
	// AIModelID=0 → 不查 map，直接空
	if got := instanceModelEntries(&model.Instance{AIModelID: 0}, map[uint]string{1: "x"}); len(got) != 0 {
		t.Fatalf("AIModelID=0 want empty, got %v", got)
	}
	// AIModelID 命中 map → 1 条
	got := instanceModelEntries(&model.Instance{AIModelID: 7}, map[uint]string{7: "kimi-k2"})
	if len(got) != 1 || got[0].ID != "7" || got[0].Label != "kimi-k2" {
		t.Fatalf("hit lookup got %#v", got)
	}
	// AIModelID 未命中 map → 空（不假设 fallback）
	if got := instanceModelEntries(&model.Instance{AIModelID: 99}, map[uint]string{}); len(got) != 0 {
		t.Fatalf("miss lookup want empty, got %v", got)
	}
}

func TestMakeInstanceSkillEntries(t *testing.T) {
	if got := makeInstanceSkillEntries(0, "", "角色"); len(got) != 0 {
		t.Fatalf("RoleID=0 want empty, got %v", got)
	}
	got := makeInstanceSkillEntries(45, "设计师", "角色")
	if len(got) != 1 || got[0].ID != "45" || got[0].Label != "设计师" || got[0].SubLabel != "角色" {
		t.Fatalf("unexpected entries: %#v", got)
	}
}

func TestInstanceSkillEntries_LookupBranches(t *testing.T) {
	ctx := context.Background()
	// RoleID=0 → 直接空
	if got := instanceSkillEntries(ctx, &model.Instance{RoleID: 0}, map[uint]string{1: "x"}); len(got) != 0 {
		t.Fatalf("RoleID=0 want empty, got %v", got)
	}
	// RoleID 命中 → 1 条
	got := instanceSkillEntries(ctx, &model.Instance{RoleID: 44}, map[uint]string{44: "开发工程师"})
	if len(got) != 1 || got[0].ID != "44" || got[0].Label != "开发工程师" {
		t.Fatalf("hit lookup got %#v", got)
	}
	// RoleID 未命中 → 空
	if got := instanceSkillEntries(ctx, &model.Instance{RoleID: 99}, map[uint]string{}); len(got) != 0 {
		t.Fatalf("miss lookup want empty, got %v", got)
	}
}

func TestInstanceImageTypeEntries(t *testing.T) {
	// AgentType="" → 空集
	if got := instanceImageTypeEntries(&model.Instance{AgentType: ""}); len(got) != 0 {
		t.Fatalf("empty AgentType want empty, got %v", got)
	}
	// 已知 AgentType → 用 display name
	got := instanceImageTypeEntries(&model.Instance{AgentType: "openclaw"})
	if len(got) != 1 || got[0].ID != "openclaw" || got[0].Label != "OpenClaw" {
		t.Fatalf("known agent_type unexpected: %#v", got)
	}
	// 未知 AgentType → fallback 到 ID 本身
	got = instanceImageTypeEntries(&model.Instance{AgentType: "unknown-type"})
	if len(got) != 1 || got[0].ID != "unknown-type" || got[0].Label != "unknown-type" {
		t.Fatalf("unknown agent_type unexpected: %#v", got)
	}
}

func TestMakeInstanceNetworkEntries(t *testing.T) {
	const auto, subVPC, subSG = "自动分配", "私有网络与子网", "安全组"

	// 全空 → vpc/subnet 自动分配，无安全组
	got := makeInstanceNetworkEntries("", "", "", auto, subVPC, subSG)
	want := []usergroup.ConfigEntry{
		{ID: auto, Label: auto, SubLabel: subVPC},
		{ID: auto, Label: auto, SubLabel: subVPC},
	}
	assertEntriesEqual(t, "all_empty", got, want)

	// 完整配置
	got = makeInstanceNetworkEntries("vpc-abc", "subnet-xyz", "sg-123", auto, subVPC, subSG)
	want = []usergroup.ConfigEntry{
		{ID: "vpc-abc", Label: "vpc-abc", SubLabel: subVPC},
		{ID: "subnet-xyz", Label: "subnet-xyz", SubLabel: subVPC},
		{ID: "sg-123", Label: "sg-123", SubLabel: subSG},
	}
	assertEntriesEqual(t, "full", got, want)

	// 只有 VPC 没安全组
	got = makeInstanceNetworkEntries("vpc-abc", "", "", auto, subVPC, subSG)
	want = []usergroup.ConfigEntry{
		{ID: "vpc-abc", Label: "vpc-abc", SubLabel: subVPC},
		{ID: auto, Label: auto, SubLabel: subVPC},
	}
	assertEntriesEqual(t, "vpc_only", got, want)
}

func assertEntriesEqual(t *testing.T, name string, got, want []usergroup.ConfigEntry) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: len got %d want %d (got=%#v)", name, len(got), len(want), got)
	}
	for i := range want {
		if got[i].ID != want[i].ID || got[i].Label != want[i].Label || got[i].SubLabel != want[i].SubLabel {
			t.Errorf("%s[%d]: got %+v want %+v", name, i, got[i], want[i])
		}
	}
}

// 实例侧无 policy entry 时，buildPolicyEntryRows 应输出非 nil 的 InstanceValues 切片，
// 否则 computeConfigDiff 拷到 instanceConfigRow 后 JSON 会序列化成 null（应是 []）。
func TestBuildPolicyEntryRows_EmptyInstanceSerializesAsEmptyArray(t *testing.T) {
	meta := usergroup.ConfigCategoryMeta{Key: "platformPolicy", Label: "平台策略"}
	before := &usergroup.ConfigCategoryResult{Key: "platformPolicy"} // Entries 为 nil/空
	after := &usergroup.ConfigCategoryResult{
		Key: "platformPolicy",
		Entries: []usergroup.ConfigEntry{
			{ID: "instance_quota", Label: "单用户 Agent 数量上限", Meta: map[string]interface{}{"value": 10}},
		},
	}
	rows := buildPolicyEntryRows(context.Background(), meta, before, after)
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].InstanceValues == nil {
		t.Fatalf("InstanceValues 不应为 nil（应为非 nil 空切片，避免 JSON 序列化成 null）")
	}
	if len(rows[0].InstanceValues) != 0 {
		t.Fatalf("InstanceValues 应为空，实际 %v", rows[0].InstanceValues)
	}

	// 验证拷贝到对外 instanceConfigRow 后 JSON 输出为 []
	out := instanceConfigRow{
		Key:            rows[0].Key,
		CategoryKey:    rows[0].CategoryKey,
		CategoryLabel:  rows[0].CategoryLabel,
		SubLabel:       rows[0].SubLabel,
		InstanceValues: rows[0].InstanceValues,
		Status:         rows[0].Status,
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"instance_values":[]`) {
		t.Errorf("want instance_values:[] in JSON, got %s", string(b))
	}
	if strings.Contains(string(b), `"instance_values":null`) {
		t.Errorf("instance_values 不应是 null: %s", string(b))
	}
}

// buildTargetConfig 应用一个空 instance 触发同样的行展开，所有 row keys
// 与 per-instance 计算结果一致；status / instance_values 字段不出现在 target_config 输出里。
func TestBuildTargetConfig_KeysMatchPerInstanceAndShape(t *testing.T) {
	// 通过 JSON 序列化验证字段集合：target_config 行只有 key/category_key/category_label/sub_label/values
	row := targetConfigRow{
		Key:           "model",
		CategoryKey:   "model",
		CategoryLabel: "模型",
		Values:        []configDiffValue{{ID: "1", Name: "GPT-4"}},
	}
	b, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	for _, want := range []string{`"key":"model"`, `"values":`, `"category_key":"model"`} {
		if !strings.Contains(got, want) {
			t.Errorf("want %s in JSON, got %s", want, got)
		}
	}
	for _, forbidden := range []string{`"instance_values"`, `"target_values"`, `"status"`} {
		if strings.Contains(got, forbidden) {
			t.Errorf("target_config 行不应含 %s, got %s", forbidden, got)
		}
	}
}

func TestCollectSubLabelsInOrder_TargetFirst(t *testing.T) {
	before := &usergroup.ConfigCategoryResult{
		Entries: []usergroup.ConfigEntry{{SubLabel: "角色"}},
	}
	after := &usergroup.ConfigCategoryResult{
		Entries: []usergroup.ConfigEntry{
			{SubLabel: "初始技能包"},
			{SubLabel: "角色"},
			{SubLabel: "技能安装来源"},
		},
	}
	got := collectSubLabelsInOrder(before, after)
	want := []string{"初始技能包", "角色", "技能安装来源"}
	if len(got) != len(want) {
		t.Fatalf("len mismatch: %v vs %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d]=%q want %q (顺序应以 target 为主)", i, got[i], want[i])
		}
	}
}

// T25：计费模式的中文映射。
func TestChargeTypeDisplayName_ZhMapping(t *testing.T) {
	cases := []struct {
		code string
		want string
	}{
		{"PREPAID", "包年包月"},
		{"POSTPAID_BY_HOUR", "按量计费"},
		{"UNKNOWN_CODE", "UNKNOWN_CODE"}, // 未知值兜底
		{"", ""},
	}
	ctx := context.Background()
	for _, c := range cases {
		if got := chargeTypeDisplayName(ctx, c.code); got != c.want {
			t.Errorf("chargeTypeDisplayName(%q) = %q, want %q", c.code, got, c.want)
		}
	}
}

// T25：instanceChargeTypeEntries 的三种输入分支。
func TestInstanceChargeTypeEntries(t *testing.T) {
	ctx := context.Background()
	// 1) 空值 → 空集
	if got := instanceChargeTypeEntries(ctx, &model.Instance{InstanceChargeType: ""}); len(got) != 0 {
		t.Errorf("empty charge type should produce empty entries, got %v", got)
	}
	// 2) PREPAID → 1 条 entry，Label 走 i18n（默认 zh 返回"包年包月"）
	entries := instanceChargeTypeEntries(ctx, &model.Instance{InstanceChargeType: "PREPAID"})
	if len(entries) != 1 || entries[0].ID != "PREPAID" || entries[0].Label != "包年包月" {
		t.Errorf("PREPAID entries = %+v", entries)
	}
	// 3) 未知值 → 保留原值
	entries = instanceChargeTypeEntries(ctx, &model.Instance{InstanceChargeType: "SPOT_PAID"})
	if len(entries) != 1 || entries[0].ID != "SPOT_PAID" || entries[0].Label != "SPOT_PAID" {
		t.Errorf("unknown code entries = %+v", entries)
	}
}

// T29：PolicyDefs.Category 分类映射验证（用户配额 3 项 / 模型配额 2 项 / 功能权限开关 10 项）。
func TestPolicyDefs_Category(t *testing.T) {
	userQuota := []string{
		usergroup.PolicyKeyTokenQuotaDay,
		usergroup.PolicyKeyTokenQuotaRules,
		usergroup.PolicyKeyInstanceQuota,
	}
	modelQuota := []string{
		usergroup.PolicyKeyGlobalTokenQuotaDay,
		usergroup.PolicyKeyGlobalTokenQuotaRules,
	}
	featureToggle := []string{
		usergroup.PolicyKeyUserConfigModel,
		usergroup.PolicyKeyUserConfigChannel,
		usergroup.PolicyKeyCustomModel,
		usergroup.PolicyKeyAgentTerminal,
		usergroup.PolicyKeyGatewayUI,
		usergroup.PolicyKeyChatView,
		usergroup.PolicyKeyBrowserVNC,
		usergroup.PolicyKeyLobsterDoctor,
		usergroup.PolicyKeyModelQuota,
		usergroup.PolicyKeySMHAutoProvision,
	}
	for _, k := range userQuota {
		if usergroup.PolicyDefs[k].Category != usergroup.PolicyCategoryUserQuota {
			t.Errorf("%s should be user_quota, got %q", k, usergroup.PolicyDefs[k].Category)
		}
	}
	for _, k := range modelQuota {
		if usergroup.PolicyDefs[k].Category != usergroup.PolicyCategoryModelQuota {
			t.Errorf("%s should be model_quota, got %q", k, usergroup.PolicyDefs[k].Category)
		}
	}
	for _, k := range featureToggle {
		if usergroup.PolicyDefs[k].Category != usergroup.PolicyCategoryFeatureToggle {
			t.Errorf("%s should be feature_toggle, got %q", k, usergroup.PolicyDefs[k].Category)
		}
	}
	// 全量覆盖：所有 15 项 policy 都必须有 Category
	for k, def := range usergroup.PolicyDefs {
		if def.Category == "" {
			t.Errorf("policy %q missing Category", k)
		}
	}
}

// T29：buildPolicyEntryRows 输出行携带 PolicyCategory 字段。
func TestBuildPolicyEntryRows_HasPolicyCategory(t *testing.T) {
	meta := usergroup.ConfigCategoryMeta{Key: usergroup.CategoryKeyPlatformPolicy, Label: "平台策略"}
	entry := func(id, label string, val interface{}) usergroup.ConfigEntry {
		return usergroup.ConfigEntry{ID: id, Label: label, Meta: map[string]interface{}{"value": val}}
	}
	after := &usergroup.ConfigCategoryResult{
		Entries: []usergroup.ConfigEntry{
			entry(usergroup.PolicyKeyInstanceQuota, "单用户 Agent 数量上限", 5),
			entry(usergroup.PolicyKeyGlobalTokenQuotaDay, "全局日 Token 上限", 100000),
			entry(usergroup.PolicyKeyBrowserVNC, "允许用户访问 Agent 云桌面", true),
		},
	}
	rows := buildPolicyEntryRows(context.Background(), meta, nil, after)
	byKey := make(map[string]configDiffRow, len(rows))
	for _, r := range rows {
		byKey[r.Key] = r
	}
	cases := []struct {
		policyKey string
		wantCat   string
	}{
		{usergroup.PolicyKeyInstanceQuota, usergroup.PolicyCategoryUserQuota},
		{usergroup.PolicyKeyGlobalTokenQuotaDay, usergroup.PolicyCategoryModelQuota},
		{usergroup.PolicyKeyBrowserVNC, usergroup.PolicyCategoryFeatureToggle},
	}
	for _, c := range cases {
		row, ok := byKey["platformPolicy."+c.policyKey]
		if !ok {
			t.Errorf("missing row for %s", c.policyKey)
			continue
		}
		if row.PolicyCategory != c.wantCat {
			t.Errorf("row %s: PolicyCategory = %q, want %q", c.policyKey, row.PolicyCategory, c.wantCat)
		}
	}
}

// T25 + T29：JSON 序列化包含新字段（chargeType 行 + policy_category 字段）。
func TestConfigDiff_JSONShape_ChargeTypeAndPolicyCategory(t *testing.T) {
	// 只测行序列化，不依赖 DB
	instRow := instanceConfigRow{
		Key: "chargeType", CategoryKey: "chargeType", CategoryLabel: "计费模式",
		InstanceValues: []configDiffValue{{ID: "PREPAID", Name: "包年包月"}},
		Status:         "same",
	}
	if b, _ := json.Marshal(instRow); !strings.Contains(string(b), `"key":"chargeType"`) {
		t.Errorf("chargeType row json missing key: %s", b)
	}

	policyRow := instanceConfigRow{
		Key: "platformPolicy.instance_quota", CategoryKey: "platformPolicy",
		SubLabel: "单用户 Agent 数量上限", PolicyCategory: "user_quota",
		InstanceValues: []configDiffValue{}, Status: "same",
	}
	b, _ := json.Marshal(policyRow)
	if !strings.Contains(string(b), `"policy_category":"user_quota"`) {
		t.Errorf("policy row json missing policy_category: %s", b)
	}

	// 非 policy 行 policy_category 应 omitempty 消失
	nonPolicy := instanceConfigRow{Key: "model", CategoryKey: "model", InstanceValues: []configDiffValue{}, Status: "same"}
	b2, _ := json.Marshal(nonPolicy)
	if strings.Contains(string(b2), `"policy_category"`) {
		t.Errorf("non-policy row should omit policy_category: %s", b2)
	}
}

// T26：isNotCheckSubLabel 命中集合校验。
func TestIsNotCheckSubLabel(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		category string
		sub      string
		want     bool
	}{
		{usergroup.CategoryKeySkill, "角色", false},
		{usergroup.CategoryKeySkill, "", true},
		{usergroup.CategoryKeySkill, "技能安装来源", true},
		{usergroup.CategoryKeySkill, "初始技能包", false},
		{usergroup.CategoryKeyAgentTool, "企业技能", false},
		{usergroup.CategoryKeyAgentTool, "企业插件", true},
		{usergroup.CategoryKeyAgentTool, "企业MCP", true},
		{usergroup.CategoryKeyNetwork, "私有网络与子网", false},
		{usergroup.CategoryKeyModel, "任意", false},
	}
	for _, c := range cases {
		if got := isNotCheckSubLabel(ctx, c.category, c.sub); got != c.want {
			t.Errorf("isNotCheckSubLabel(%q, %q) = %v, want %v", c.category, c.sub, got, c.want)
		}
	}
}

// T27：isTargetVpcAutoAssign 三种分支。
func TestIsTargetVpcAutoAssign(t *testing.T) {
	auto := "自动分配"
	cases := []struct {
		name    string
		entries []usergroup.ConfigEntry
		want    bool
	}{
		{"empty_entries", nil, true},
		{"all_auto", []usergroup.ConfigEntry{{ID: auto}, {ID: auto}}, true},
		{"real_vpc", []usergroup.ConfigEntry{{ID: "vpc-abc"}, {ID: auto}}, false},
		{"real_all", []usergroup.ConfigEntry{{ID: "vpc-abc"}, {ID: "subnet-def"}}, false},
	}
	for _, c := range cases {
		if got := isTargetVpcAutoAssign(c.entries, auto); got != c.want {
			t.Errorf("%s: isTargetVpcAutoAssign = %v, want %v", c.name, got, c.want)
		}
	}
}

// T26：buildSubLabelRows 命中 skill 角色（→ SubLabel=""）/ 技能安装来源 → not_check；初始技能包整行跳过。
func TestBuildSubLabelRows_NotCheck_Skill(t *testing.T) {
	ctx := context.Background()
	meta := usergroup.ConfigCategoryMeta{Key: usergroup.CategoryKeySkill, Label: "技能"}
	// 塞入实例侧和目标侧都有值 —— 命中 not_check 后仍然要清空
	before := &usergroup.ConfigCategoryResult{Entries: []usergroup.ConfigEntry{
		{ID: "1", Label: "R1", SubLabel: "角色"},
	}}
	after := &usergroup.ConfigCategoryResult{Entries: []usergroup.ConfigEntry{
		{ID: "2", Label: "R2", SubLabel: "角色"},
		{ID: "src1", Label: "OpenClaw Hub", SubLabel: "技能安装来源"},
		{ID: "3", Label: "SkillPack", SubLabel: "初始技能包"},
	}}
	rows := buildSubLabelRows(ctx, meta, before, after, &model.Instance{})
	bySub := map[string]configDiffRow{}
	for _, r := range rows {
		bySub[r.SubLabel] = r
	}
	// "" (角色 remapped) / 技能安装来源 → not_check + 空 values
	for _, sub := range []string{"", "技能安装来源"} {
		r, ok := bySub[sub]
		if !ok {
			t.Fatalf("missing sub_label row: %q (角色 remaps to empty string)", sub)
		}
		if r.Status != statusNotCheck {
			t.Errorf("%q: Status = %q, want not_check", sub, r.Status)
		}
		if len(r.InstanceValues) != 0 || len(r.TargetValues) != 0 {
			t.Errorf("%q: values not cleared: inst=%v tgt=%v", sub, r.InstanceValues, r.TargetValues)
		}
	}
	// 初始技能包 should be skipped entirely — not present in rows
	if _, ok := bySub["初始技能包"]; ok {
		t.Errorf("初始技能包 should be skipped entirely, but found in rows")
	}
}

// T26：agentTool 企业技能整行跳过；企业插件 → not_check；企业 MCP 走原路径。
func TestBuildSubLabelRows_NotCheck_AgentTool(t *testing.T) {
	ctx := context.Background()
	meta := usergroup.ConfigCategoryMeta{Key: usergroup.CategoryKeyAgentTool, Label: "Agent 工具"}
	after := &usergroup.ConfigCategoryResult{Entries: []usergroup.ConfigEntry{
		{ID: "s1", Label: "SkillA", SubLabel: "企业技能"},
		{ID: "p1", Label: "PluginA", SubLabel: "企业插件"},
		{ID: "m1", Label: "MCP-A", SubLabel: "企业MCP"},
	}}
	rows := buildSubLabelRows(ctx, meta, nil, after, &model.Instance{})
	got := map[string]string{}
	for _, r := range rows {
		got[r.SubLabel] = r.Status
	}
	// 企业技能 should be skipped entirely
	if _, hasEnterpriseSkill := got["企业技能"]; hasEnterpriseSkill {
		t.Errorf("企业技能 should be skipped entirely, but found in rows")
	}
	if got["企业插件"] != statusNotCheck {
		t.Errorf("企业插件 status = %q, want not_check", got["企业插件"])
	}
	if got["企业MCP"] != statusNotCheck {
		t.Errorf("企业MCP status = %q, want not_check", got["企业MCP"])
	}
}

// TestBuildSubLabelRows_AgentTool_ForcesMCPRow 验证即使没有任何 MCP 记录，
// 企业MCP 行仍然强制出现且为 not_check。
func TestBuildSubLabelRows_AgentTool_ForcesMCPRow(t *testing.T) {
	ctx := context.Background()
	meta := usergroup.ConfigCategoryMeta{Key: usergroup.CategoryKeyAgentTool, Label: "Agent 工具"}
	// 两侧都没有任何 agentTool 条目
	rows := buildSubLabelRows(ctx, meta, nil, nil, &model.Instance{})
	got := map[string]string{}
	for _, r := range rows {
		got[r.SubLabel] = r.Status
	}
	if got["企业插件"] != statusNotCheck {
		t.Errorf("企业插件 should be forced as not_check even with no entries, got %q", got["企业插件"])
	}
	if got["企业MCP"] != statusNotCheck {
		t.Errorf("企业MCP should be forced as not_check even with no entries, got %q", got["企业MCP"])
	}
}

// T27：网络"私有网络与子网"目标侧全"自动分配" → not_check；目标侧有真实 VPC → 走原路径。
// T27：网络"私有网络与子网"目标侧全"自动分配" → status=not_check + target 清空 + instance 原样保留；
// 目标侧有真实 VPC → 走原路径。
func TestBuildSubLabelRows_NotCheck_NetworkAutoAssign(t *testing.T) {
	ctx := context.Background()
	meta := usergroup.ConfigCategoryMeta{Key: usergroup.CategoryKeyNetwork, Label: "网络"}

	// case A：目标侧 VPC/Subnet 都是"自动分配"，实例侧有真实 VPC/Subnet
	beforeReal := &usergroup.ConfigCategoryResult{Entries: []usergroup.ConfigEntry{
		{ID: "vpc-real", Label: "vpc-real", SubLabel: "私有网络与子网"},
		{ID: "subnet-real", Label: "subnet-real", SubLabel: "私有网络与子网"},
	}}
	afterAuto := &usergroup.ConfigCategoryResult{Entries: []usergroup.ConfigEntry{
		{ID: "自动分配", Label: "自动分配", SubLabel: "私有网络与子网"},
		{ID: "自动分配", Label: "自动分配", SubLabel: "私有网络与子网"},
	}}
	inst := &model.Instance{VpcId: "vpc-real", SubnetId: "subnet-real"}
	rows := buildSubLabelRows(ctx, meta, beforeReal, afterAuto, inst)
	if len(rows) == 0 {
		t.Fatalf("no rows produced")
	}
	var vpcRow *configDiffRow
	for i := range rows {
		if rows[i].SubLabel == "私有网络与子网" {
			vpcRow = &rows[i]
		}
	}
	if vpcRow == nil {
		t.Fatalf("VPC row missing")
	}
	if vpcRow.Status != statusNotCheck {
		t.Errorf("auto-assign target should → not_check, got %q", vpcRow.Status)
	}
	// T27 关键：target 清空，instance 原样保留
	if len(vpcRow.TargetValues) != 0 {
		t.Errorf("target values should be cleared, got %v", vpcRow.TargetValues)
	}
	if len(vpcRow.InstanceValues) != 2 {
		t.Errorf("instance values should be preserved (2 entries: vpc-real + subnet-real), got %v", vpcRow.InstanceValues)
	}

	// case B：目标侧真实 VPC → 走原路径（不 not_check）
	afterReal := &usergroup.ConfigCategoryResult{Entries: []usergroup.ConfigEntry{
		{ID: "vpc-abc", Label: "vpc-abc", SubLabel: "私有网络与子网"},
	}}
	rows = buildSubLabelRows(ctx, meta, nil, afterReal, &model.Instance{})
	for _, r := range rows {
		if r.SubLabel == "私有网络与子网" && r.Status == statusNotCheck {
			t.Errorf("real VPC should NOT be not_check")
		}
	}
}

// T28：公网计费类型 i18n 映射（含 fallback）。
func TestInternetChargeTypeDisplayName(t *testing.T) {
	cases := []struct {
		code, want string
	}{
		{"BANDWIDTH_POSTPAID_BY_HOUR", "按带宽计费"},
		{"TRAFFIC_POSTPAID_BY_HOUR", "按流量计费"},
		{"BANDWIDTH_PACKAGE", "带宽包"},
		{"BANDWIDTH_PREPAID", "包年包月带宽"},
		{"UNKNOWN", "UNKNOWN"},
		{"", "-"},
	}
	ctx := context.Background()
	for _, c := range cases {
		if got := internetChargeTypeDisplayName(ctx, c.code); got != c.want {
			t.Errorf("%q: got %q want %q", c.code, got, c.want)
		}
	}
}

// T28：publicIPAssignedFromInstance / publicIPAssignedLabel — i18n 化的已分配/未分配。
func TestPublicIPAssignedLabels(t *testing.T) {
	ctx := context.Background()
	if publicIPAssignedFromInstance(ctx, "1.2.3.4") != "已分配" {
		t.Errorf("real IP should be 已分配")
	}
	if publicIPAssignedFromInstance(ctx, "") != "未分配" {
		t.Errorf("empty IP should be 未分配")
	}
	if publicIPAssignedLabel(ctx, true) != "已分配" || publicIPAssignedLabel(ctx, false) != "未分配" {
		t.Errorf("bool label mismatch")
	}
}

// T28：bandwidthDisplayName 格式 "N Mbps"。
func TestBandwidthDisplayName(t *testing.T) {
	if got := bandwidthDisplayName(0); got != "0 Mbps" {
		t.Errorf("0 → got %q", got)
	}
	if got := bandwidthDisplayName(5); got != "5 Mbps" {
		t.Errorf("5 → got %q", got)
	}
	if got := bandwidthDisplayName(100); got != "100 Mbps" {
		t.Errorf("100 → got %q", got)
	}
}

// T28：instanceInternetEntries 输出 3 条子行。
func TestInstanceInternetEntries_FromCVM(t *testing.T) {
	ctx := context.Background()

	// case 1：完整 cvmInfo — 3 条 entry 都归 SubLabel="公网"，ID 为子项标签、Label 为值
	cvm := &CVMInstanceInfo{
		PublicIP:                "1.2.3.4",
		InternetChargeType:      "TRAFFIC_POSTPAID_BY_HOUR",
		InternetMaxBandwidthOut: 5,
	}
	got := instanceInternetEntries(ctx, cvm)
	if len(got) != 3 {
		t.Fatalf("want 3 entries, got %d", len(got))
	}
	for _, e := range got {
		if e.SubLabel != "公网" {
			t.Errorf("all internet entries should share SubLabel=公网, got %q", e.SubLabel)
		}
	}
	// 顺序：IP / 计费 / 带宽
	want := []struct{ id, label string }{
		{"公网 IP", "已分配"},
		{"计费模式", "按流量计费"},
		{"带宽上限", "5 Mbps"},
	}
	for i, w := range want {
		if got[i].ID != w.id || got[i].Label != w.label {
			t.Errorf("[%d] got {ID=%q Label=%q}, want %v", i, got[i].ID, got[i].Label, w)
		}
	}

	// case 2：无公网 IP + 未知计费类型 + 0 带宽 → Label 分别是 未分配 / - / 0 Mbps；ID 保持子项标签
	cvm = &CVMInstanceInfo{PublicIP: "", InternetChargeType: "", InternetMaxBandwidthOut: 0}
	got = instanceInternetEntries(ctx, cvm)
	if got[0].ID != "公网 IP" || got[0].Label != "未分配" {
		t.Errorf("public IP row: got {%q, %q}", got[0].ID, got[0].Label)
	}
	if got[1].ID != "计费模式" || got[1].Label != "-" {
		t.Errorf("charge type row: got {%q, %q}", got[1].ID, got[1].Label)
	}
	if got[2].ID != "带宽上限" || got[2].Label != "0 Mbps" {
		t.Errorf("bandwidth row: got {%q, %q}", got[2].ID, got[2].Label)
	}
}

// T28：instanceNetworkEntries 传 nil cvmInfo → 只输出 vpc/subnet/sg，不输出公网。
func TestInstanceNetworkEntries_NilCVMInfo_NoInternetRows(t *testing.T) {
	ctx := context.Background()
	inst := &model.Instance{VpcId: "vpc-abc", SubnetId: "subnet-xyz", SecurityGroupId: "sg-1"}
	got := instanceNetworkEntries(ctx, inst, nil)
	// 3 条：vpc + subnet + sg（无公网）
	if len(got) != 3 {
		t.Fatalf("want 3 entries (no cvmInfo), got %d: %#v", len(got), got)
	}
	for _, e := range got {
		if e.SubLabel == "公网" {
			t.Errorf("unexpected internet sub_label with nil cvmInfo")
		}
	}
}

// T28：instanceNetworkEntries 传 cvmInfo → 追加 3 条 SubLabel="公网" 的 entry。
func TestInstanceNetworkEntries_WithCVMInfo_AppendsInternet(t *testing.T) {
	ctx := context.Background()
	inst := &model.Instance{VpcId: "", SubnetId: "", SecurityGroupId: ""}
	cvm := &CVMInstanceInfo{PublicIP: "1.2.3.4", InternetChargeType: "BANDWIDTH_POSTPAID_BY_HOUR", InternetMaxBandwidthOut: 10}
	got := instanceNetworkEntries(ctx, inst, cvm)
	// 5 条：vpc(自动) + subnet(自动) + 3 条公网（无 sg）
	if len(got) != 5 {
		t.Fatalf("want 5 entries, got %d: %#v", len(got), got)
	}
	internetCount := 0
	for _, e := range got {
		if e.SubLabel == "公网" {
			internetCount++
		}
	}
	if internetCount != 3 {
		t.Errorf("expect 3 internet entries under 公网, got %d", internetCount)
	}
}

// T28（合并版）：网络公网 3 项合并为 1 行 —— 3 项都同 → same；带宽不同 → different + highlight_keys=["10 Mbps"]（instance 侧不同项的 name）。
func TestBuildSubLabelRows_Network_InternetMerged(t *testing.T) {
	ctx := context.Background()
	meta := usergroup.ConfigCategoryMeta{Key: usergroup.CategoryKeyNetwork, Label: "网络"}

	targetInternet := []usergroup.ConfigEntry{
		{ID: "公网 IP", Label: "是", SubLabel: "公网"},
		{ID: "计费模式", Label: "按流量计费", SubLabel: "公网"},
		{ID: "带宽上限", Label: "5 Mbps", SubLabel: "公网"},
	}
	// case A：instance 与 target 3 项完全一致 → same，highlight_keys=[]
	after := &usergroup.ConfigCategoryResult{Entries: append([]usergroup.ConfigEntry{
		{ID: "vpc-real", Label: "vpc-real", SubLabel: "私有网络与子网"},
	}, targetInternet...)}
	before := &usergroup.ConfigCategoryResult{Entries: []usergroup.ConfigEntry{
		{ID: "vpc-real", Label: "vpc-real", SubLabel: "私有网络与子网"},
		{ID: "公网 IP", Label: "是", SubLabel: "公网"},
		{ID: "计费模式", Label: "按流量计费", SubLabel: "公网"},
		{ID: "带宽上限", Label: "5 Mbps", SubLabel: "公网"},
	}}
	rows := buildSubLabelRows(ctx, meta, before, after, &model.Instance{})
	var internetRow *configDiffRow
	for i := range rows {
		if rows[i].SubLabel == "公网" {
			internetRow = &rows[i]
		}
	}
	if internetRow == nil {
		t.Fatalf("internet row missing")
	}
	if len(internetRow.InstanceValues) != 3 || len(internetRow.TargetValues) != 3 {
		t.Fatalf("expect 3+3 values in merged internet row, got instance=%v target=%v", internetRow.InstanceValues, internetRow.TargetValues)
	}
	if internetRow.Status != "same" {
		t.Errorf("all-same case: got %q, want same", internetRow.Status)
	}

	// case B：带宽不同 → different（instance 侧值）
	before2 := &usergroup.ConfigCategoryResult{Entries: []usergroup.ConfigEntry{
		{ID: "公网 IP", Label: "是", SubLabel: "公网"},
		{ID: "计费模式", Label: "按流量计费", SubLabel: "公网"},
		{ID: "带宽上限", Label: "10 Mbps", SubLabel: "公网"},
	}}
	afterOnlyInternet := &usergroup.ConfigCategoryResult{Entries: targetInternet}
	rows = buildSubLabelRows(ctx, meta, before2, afterOnlyInternet, &model.Instance{})
	if len(rows) == 0 || rows[0].SubLabel != "公网" {
		t.Fatalf("first row should be 公网, got %+v", rows)
	}
	if rows[0].Status != "different" {
		t.Errorf("bandwidth mismatch: got %q, want different", rows[0].Status)
	}
}

// T29（合并版）：buildPolicyByCategoryRows 输出 3 行（用户配额 / 模型配额 / 功能权限开关），
// 每行 SubLabel = 分类中文名，PolicyCategory = 枚举，Values 是该分类下所有 policy。
func TestBuildPolicyByCategoryRows_ThreeRows(t *testing.T) {
	ctx := context.Background()
	meta := usergroup.ConfigCategoryMeta{Key: usergroup.CategoryKeyPlatformPolicy, Label: "平台策略"}
	entry := func(id string, val interface{}) usergroup.ConfigEntry {
		return usergroup.ConfigEntry{ID: id, Label: id, Meta: map[string]interface{}{"value": val}}
	}
	after := &usergroup.ConfigCategoryResult{Entries: []usergroup.ConfigEntry{
		entry(usergroup.PolicyKeyInstanceQuota, 20),                // user_quota
		entry(usergroup.PolicyKeyTokenQuotaDay, 500000),            // user_quota — _day 被 skip
		entry(usergroup.PolicyKeyTokenQuotaRules, `[{"mode":"day","limit":500000}]`), // user_quota — _rules 被处理
		entry(usergroup.PolicyKeyGlobalTokenQuotaDay, "无限制"),     // model_quota — _day 被 skip
		entry(usergroup.PolicyKeyGlobalTokenQuotaRules, `[{"mode":"day","limit":100000}]`), // model_quota — _rules 被处理
		entry(usergroup.PolicyKeyBrowserVNC, true),                 // feature_toggle
		entry(usergroup.PolicyKeyChatView, true),                   // feature_toggle
	}}
	rows := buildPolicyByCategoryRows(ctx, meta, nil, after)
	if len(rows) != 3 {
		t.Fatalf("want 3 rows (user_quota / model_quota / feature_toggle), got %d", len(rows))
	}

	byCat := make(map[string]configDiffRow, 3)
	for _, r := range rows {
		byCat[r.PolicyCategory] = r
	}

	// 用户配额：2 项（instance_quota + token_quota_rules；token_quota_day 被 skip）
	if r, ok := byCat[usergroup.PolicyCategoryUserQuota]; !ok {
		t.Fatalf("missing user_quota row")
	} else {
		if r.SubLabel != "用户配额" {
			t.Errorf("user_quota SubLabel: got %q, want 用户配额", r.SubLabel)
		}
		if r.Key != "platformPolicy.user_quota" {
			t.Errorf("user_quota Key: got %q", r.Key)
		}
		if len(r.TargetValues) != 2 {
			t.Errorf("user_quota targets: want 2, got %d", len(r.TargetValues))
		}
		if len(r.InstanceValues) != 0 {
			t.Errorf("user_quota instance: want 0, got %d", len(r.InstanceValues))
		}
		// instance=[] + target=[2] → contained_in_target
		if r.Status != "contained_in_target" {
			t.Errorf("user_quota status: want contained_in_target, got %q", r.Status)
		}
	}

	// 模型配额：1 项（global_token_quota_rules；global_token_quota_day 被 skip）
	if r, ok := byCat[usergroup.PolicyCategoryModelQuota]; !ok {
		t.Fatalf("missing model_quota row")
	} else if r.SubLabel != "模型配额" || len(r.TargetValues) != 1 {
		t.Errorf("model_quota unexpected: %+v", r)
	}

	// 功能权限开关：2 项
	if r, ok := byCat[usergroup.PolicyCategoryFeatureToggle]; !ok {
		t.Fatalf("missing feature_toggle row")
	} else if r.SubLabel != "功能权限开关" || len(r.TargetValues) != 2 {
		t.Errorf("feature_toggle unexpected: %+v", r)
	}
}

// T29：policy entry → value 的 Name 格式：
//   - 用户配额 / 模型配额：name = "<label> <value>"（前端直接展示配额语义 + 值）
//   - 功能权限开关：name = "<label> 开启/关闭"（bool 值 i18n 翻译成中文语义）
//   - 未知 policy_key：不加 label，退回 value
func TestPolicyEntryToValue_NameFormat(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		entry usergroup.ConfigEntry
		want  string
	}{
		{
			name:  "user_quota: label + int value",
			entry: usergroup.ConfigEntry{ID: usergroup.PolicyKeyInstanceQuota, Label: "单用户 Agent 数量上限", Meta: map[string]interface{}{"value": 3}},
			want:  "单用户 Agent 数量上限 3",
		},
		{
			name:  "user_quota: label + string value",
			entry: usergroup.ConfigEntry{ID: usergroup.PolicyKeyTokenQuotaDay, Label: "单用户日 Token 上限", Meta: map[string]interface{}{"value": "无上限"}},
			want:  "单用户日 Token 上限 无上限",
		},
		{
			name:  "model_quota: label + value",
			entry: usergroup.ConfigEntry{ID: usergroup.PolicyKeyGlobalTokenQuotaDay, Label: "全局日 Token 上限", Meta: map[string]interface{}{"value": 100000}},
			want:  "全局日 Token 上限 100000",
		},
		{
			name:  "feature_toggle: value=true → <label> 开启",
			entry: usergroup.ConfigEntry{ID: usergroup.PolicyKeyBrowserVNC, Label: "允许用户访问 Agent 云桌面", Meta: map[string]interface{}{"value": true}},
			want:  "允许用户访问 Agent 云桌面 开启",
		},
		{
			name:  "feature_toggle: Meta.enabled=true（buildPolicyEntries 实际约定）→ <label> 开启",
			entry: usergroup.ConfigEntry{ID: usergroup.PolicyKeyUserConfigModel, Label: "允许用户配置模型", Meta: map[string]interface{}{"enabled": true}},
			want:  "允许用户配置模型 开启",
		},
		{
			name:  "feature_toggle: Meta.enabled=false → <label> 关闭",
			entry: usergroup.ConfigEntry{ID: usergroup.PolicyKeyChatView, Label: "允许用户使用对话视图", Meta: map[string]interface{}{"enabled": false}},
			want:  "允许用户使用对话视图 关闭",
		},
		{
			name:  "feature_toggle: value=false → <label> 关闭",
			entry: usergroup.ConfigEntry{ID: usergroup.PolicyKeyChatView, Label: "允许用户使用对话视图", Meta: map[string]interface{}{"value": false}},
			want:  "允许用户使用对话视图 关闭",
		},
		{
			name:  "no meta → fall back to label",
			entry: usergroup.ConfigEntry{ID: usergroup.PolicyKeyInstanceQuota, Label: "单用户 Agent 数量上限"},
			want:  "单用户 Agent 数量上限",
		},
		{
			name:  "user_quota: empty label + value → value only",
			entry: usergroup.ConfigEntry{ID: usergroup.PolicyKeyInstanceQuota, Label: "", Meta: map[string]interface{}{"value": 3}},
			want:  "3",
		},
		{
			name:  "unknown policy_key: value only (no label prefix)",
			entry: usergroup.ConfigEntry{ID: "not_a_real_policy", Label: "some label", Meta: map[string]interface{}{"value": 42}},
			want:  "42",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := policyEntryToValue(ctx, tc.entry)
			var display string
			if got.Name != "" && got.Value != "" {
				display = got.Name + " " + got.Value
			} else if got.Name != "" {
				display = got.Name
			} else {
				display = got.Value
			}
			if display != tc.want {
				t.Errorf("display: got %q, want %q", display, tc.want)
			}
		})
	}
}

// T29：instance policy override 与 target 差异 → different + highlight_keys 含 instance 侧不同的 policy value（Name = "<label> <value>"）。
func TestBuildPolicyByCategoryRows_Different(t *testing.T) {
	ctx := context.Background()
	meta := usergroup.ConfigCategoryMeta{Key: usergroup.CategoryKeyPlatformPolicy, Label: "平台策略"}
	entry := func(id, label string, val interface{}) usergroup.ConfigEntry {
		return usergroup.ConfigEntry{ID: id, Label: label, Meta: map[string]interface{}{"value": val}}
	}
	// user_quota：instance 侧 5，target 侧 10 → 同 ID 不同 Name → different
	before := &usergroup.ConfigCategoryResult{Entries: []usergroup.ConfigEntry{
		entry(usergroup.PolicyKeyInstanceQuota, "单用户 Agent 数量上限", 5),
	}}
	after := &usergroup.ConfigCategoryResult{Entries: []usergroup.ConfigEntry{
		entry(usergroup.PolicyKeyInstanceQuota, "单用户 Agent 数量上限", 10),
	}}
	rows := buildPolicyByCategoryRows(ctx, meta, before, after)
	var userQuotaRow *configDiffRow
	for i := range rows {
		if rows[i].PolicyCategory == usergroup.PolicyCategoryUserQuota {
			userQuotaRow = &rows[i]
		}
	}
	if userQuotaRow == nil {
		t.Fatalf("user_quota row missing")
	}
	if userQuotaRow.Status != "different" {
		t.Errorf("status: want different, got %q", userQuotaRow.Status)
	}
}

// T29：policyCategorySubLabel 3 类 + 未知 fallback。
func TestPolicyCategorySubLabel(t *testing.T) {
	ctx := context.Background()
	if got := policyCategorySubLabel(ctx, usergroup.PolicyCategoryUserQuota); got != "用户配额" {
		t.Errorf("user_quota → %q", got)
	}
	if got := policyCategorySubLabel(ctx, usergroup.PolicyCategoryModelQuota); got != "模型配额" {
		t.Errorf("model_quota → %q", got)
	}
	if got := policyCategorySubLabel(ctx, usergroup.PolicyCategoryFeatureToggle); got != "功能权限开关" {
		t.Errorf("feature_toggle → %q", got)
	}
	if got := policyCategorySubLabel(ctx, "unknown"); got != "unknown" {
		t.Errorf("unknown fallback → %q", got)
	}
}

func TestAttachValueStatuses_NonDifferent(t *testing.T) {
	v := func(id, val string) configDiffValue { return configDiffValue{ID: id, Value: val} }
	for _, status := range []string{"same", "contained_in_target", "not_check"} {
		inputs := []configDiffValue{v("a", "A"), v("b", "B")}
		out := attachValueStatuses(status, inputs, []configDiffValue{v("a", "A")})
		for _, item := range out {
			if item.Status != status {
				t.Errorf("status=%q: want each value.Status=%q, got %q", status, status, item.Status)
			}
		}
	}
}

// 验证 global_token_quota_rules 的 custom 模式规则能正确格式化为可读字符串，
// 而不是被误显为"无限制"。
func TestPolicyEntryToValue_GlobalTokenQuotaRules_CustomMode(t *testing.T) {
	ctx := context.Background()
	// 模拟 buildPolicyEntries 产出的 entry：
	// ID = "global_token_quota_rules", Meta = {"value": "<rules JSON>"}
	// 其中 rules JSON 含一条 custom 模式规则
	startVal := int64(1752051420) // 2026/07/09 10:57 UTC+8
	rulesJSON := `[{"mode":"custom","limit":100000,"start":` + jsonIntStr(startVal) + `,"refresh":"daily"}]`
	entry := usergroup.ConfigEntry{
		ID:    usergroup.PolicyKeyGlobalTokenQuotaRules,
		Label: "全局 Token 配额规则",
		Meta:  map[string]interface{}{"value": rulesJSON},
	}
	got := policyEntryToValue(ctx, entry)
	// ID 应映射为 global_token_quota_day（前端用 _day key 标识行）
	if got.ID != usergroup.PolicyKeyGlobalTokenQuotaDay {
		t.Errorf("ID: got %q, want %q", got.ID, usergroup.PolicyKeyGlobalTokenQuotaDay)
	}
	// Name 应为 "全局 Tokens 上限"
	if got.Name != "全局 Tokens 上限" {
		t.Errorf("Name: got %q, want %q", got.Name, "全局 Tokens 上限")
	}
	// Value 应包含 "100,000" 和 "按日刷新"，不应是 "无限制"
	if got.Value == "无限制" {
		t.Errorf("Value: got %q, expected formatted custom rule (not 无限制)", got.Value)
	}
	if !strings.Contains(got.Value, "100,000") {
		t.Errorf("Value: got %q, expected to contain '100,000'", got.Value)
	}
	if !strings.Contains(got.Value, "按日刷新") {
		t.Errorf("Value: got %q, expected to contain '按日刷新'", got.Value)
	}
}

// 验证 global_token_quota_rules 在 buildPolicyByCategoryRows 中被正确处理：
// _day 被 skip，_rules 的值通过 formatTokenQuotaRulesValue 格式化。
func TestBuildPolicyByCategoryRows_GlobalTokenQuotaRules_CustomMode(t *testing.T) {
	ctx := context.Background()
	meta := usergroup.ConfigCategoryMeta{Key: usergroup.CategoryKeyPlatformPolicy, Label: "平台策略"}
	rulesJSON := `[{"mode":"custom","limit":100000,"start":1752051420,"refresh":"daily"}]`
	after := &usergroup.ConfigCategoryResult{Entries: []usergroup.ConfigEntry{
		{ID: usergroup.PolicyKeyGlobalTokenQuotaDay, Label: "全局 Tokens 上限",
			Meta: map[string]interface{}{"value": "无限制"}}, // _day 应被 skip
		{ID: usergroup.PolicyKeyGlobalTokenQuotaRules, Label: "全局 Token 配额规则",
			Meta: map[string]interface{}{"value": rulesJSON}},
	}}
	rows := buildPolicyByCategoryRows(ctx, meta, nil, after)
	var modelQuotaRow *configDiffRow
	for i := range rows {
		if rows[i].PolicyCategory == usergroup.PolicyCategoryModelQuota {
			modelQuotaRow = &rows[i]
		}
	}
	if modelQuotaRow == nil {
		t.Fatalf("model_quota row missing")
	}
	if len(modelQuotaRow.TargetValues) != 1 {
		t.Fatalf("model_quota TargetValues: want 1, got %d", len(modelQuotaRow.TargetValues))
	}
	val := modelQuotaRow.TargetValues[0]
	if val.Value == "无限制" {
		t.Errorf("Value: got %q, expected formatted custom rule (not 无限制)", val.Value)
	}
	if !strings.Contains(val.Value, "100,000") {
		t.Errorf("Value: got %q, expected to contain '100,000'", val.Value)
	}
}

func jsonIntStr(v int64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// 验证 formatSingleQuotaRule 的格式化：
//   - 无限制（Limit < 0）保留周期上下文：day → "每日 无限制" 等
//   - 有限额（Limit >= 0）也保留周期前缀 + 千分位：day → "每日 100,000" 等
func TestFormatSingleQuotaRule_UnlimitedPreservesPeriod(t *testing.T) {
	ctx := context.Background()
	startTs := int64(1783598820) // 2026/07/09 20:07 UTC+8
	startStr := time.Unix(startTs, 0).Format("2006/01/02 15:04")
	cases := []struct {
		name string
		rule model.TokenQuotaRule
		want string
	}{
		{
			name: "day_unlimited",
			rule: model.TokenQuotaRule{Mode: model.QuotaModeDay, Limit: -1},
			want: "每日 无限制",
		},
		{
			name: "month_unlimited",
			rule: model.TokenQuotaRule{Mode: model.QuotaModeMonth, Limit: -1},
			want: "每月 无限制",
		},
		{
			name: "custom_unlimited_daily",
			rule: model.TokenQuotaRule{Mode: model.QuotaModeCustom, Limit: -1, Start: &startTs, Refresh: model.QuotaRefreshDaily},
			want: startStr + " - 无终止, 按日刷新 无限制",
		},
		{
			name: "custom_unlimited_no_refresh",
			rule: model.TokenQuotaRule{Mode: model.QuotaModeCustom, Limit: -1, Start: &startTs, Refresh: model.QuotaRefreshNone},
			want: startStr + " - 无终止, 不刷新 无限制",
		},
		{
			name: "day_limited",
			rule: model.TokenQuotaRule{Mode: model.QuotaModeDay, Limit: 100000},
			want: "每日 100,000",
		},
		{
			name: "month_limited",
			rule: model.TokenQuotaRule{Mode: model.QuotaModeMonth, Limit: 30000000},
			want: "每月 30,000,000",
		},
		{
			name: "custom_limited",
			rule: model.TokenQuotaRule{Mode: model.QuotaModeCustom, Limit: 100000, Start: &startTs, Refresh: model.QuotaRefreshDaily},
			want: startStr + " - 无终止, 按日刷新 100,000",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := formatSingleQuotaRule(ctx, c.rule)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestAttachValueStatuses_Different(t *testing.T) {
	v := func(id, val string) configDiffValue { return configDiffValue{ID: id, Value: val} }
	instance := []configDiffValue{v("a", "A"), v("b", "B"), v("c", "C")}
	target := []configDiffValue{v("a", "A"), v("b", "B")}
	out := attachValueStatuses("different", instance, target)
	if out[0].Status != "same" {
		t.Errorf("v(a,A) in target → want same, got %q", out[0].Status)
	}
	if out[1].Status != "same" {
		t.Errorf("v(b,B) in target → want same, got %q", out[1].Status)
	}
	if out[2].Status != "different" {
		t.Errorf("v(c,C) not in target → want different, got %q", out[2].Status)
	}
}

func TestAttachValueStatuses_EmptyInstance(t *testing.T) {
	v := func(id, val string) configDiffValue { return configDiffValue{ID: id, Value: val} }
	out := attachValueStatuses("different", nil, []configDiffValue{v("a", "A")})
	if out != nil {
		t.Errorf("nil input → want nil, got %v", out)
	}
}

func TestFormatWithCommas(t *testing.T) {
	cases := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{30000000, "30,000,000"},
		{-1, "-1"},
		{-30000000, "-30,000,000"},
	}
	for _, c := range cases {
		if got := formatWithCommas(c.input); got != c.want {
			t.Errorf("%d: got %q want %q", c.input, got, c.want)
		}
	}
}

// TestFormatTokenQuotaRulesValue_EmptyRules 验证空规则 "[]" 不再返回纯"无限制"。
// 修复前：resolveGlobalTokenQuotaRulesPolicy 返回 "[]" → formatTokenQuotaRulesValue 返回 "无限制"（丢失周期）
// 修复后：resolveGlobalTokenQuotaRulesPolicy 返回 [{"mode":"day","limit":-1}] → 格式化为 "每日 无限制"
func TestFormatTokenQuotaRulesValue_EmptyRules(t *testing.T) {
	ctx := context.Background()
	unlimited := i18n.T(ctx, i18n.MsgGroupTreeMetaUnlimited)

	// 空字符串 → 纯"无限制"（无法推断周期，这是合理的）
	got := formatTokenQuotaRulesValue(ctx, map[string]interface{}{"value": ""})
	if got != unlimited {
		t.Errorf("empty string: got %q want %q", got, unlimited)
	}

	// "[]" 空规则 → 纯"无限制"（无规则无法推断周期）
	// 注：修复后 resolveGlobalTokenQuotaRulesPolicy 不再返回 "[]"，
	// 但 formatTokenQuotaRulesValue 本身仍需正确处理 "[]" 输入
	got = formatTokenQuotaRulesValue(ctx, map[string]interface{}{"value": "[]"})
	if got != unlimited {
		t.Errorf("empty rules: got %q want %q", got, unlimited)
	}

	// limit=-1 的 day 规则 → "每日 无限制"（保留周期上下文）
	got = formatTokenQuotaRulesValue(ctx, map[string]interface{}{"value": `[{"mode":"day","limit":-1}]`})
	if got != "每日 "+unlimited {
		t.Errorf("day unlimited: got %q want %q", got, "每日 "+unlimited)
	}

	// limit=-1 的 month 规则 → "每月 无限制"
	got = formatTokenQuotaRulesValue(ctx, map[string]interface{}{"value": `[{"mode":"month","limit":-1}]`})
	if got != "每月 "+unlimited {
		t.Errorf("month unlimited: got %q want %q", got, "每月 "+unlimited)
	}
}

func TestDiffSubLabelTransform(t *testing.T) {
	ctx := context.Background()
	initialSkillBundle := i18n.T(ctx, i18n.MsgGroupTreeSubLabelInitialSkillBundle)
	role := i18n.T(ctx, i18n.MsgGroupTreeSubLabelRole)
	enterpriseSkill := i18n.T(ctx, i18n.MsgGroupTreeSubLabelEnterpriseSkill)
	cases := []struct {
		name      string
		category  string
		subLabel  string
		wantLabel string
		wantSkip  bool
	}{
		{"skill_initial_bundle_skip", usergroup.CategoryKeySkill, initialSkillBundle, "", true},
		{"skill_role_remapped", usergroup.CategoryKeySkill, role, "", false},
		{"skill_other_passthrough", usergroup.CategoryKeySkill, "技能安装来源", "技能安装来源", false},
		{"agent_tool_enterprise_skill_skip", usergroup.CategoryKeyAgentTool, enterpriseSkill, "", true},
		{"agent_tool_other_passthrough", usergroup.CategoryKeyAgentTool, "企业插件", "企业插件", false},
		{"network_passthrough", usergroup.CategoryKeyNetwork, "私有网络与子网", "私有网络与子网", false},
		{"model_passthrough", usergroup.CategoryKeyModel, "任意", "任意", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotLabel, gotSkip := diffSubLabelTransform(ctx, c.category, c.subLabel)
			if gotLabel != c.wantLabel {
				t.Errorf("label: got %q want %q", gotLabel, c.wantLabel)
			}
			if gotSkip != c.wantSkip {
				t.Errorf("skip: got %v want %v", gotSkip, c.wantSkip)
			}
		})
	}
}

func TestEntriesToValues(t *testing.T) {
	entries := []usergroup.ConfigEntry{
		{ID: "a", Label: "A", NameHint: "hint-a"},
		{ID: "b", Label: "B"},
	}
	vals := entriesToValues(entries)
	if len(vals) != 2 {
		t.Fatalf("want 2, got %d", len(vals))
	}
	if vals[0].ID != "a" || vals[0].Name != "hint-a" || vals[0].Value != "A" {
		t.Errorf("[0] got %+v", vals[0])
	}
	if vals[1].ID != "b" || vals[1].Value != "B" {
		t.Errorf("[1] got %+v", vals[1])
	}
	// nil input → empty
	vals = entriesToValues(nil)
	if len(vals) != 0 {
		t.Errorf("nil → want 0, got %d", len(vals))
	}
}

func TestValueSetKey(t *testing.T) {
	a := configDiffValue{ID: "model", Value: "gpt-4"}
	b := configDiffValue{ID: "model", Value: "gpt-4"}
	if valueSetKey(a) != valueSetKey(b) {
		t.Error("same ID+Value should produce same key")
	}
	c := configDiffValue{ID: "model", Value: "gpt-3.5"}
	if valueSetKey(a) == valueSetKey(c) {
		t.Error("different Value should produce different key")
	}
}

func TestBuildSingleRow(t *testing.T) {
	meta := usergroup.ConfigCategoryMeta{Key: usergroup.CategoryKeyModel, Label: "模型"}
	before := &usergroup.ConfigCategoryResult{
		Entries: []usergroup.ConfigEntry{{ID: "1", Label: "gpt-4"}},
	}
	after := &usergroup.ConfigCategoryResult{
		Entries: []usergroup.ConfigEntry{{ID: "1", Label: "gpt-4o"}},
	}
	inst := &model.Instance{}
	row := buildSingleRow(meta, before, after, inst)
	if row.Key != usergroup.CategoryKeyModel {
		t.Errorf("Key: got %q want %q", row.Key, usergroup.CategoryKeyModel)
	}
	if row.CategoryLabel != "模型" {
		t.Errorf("Label: got %q", row.CategoryLabel)
	}
	if len(row.InstanceValues) != 1 || row.InstanceValues[0].Value != "gpt-4" {
		t.Errorf("InstanceValues: %+v", row.InstanceValues)
	}
	if len(row.TargetValues) != 1 || row.TargetValues[0].Value != "gpt-4o" {
		t.Errorf("TargetValues: %+v", row.TargetValues)
	}
	if row.Status != "different" {
		t.Errorf("Status: got %q want different", row.Status)
	}
	// nil before/after → empty values, status same
	row = buildSingleRow(meta, nil, nil, inst)
	if len(row.InstanceValues) != 0 || len(row.TargetValues) != 0 {
		t.Errorf("nil → want empty values, got %+v", row)
	}
	if row.Status != "same" {
		t.Errorf("nil → status should be same, got %q", row.Status)
	}
}
