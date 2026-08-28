package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	"gorm.io/gorm"
)

// ============================================================================
// controller/sg_pool.go 单元测试
//
// 覆盖目标：SelectSGForNewInstance 四路径 / AutoScaleSG / GetDefaultRuleSet
// 缓存 / MarkInstanceBound|Unbound / buildManagedSG(Name|Description) /
// rulesJSONToPolicySet / ruleToPolicy / truncateIdentifier / tryDeleteCloudSG
// ============================================================================

// setupSGPoolTestDB 在内存 SQLite 建包含 sg_pool 所需的 4 张表：
// RuleSet / ManagedSGPool / SiteConfig / Instance。
// 测试结束后恢复 model.DB 为原值，避免跨用例 DB 污染（后续测试可能
// 依赖包级 model.DB 指向的实例含有它们自己的表）。
func setupSGPoolTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	// SQLite in-memory 数据库是连接私有的，必须固定为单连接，
	// 否则连接池新开连接会看到空数据库（无任何表）。
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&model.RuleSet{},
		&model.ManagedSGPool{},
		&model.SiteConfig{},
		&model.Instance{},
		&model.AuditLog{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	sqlDB, _ = db.DB()
	sqlDB.SetMaxOpenConns(1)
	restoreDB := model.UseDBForTest(db)
	// 清理 GetDefaultRuleSet 的进程缓存，避免跨用例脏读
	defaultRuleSetCache = sync.Map{}
	// 替换 AutoScaleSG 异步审计为同步 no-op，避免 goroutine 在测试 cleanup
	// 后访问已销毁的 model.DB 造成 nil 解引用 panic。
	origLogAuditFn := sgPoolLogAuditFn
	sgPoolLogAuditFn = func(ctx context.Context, _ time.Time, _ uint, _, _, _, _, _ string) {}
	t.Cleanup(func() {
		sgPoolLogAuditFn = origLogAuditFn
		restoreDB()
		defaultRuleSetCache = sync.Map{}
	})
	return db
}

// seedRuleSetAndSGs 插入一行 RuleSet + 指定的 SG 列表（status/cvm_count 独立控制）。
func seedRuleSetAndSGs(t *testing.T, db *gorm.DB, sgs []model.ManagedSGPool) *model.RuleSet {
	t.Helper()
	rs := model.RuleSet{
		Name:         model.DefaultRuleSetName,
		Description:  "test ruleset",
		Rules:        "[]",
		Version:      1,
		UserGroupIDs: "[]",
		IsDefault:    true,
	}
	if err := db.Create(&rs).Error; err != nil {
		t.Fatalf("seed rule_set: %v", err)
	}
	for i := range sgs {
		sgs[i].RuleSetID = rs.ID
		if sgs[i].Status == "" {
			sgs[i].Status = model.SGStatusActive
		}
		// 若未显式设置 RuleVersion，默认给 1（NextSGOrdinalForRuleSet 只算
		// rule_version > 0 的行；默认值 0 会被过滤掉导致 ordinal 计算偏差）
		if sgs[i].RuleVersion == 0 {
			sgs[i].RuleVersion = 1
		}
		if err := db.Create(&sgs[i]).Error; err != nil {
			t.Fatalf("seed sg %s: %v", sgs[i].SGID, err)
		}
	}
	// SelectSGForNewInstance 会通过 effectiveSGPoolThreshold 读 site_configs，
	// 确保有一行 SiteConfig，默认 1800。
	_ = db.Create(&model.SiteConfig{SGPoolAutoScaleThreshold: 1800}).Error
	return &rs
}

// ------------------------------------------------------------
// buildManagedSGName / buildManagedSGDescription / truncateIdentifier
// ------------------------------------------------------------

func TestTruncateIdentifier(t *testing.T) {
	cases := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"short unchanged", "acme", 20, "acme"},
		{"exactly at max", "abcdefghij", 10, "abcdefghij"},
		{"over max cut", "abcdefghijklm", 10, "abcdefghij"},
		{"empty", "", 10, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := truncateIdentifier(c.in, c.max)
			if got != c.want {
				t.Errorf("truncateIdentifier(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
			}
		})
	}
}

func TestBuildManagedSGName(t *testing.T) {
	cases := []struct {
		ident string
		rs    string
		ord   int
		want  string
	}{
		{"acme", "clawpro-default", 1, "clawpro-sg-acme-clawpro-default-01"},
		{"acme", "clawpro-default", 12, "clawpro-sg-acme-clawpro-default-12"},
		// 序号 100 也能输出（不再补零位数）
		{"acme", "default", 100, "clawpro-sg-acme-default-100"},
		// identifier 超 20 字符被截断
		{strings.Repeat("x", 25), "default", 1, "clawpro-sg-" + strings.Repeat("x", 20) + "-default-01"},
		// name 超 20 字符被截断
		{"acme", strings.Repeat("n", 25), 1, "clawpro-sg-acme-" + strings.Repeat("n", 20) + "-01"},
	}
	for i, c := range cases {
		got := buildManagedSGName(c.ident, c.rs, c.ord)
		if got != c.want {
			t.Errorf("case %d buildManagedSGName(%q, %q, %d) = %q, want %q", i, c.ident, c.rs, c.ord, got, c.want)
		}
	}
	// 外部导出入口跟内部版本一致
	if BuildManagedSGName("a", "b", 1) != buildManagedSGName("a", "b", 1) {
		t.Error("BuildManagedSGName(exported) != buildManagedSGName(internal)")
	}
}

func TestBuildManagedSGDescription(t *testing.T) {
	got := buildManagedSGDescription("acme", "clawpro-default", 3)
	// 检查关键片段：【请勿删除】 + identifier-name + #ordinal
	for _, want := range []string{"【请勿删除】", "acme-clawpro-default", "#3", "手动改动无效"} {
		if !strings.Contains(got, want) {
			t.Errorf("description 缺少片段 %q, got: %s", want, got)
		}
	}
	if BuildManagedSGDescription("a", "b", 1) != buildManagedSGDescription("a", "b", 1) {
		t.Error("exported != internal")
	}
}

func TestIsClawproManagedSGName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"clawpro-sg-acme-default-01", true},
		{"clawpro-sg-", true},
		{"my-sg", false},
		{"", false},
		{"default", false},
	}
	for _, c := range cases {
		if got := IsClawproManagedSGName(c.name); got != c.want {
			t.Errorf("IsClawproManagedSGName(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestShortRand(t *testing.T) {
	// 同 n 两次调用不保证相等，但长度和字符集必须稳定
	a := shortRand(8)
	b := shortRand(8)
	if len(a) != 8 || len(b) != 8 {
		t.Errorf("shortRand(8) returned len(a)=%d len(b)=%d", len(a), len(b))
	}
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	for _, r := range a + b {
		if !strings.ContainsRune(charset, r) {
			t.Errorf("shortRand produced out-of-charset rune %q", r)
		}
	}
}

// ------------------------------------------------------------
// rulesJSONToPolicySet / ruleToPolicy
// ------------------------------------------------------------

func TestRulesJSONToPolicySet(t *testing.T) {
	t.Run("empty string returns empty set", func(t *testing.T) {
		set, err := rulesJSONToPolicySet("")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(set.Ingress) != 0 || len(set.Egress) != 0 {
			t.Error("empty string should produce empty Ingress/Egress")
		}
	})
	t.Run("empty array same", func(t *testing.T) {
		set, err := rulesJSONToPolicySet("[]")
		if err != nil || len(set.Ingress) != 0 || len(set.Egress) != 0 {
			t.Errorf("[] should produce empty set, err=%v", err)
		}
	})
	t.Run("invalid JSON errors", func(t *testing.T) {
		_, err := rulesJSONToPolicySet("not-json")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
	t.Run("ingress egress split", func(t *testing.T) {
		in := `[
			{"direction":"INGRESS","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT"},
			{"direction":"EGRESS","protocol":"ALL","port":"ALL","cidr_block":"::/0","action":"ACCEPT"},
			{"direction":"unknown","protocol":"TCP","port":"80","cidr_block":"10.0.0.0/8","action":"ACCEPT"}
		]`
		set, err := rulesJSONToPolicySet(in)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(set.Ingress) != 2 { // INGRESS 1 + unknown 1（容忍回落到 Ingress）
			t.Errorf("expected 2 ingress policies (including 1 fallback), got %d", len(set.Ingress))
		}
		if len(set.Egress) != 1 {
			t.Errorf("expected 1 egress policy, got %d", len(set.Egress))
		}
		// IPv6 CIDR 应该被放在 Ipv6CidrBlock
		v6 := set.Egress[0]
		if v6.Ipv6CidrBlock == nil || *v6.Ipv6CidrBlock != "::/0" {
			t.Errorf("IPv6 CIDR should land in Ipv6CidrBlock, got %+v", v6)
		}
		if v6.CidrBlock != nil {
			t.Errorf("CidrBlock should be nil for IPv6 rule, got %v", *v6.CidrBlock)
		}
	})
}

func TestRuleToPolicy_AllFieldsPopulated(t *testing.T) {
	r := &Rule{
		Direction: "INGRESS", Protocol: "TCP", Port: "80",
		CidrBlock: "1.2.3.4/32", Action: "ACCEPT",
		PolicyDescription: "web",
	}
	p := ruleToPolicy(r)
	if p.Protocol == nil || *p.Protocol != "TCP" {
		t.Errorf("Protocol not copied")
	}
	if p.Port == nil || *p.Port != "80" {
		t.Errorf("Port not copied")
	}
	if p.CidrBlock == nil || *p.CidrBlock != "1.2.3.4/32" {
		t.Errorf("CidrBlock not copied")
	}
	if p.Action == nil || *p.Action != "ACCEPT" {
		t.Errorf("Action not copied")
	}
	if p.PolicyDescription == nil || *p.PolicyDescription != "web" {
		t.Errorf("PolicyDescription not copied")
	}
}

func TestRuleToPolicy_EmptyFieldsNotSet(t *testing.T) {
	// 空字符串应保持 nil 指针，避免把 "" 发给云 API
	r := &Rule{Direction: "INGRESS"}
	p := ruleToPolicy(r)
	if p.Protocol != nil || p.Port != nil || p.CidrBlock != nil || p.Action != nil || p.PolicyDescription != nil {
		t.Errorf("empty rule fields should not be set, got %+v", p)
	}
}

// TestRuleToPolicy_ALLProtocolAndPortNormalizedToNil 验证 "ALL" 的适配层归一化：
// 本地 RuleSet schema 允许 "ALL" 作为业务层简写，但腾讯云 ModifySecurityGroupPolicies
// API 对 Protocol=="ALL" 返回 InvalidParameterValue —— 云端要求"字段不传"才表达
// "全部协议"。Port 同理。回归这个分支避免未来有人重新引入 bug。
//
// ⚠️ 注意：CreateSecurityGroupPolicies API 行为不同（接受 "ALL"），但 ruleToPolicy
// 仅服务于 Modify 路径，所以这里以 Modify 行为为准。
func TestRuleToPolicy_ALLProtocolAndPortNormalizedToNil(t *testing.T) {
	cases := []struct {
		name string
		in   Rule
	}{
		{"uppercase ALL", Rule{Protocol: "ALL", Port: "ALL", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}},
		{"lowercase all", Rule{Protocol: "all", Port: "all", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}},
		{"mixed case All", Rule{Protocol: "All", Port: "All", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := ruleToPolicy(&c.in)
			if p.Protocol != nil {
				t.Errorf("Protocol=%q 应归一化成 nil，got %q", c.in.Protocol, *p.Protocol)
			}
			if p.Port != nil {
				t.Errorf("Port=%q 应归一化成 nil，got %q", c.in.Port, *p.Port)
			}
			// 其他字段仍应正常落
			if p.CidrBlock == nil || *p.CidrBlock != "0.0.0.0/0" {
				t.Errorf("CidrBlock 应正常透传")
			}
			if p.Action == nil || *p.Action != "ACCEPT" {
				t.Errorf("Action 应正常透传")
			}
		})
	}
}

// TestRuleToPolicy_NonALLProtocolPreserved 确保非 "ALL" 协议不被误伤（回归 fix 的边界）。
func TestRuleToPolicy_NonALLProtocolPreserved(t *testing.T) {
	r := &Rule{Protocol: "TCP", Port: "22"}
	p := ruleToPolicy(r)
	if p.Protocol == nil || *p.Protocol != "TCP" {
		t.Errorf("TCP 不应被归一化，got %v", p.Protocol)
	}
	if p.Port == nil || *p.Port != "22" {
		t.Errorf("具体端口 22 不应被归一化，got %v", p.Port)
	}
}

// ------------------------------------------------------------
// GetDefaultRuleSet 缓存路径
// ------------------------------------------------------------

func TestGetDefaultRuleSet_CacheHitAndMiss(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := seedRuleSetAndSGs(t, db, nil)

	// miss → 读 DB
	got1, err := GetDefaultRuleSet(context.Background())
	if err != nil || got1.ID != rs.ID {
		t.Fatalf("first call err=%v id=%d want=%d", err, got1.ID, rs.ID)
	}

	// hit → 仍返回正确 RuleSet
	got2, err := GetDefaultRuleSet(context.Background())
	if err != nil || got2.ID != rs.ID {
		t.Errorf("cache hit err=%v id=%d", err, got2.ID)
	}

	// 手动把底层 RuleSet 删掉，并且不清缓存；此时 First 读失败 → 会自动清缓存并重查
	if err := db.Delete(&model.RuleSet{}, rs.ID).Error; err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = GetDefaultRuleSet(context.Background())
	if err == nil {
		t.Error("expected error after RuleSet removed")
	}

	// InvalidateDefaultRuleSetCache 公开入口：调用不报错即可
	InvalidateDefaultRuleSetCache(model.CurrentIdentifier(context.Background()))
}

// ------------------------------------------------------------
// selectActiveSGBelowCount
// ------------------------------------------------------------

func TestSelectActiveSGBelowCount(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := seedRuleSetAndSGs(t, db, []model.ManagedSGPool{
		{SGID: "sg-full", Status: model.SGStatusActive, CVMCount: 2000},
		{SGID: "sg-med", Status: model.SGStatusActive, CVMCount: 100},
		{SGID: "sg-low", Status: model.SGStatusActive, CVMCount: 5},
		{SGID: "sg-frozen", Status: model.SGStatusFrozen, CVMCount: 10},
	})

	t.Run("below threshold picks smallest", func(t *testing.T) {
		got, err := selectActiveSGBelowCount(context.Background(), rs.ID, 1800)
		if err != nil || got == nil || got.SGID != "sg-low" {
			t.Fatalf("want sg-low got=%+v err=%v", got, err)
		}
	})

	t.Run("buffer range includes sg-med", func(t *testing.T) {
		// 设较低阈值，sg-low 5 < 10；sg-med 100 超 → 只选到 sg-low
		got, err := selectActiveSGBelowCount(context.Background(), rs.ID, 10)
		if err != nil || got == nil || got.SGID != "sg-low" {
			t.Fatalf("want sg-low got=%+v err=%v", got, err)
		}
	})

	t.Run("nothing below returns nil", func(t *testing.T) {
		// 只有 sg-full cvm_count=2000, 阈值 1，全部大于；FROZEN 不被选；结果 nil
		got, err := selectActiveSGBelowCount(context.Background(), rs.ID, 1)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})
}

// ------------------------------------------------------------
// MarkInstanceBound / MarkInstanceUnbound
// ------------------------------------------------------------

func TestMarkInstanceBoundUnbound(t *testing.T) {
	db := setupSGPoolTestDB(t)
	seedRuleSetAndSGs(t, db, []model.ManagedSGPool{
		{SGID: "sg-x", Status: model.SGStatusActive, CVMCount: 5},
	})

	// bind: 5 → 6
	MarkInstanceBound(context.Background(), "sg-x")
	var got model.ManagedSGPool
	db.Where("sg_id = ?", "sg-x").First(&got)
	if got.CVMCount != 6 {
		t.Errorf("MarkInstanceBound: want 6, got %d", got.CVMCount)
	}

	// unbind: 6 → 5
	MarkInstanceUnbound(context.Background(), "", "sg-x")
	db.Where("sg_id = ?", "sg-x").First(&got)
	if got.CVMCount != 5 {
		t.Errorf("MarkInstanceUnbound: want 5, got %d", got.CVMCount)
	}

	// 空字符串应忽略（no-op），不 panic
	MarkInstanceBound(context.Background(), "")
	MarkInstanceUnbound(context.Background(), "", "")

	// 不存在的 sg：不报错（SQL 影响 0 行）
	MarkInstanceBound(context.Background(), "sg-nonexistent")
	MarkInstanceUnbound(context.Background(), "sg-nonexistent", "")
}

// ------------------------------------------------------------
// tryDeleteCloudSG + createCloudSG + applyRulesToCloudSG + fake hooks
// ------------------------------------------------------------

// fakeSGPoolVpcClient 实现 sgVpcClient，关注 sg_pool.go 真正用到的三个方法：
// CreateSecurityGroup / DeleteSecurityGroup / ModifySecurityGroupPolicies。
// 其他方法返回空响应（满足接口即可）。
type fakeSGPoolVpcClient struct {
	// CreateSecurityGroup
	createResp *vpc.CreateSecurityGroupResponse
	createErr  error
	createName string // 捕获调用参数
	createDesc string

	// DeleteSecurityGroup
	deletedSGID string
	deleteErr   error

	// ModifySecurityGroupPolicies
	modifyReqs []*vpc.ModifySecurityGroupPoliciesRequest
	modifyErr  error

	// DescribeSecurityGroupPolicies（用于 clearAllRulesForSG 测试注入）
	describePoliciesResp *vpc.DescribeSecurityGroupPoliciesResponse
	describePoliciesErr  error

	// DeleteSecurityGroupPolicies（用于 clearAllRulesForSG 测试注入）
	deletePoliciesReqs []*vpc.DeleteSecurityGroupPoliciesRequest
	deletePoliciesErr  error
}

func (f *fakeSGPoolVpcClient) DescribeSecurityGroups(req *vpc.DescribeSecurityGroupsRequest) (*vpc.DescribeSecurityGroupsResponse, error) {
	return &vpc.DescribeSecurityGroupsResponse{Response: &vpc.DescribeSecurityGroupsResponseParams{}}, nil
}

func (f *fakeSGPoolVpcClient) CreateSecurityGroup(req *vpc.CreateSecurityGroupRequest) (*vpc.CreateSecurityGroupResponse, error) {
	if req.GroupName != nil {
		f.createName = *req.GroupName
	}
	if req.GroupDescription != nil {
		f.createDesc = *req.GroupDescription
	}
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp != nil {
		return f.createResp, nil
	}
	return &vpc.CreateSecurityGroupResponse{
		Response: &vpc.CreateSecurityGroupResponseParams{
			SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: common.StringPtr("sg-created")},
		},
	}, nil
}

func (f *fakeSGPoolVpcClient) DeleteSecurityGroup(req *vpc.DeleteSecurityGroupRequest) (*vpc.DeleteSecurityGroupResponse, error) {
	if req.SecurityGroupId != nil {
		f.deletedSGID = *req.SecurityGroupId
	}
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	return &vpc.DeleteSecurityGroupResponse{Response: &vpc.DeleteSecurityGroupResponseParams{}}, nil
}

func (f *fakeSGPoolVpcClient) ModifySecurityGroupAttribute(req *vpc.ModifySecurityGroupAttributeRequest) (*vpc.ModifySecurityGroupAttributeResponse, error) {
	return &vpc.ModifySecurityGroupAttributeResponse{}, nil
}

func (f *fakeSGPoolVpcClient) DescribeSecurityGroupPolicies(req *vpc.DescribeSecurityGroupPoliciesRequest) (*vpc.DescribeSecurityGroupPoliciesResponse, error) {
	if f.describePoliciesErr != nil {
		return nil, f.describePoliciesErr
	}
	if f.describePoliciesResp != nil {
		return f.describePoliciesResp, nil
	}
	return &vpc.DescribeSecurityGroupPoliciesResponse{
		Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
			SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{},
		},
	}, nil
}

func (f *fakeSGPoolVpcClient) CreateSecurityGroupPolicies(req *vpc.CreateSecurityGroupPoliciesRequest) (*vpc.CreateSecurityGroupPoliciesResponse, error) {
	return &vpc.CreateSecurityGroupPoliciesResponse{}, nil
}

func (f *fakeSGPoolVpcClient) ReplaceSecurityGroupPolicy(req *vpc.ReplaceSecurityGroupPolicyRequest) (*vpc.ReplaceSecurityGroupPolicyResponse, error) {
	return &vpc.ReplaceSecurityGroupPolicyResponse{}, nil
}

func (f *fakeSGPoolVpcClient) ModifySecurityGroupPolicies(req *vpc.ModifySecurityGroupPoliciesRequest) (*vpc.ModifySecurityGroupPoliciesResponse, error) {
	f.modifyReqs = append(f.modifyReqs, req)
	if f.modifyErr != nil {
		return nil, f.modifyErr
	}
	return &vpc.ModifySecurityGroupPoliciesResponse{}, nil
}

func (f *fakeSGPoolVpcClient) DeleteSecurityGroupPolicies(req *vpc.DeleteSecurityGroupPoliciesRequest) (*vpc.DeleteSecurityGroupPoliciesResponse, error) {
	f.deletePoliciesReqs = append(f.deletePoliciesReqs, req)
	if f.deletePoliciesErr != nil {
		return nil, f.deletePoliciesErr
	}
	return &vpc.DeleteSecurityGroupPoliciesResponse{}, nil
}

func (f *fakeSGPoolVpcClient) DescribeVpcs(req *vpc.DescribeVpcsRequest) (*vpc.DescribeVpcsResponse, error) {
	return &vpc.DescribeVpcsResponse{}, nil
}

func (f *fakeSGPoolVpcClient) DescribeSecurityGroupAssociationStatistics(req *vpc.DescribeSecurityGroupAssociationStatisticsRequest) (*vpc.DescribeSecurityGroupAssociationStatisticsResponse, error) {
	return &vpc.DescribeSecurityGroupAssociationStatisticsResponse{}, nil
}

// withFakeSGPoolVpcClient 替换 newVpcClientForSGFn；返回 teardown。
func withFakeSGPoolVpcClient(fake *fakeSGPoolVpcClient) func() {
	orig := newVpcClientForSGFn
	newVpcClientForSGFn = func(ctx context.Context) (sgVpcClient, error) { return fake, nil }
	return func() { newVpcClientForSGFn = orig }
}

func withFakeSGPoolVpcClientErr(err error) func() {
	orig := newVpcClientForSGFn
	newVpcClientForSGFn = func(ctx context.Context) (sgVpcClient, error) { return nil, err }
	return func() { newVpcClientForSGFn = orig }
}

func TestCreateCloudSG_Success(t *testing.T) {
	fake := &fakeSGPoolVpcClient{}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	sgID, err := createCloudSG(context.Background(), "clawpro-sg-acme-default-01", "desc")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if sgID != "sg-created" {
		t.Errorf("sg_id = %q want sg-created", sgID)
	}
	if fake.createName != "clawpro-sg-acme-default-01" || fake.createDesc != "desc" {
		t.Errorf("capture name=%q desc=%q", fake.createName, fake.createDesc)
	}

	// 测试导出入口
	if _, err := CreateCloudSG(context.Background(), "a", "b"); err != nil {
		t.Errorf("CreateCloudSG exported: %v", err)
	}
}

func TestCreateCloudSG_EmptyResponse(t *testing.T) {
	fake := &fakeSGPoolVpcClient{createResp: &vpc.CreateSecurityGroupResponse{
		Response: &vpc.CreateSecurityGroupResponseParams{},
	}}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	_, err := createCloudSG(context.Background(), "n", "d")
	if err == nil || !errors.Is(err, hcommon.I18nError(i18n.MsgSGPoolCreateSGEmptyResp)) {
		t.Errorf("expected empty response error, got %v", err)
	}
}

func TestCreateCloudSG_ClientError(t *testing.T) {
	restore := withFakeSGPoolVpcClientErr(errors.New("no creds"))
	defer restore()

	if _, err := createCloudSG(context.Background(), "n", "d"); err == nil {
		t.Error("expected error")
	}
}

func TestCreateCloudSG_APIError(t *testing.T) {
	fake := &fakeSGPoolVpcClient{createErr: errors.New("api broken")}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	if _, err := createCloudSG(context.Background(), "n", "d"); err == nil || !strings.Contains(err.Error(), "api broken") {
		t.Errorf("expected api error, got %v", err)
	}
}

func TestApplyRulesToCloudSG_Success(t *testing.T) {
	fake := &fakeSGPoolVpcClient{}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	rules := `[{"direction":"INGRESS","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`
	if err := applyRulesToCloudSG(context.Background(), "sg-abc", rules); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(fake.modifyReqs) != 1 {
		t.Fatalf("expected 1 ModifySecurityGroupPolicies call, got %d", len(fake.modifyReqs))
	}
	r := fake.modifyReqs[0]
	if r.SecurityGroupId == nil || *r.SecurityGroupId != "sg-abc" {
		t.Errorf("sg_id in request not sg-abc")
	}
	if r.SecurityGroupPolicySet == nil || len(r.SecurityGroupPolicySet.Ingress) != 1 {
		t.Errorf("ingress missing in payload")
	}

	// 导出入口
	if err := ApplyRulesToCloudSG(context.Background(), "sg", "[]"); err != nil {
		t.Errorf("exported: %v", err)
	}
}

func TestApplyRulesToCloudSG_InvalidJSON(t *testing.T) {
	fake := &fakeSGPoolVpcClient{}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	err := applyRulesToCloudSG(context.Background(), "sg-abc", "not-json")
	if err == nil || !errors.Is(err, hcommon.I18nError(i18n.MsgSGPoolParseRulesJSONFailed)) {
		t.Errorf("expected parse error, got %v", err)
	}
}

func TestApplyRulesToCloudSG_ClientError(t *testing.T) {
	restore := withFakeSGPoolVpcClientErr(errors.New("no creds"))
	defer restore()
	if err := applyRulesToCloudSG(context.Background(), "sg", "[]"); err == nil {
		t.Error("expected error")
	}
}

func TestApplyRulesToCloudSG_ModifyError(t *testing.T) {
	fake := &fakeSGPoolVpcClient{modifyErr: errors.New("modify down")}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()
	// 传一条非空规则确保走 ModifySecurityGroupPolicies 路径（空规则现在走 clearAllRulesForSG）
	rules := `[{"direction":"INGRESS","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`
	if err := applyRulesToCloudSG(context.Background(), "sg", rules); err == nil {
		t.Error("expected modify error")
	}
}

// TestApplyRulesToCloudSG_EmptyRules_ClearPath 验证空规则集合走 clearAllRulesForSG 而非 Modify。
// fake 客户端默认 Describe 返回空 PolicySet → 跳过 Delete 直接成功（幂等场景）。
func TestApplyRulesToCloudSG_EmptyRules_ClearPath(t *testing.T) {
	fake := &fakeSGPoolVpcClient{
		// 故意把 Modify 设置为返回错误，证明根本没走 Modify 路径
		modifyErr: errors.New("modify should not be called"),
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()
	if err := applyRulesToCloudSG(context.Background(), "sg", "[]"); err != nil {
		t.Errorf("empty rules should clear successfully (no-op when cloud is already empty), got: %v", err)
	}
	if len(fake.modifyReqs) != 0 {
		t.Errorf("ModifySecurityGroupPolicies should NOT be called for empty rules, got %d calls", len(fake.modifyReqs))
	}
}

// TestClearAllRulesForSG_DeletesByPolicyIndex 验证 clearAllRulesForSG 在云端有规则时
// 按 PolicyIndex 删除 ingress + egress（分两次调用），且 Delete 请求只携带 PolicyIndex
// 不带其他字段（避免 AddressTemplate / Ipv6CidrBlock 空字符串导致云 API 校验失败）。
func TestClearAllRulesForSG_DeletesByPolicyIndex(t *testing.T) {
	idx0 := int64(0)
	idx1 := int64(1)
	idx2 := int64(2)
	fake := &fakeSGPoolVpcClient{
		describePoliciesResp: &vpc.DescribeSecurityGroupPoliciesResponse{
			Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
				SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{
					Ingress: []*vpc.SecurityGroupPolicy{
						// 含空 AddressTemplate / Ipv6CidrBlock 模拟云端真实返回
						{PolicyIndex: &idx0, Protocol: common.StringPtr("TCP"), Port: common.StringPtr("22"),
							CidrBlock: common.StringPtr("0.0.0.0/0"), Action: common.StringPtr("ACCEPT"),
							AddressTemplate: &vpc.AddressTemplateSpecification{}},
						{PolicyIndex: &idx1, Protocol: common.StringPtr("ALL"), Port: common.StringPtr("ALL"),
							Ipv6CidrBlock: common.StringPtr(""), Action: common.StringPtr("DROP")},
					},
					Egress: []*vpc.SecurityGroupPolicy{
						{PolicyIndex: &idx2, Protocol: common.StringPtr("ALL"), Port: common.StringPtr("ALL"),
							CidrBlock: common.StringPtr("0.0.0.0/0"), Action: common.StringPtr("ACCEPT")},
					},
				},
			},
		},
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	if err := applyRulesToCloudSG(context.Background(), "sg-x", "[]"); err != nil {
		t.Fatalf("clear should succeed, got: %v", err)
	}
	// 应该有 2 次 Delete 调用：一次 ingress、一次 egress
	if got := len(fake.deletePoliciesReqs); got != 2 {
		t.Fatalf("expected 2 Delete calls (ingress + egress), got %d", got)
	}
	// 验证每次 Delete 的请求里只携带 PolicyIndex
	ingressReq := fake.deletePoliciesReqs[0]
	if ingressReq.SecurityGroupPolicySet == nil || len(ingressReq.SecurityGroupPolicySet.Ingress) != 2 {
		t.Errorf("ingress delete should target 2 policies, got %v", ingressReq.SecurityGroupPolicySet)
	}
	for i, p := range ingressReq.SecurityGroupPolicySet.Ingress {
		if p.PolicyIndex == nil {
			t.Errorf("ingress[%d] PolicyIndex should be set", i)
		}
		// 确保关键的"污染字段"没被回传（这正是修复的核心目的）
		if p.AddressTemplate != nil {
			t.Errorf("ingress[%d] AddressTemplate must be nil to avoid cloud API validation error", i)
		}
		if p.Ipv6CidrBlock != nil {
			t.Errorf("ingress[%d] Ipv6CidrBlock must be nil", i)
		}
		if p.Protocol != nil || p.Port != nil || p.CidrBlock != nil || p.Action != nil {
			t.Errorf("ingress[%d] should only carry PolicyIndex, got Protocol=%v Port=%v CidrBlock=%v Action=%v",
				i, p.Protocol, p.Port, p.CidrBlock, p.Action)
		}
	}
	egressReq := fake.deletePoliciesReqs[1]
	if egressReq.SecurityGroupPolicySet == nil || len(egressReq.SecurityGroupPolicySet.Egress) != 1 {
		t.Errorf("egress delete should target 1 policy, got %v", egressReq.SecurityGroupPolicySet)
	}
}

// TestClearAllRulesForSG_DescribeError 验证 Describe 失败时 clearAllRulesForSG 返回错误且不调 Delete。
func TestClearAllRulesForSG_DescribeError(t *testing.T) {
	fake := &fakeSGPoolVpcClient{describePoliciesErr: errors.New("describe down")}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()
	err := applyRulesToCloudSG(context.Background(), "sg-x", "[]")
	if err == nil || !errors.Is(err, hcommon.I18nError(i18n.MsgSGPoolDescribeForClearFailed)) {
		t.Errorf("expected describe error, got %v", err)
	}
	if len(fake.deletePoliciesReqs) != 0 {
		t.Errorf("Delete should not be called when Describe fails, got %d", len(fake.deletePoliciesReqs))
	}
}

// TestClearAllRulesForSG_DeleteError 验证 Delete 失败时返回错误。
func TestClearAllRulesForSG_DeleteError(t *testing.T) {
	idx := int64(0)
	fake := &fakeSGPoolVpcClient{
		describePoliciesResp: &vpc.DescribeSecurityGroupPoliciesResponse{
			Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
				SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{
					Ingress: []*vpc.SecurityGroupPolicy{{PolicyIndex: &idx}},
				},
			},
		},
		deletePoliciesErr: errors.New("delete down"),
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()
	err := applyRulesToCloudSG(context.Background(), "sg-x", "[]")
	if err == nil || !errors.Is(err, hcommon.I18nError(i18n.MsgSGPoolDeleteIngressFailed)) {
		t.Errorf("expected delete ingress error, got %v", err)
	}
}

// TestClearAllRulesForSG_NilDescribeResponse 验证 Describe 返回 nil PolicySet 时直接返回 nil（幂等）。
func TestClearAllRulesForSG_NilDescribeResponse(t *testing.T) {
	fake := &fakeSGPoolVpcClient{
		describePoliciesResp: &vpc.DescribeSecurityGroupPoliciesResponse{
			Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
				// SecurityGroupPolicySet 为 nil
			},
		},
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()
	if err := applyRulesToCloudSG(context.Background(), "sg-x", "[]"); err != nil {
		t.Errorf("nil PolicySet should be idempotent success, got %v", err)
	}
}

// TestClearAllRulesForSG_SkipsNilPolicies 验证当 PolicyIndex 为 nil 时跳过该条规则。
func TestClearAllRulesForSG_SkipsNilPolicies(t *testing.T) {
	idx := int64(0)
	fake := &fakeSGPoolVpcClient{
		describePoliciesResp: &vpc.DescribeSecurityGroupPoliciesResponse{
			Response: &vpc.DescribeSecurityGroupPoliciesResponseParams{
				SecurityGroupPolicySet: &vpc.SecurityGroupPolicySet{
					Ingress: []*vpc.SecurityGroupPolicy{
						nil,                 // 应被跳过
						{PolicyIndex: nil},  // 应被跳过
						{PolicyIndex: &idx}, // 应被保留
					},
				},
			},
		},
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()
	if err := applyRulesToCloudSG(context.Background(), "sg-x", "[]"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(fake.deletePoliciesReqs) != 1 {
		t.Fatalf("expected 1 delete call, got %d", len(fake.deletePoliciesReqs))
	}
	if got := len(fake.deletePoliciesReqs[0].SecurityGroupPolicySet.Ingress); got != 1 {
		t.Errorf("expected 1 valid policy after filter, got %d", got)
	}
}

func TestTryDeleteCloudSG(t *testing.T) {
	// 成功路径
	t.Run("success", func(t *testing.T) {
		fake := &fakeSGPoolVpcClient{}
		restore := withFakeSGPoolVpcClient(fake)
		defer restore()
		tryDeleteCloudSG(context.Background(), "sg-dead")
		if fake.deletedSGID != "sg-dead" {
			t.Errorf("delete not captured, got %q", fake.deletedSGID)
		}
		// 导出入口
		TryDeleteCloudSG(context.Background(), "sg-dead2")
		if fake.deletedSGID != "sg-dead2" {
			t.Errorf("TryDeleteCloudSG exported not captured")
		}
	})
	// 客户端构建失败：tryDeleteCloudSG 是 best-effort，不 panic
	t.Run("client fail no panic", func(t *testing.T) {
		restore := withFakeSGPoolVpcClientErr(errors.New("oops"))
		defer restore()
		tryDeleteCloudSG(context.Background(), "sg-x")
	})
	// 删除 API 失败同样 best-effort
	t.Run("api fail no panic", func(t *testing.T) {
		fake := &fakeSGPoolVpcClient{deleteErr: errors.New("nope")}
		restore := withFakeSGPoolVpcClient(fake)
		defer restore()
		tryDeleteCloudSG(context.Background(), "sg-x")
	})
}

// ------------------------------------------------------------
// effectiveSGPoolThreshold
// ------------------------------------------------------------

func TestEffectiveSGPoolThreshold(t *testing.T) {
	db := setupSGPoolTestDB(t)

	// 无 SiteConfig 行：model.GetSiteConfig 返回默认对象 → 阈值回落 1800
	if got := effectiveSGPoolThreshold(context.Background()); got != model.DefaultSGPoolAutoScaleThreshold {
		t.Errorf("no site config: want %d got %d", model.DefaultSGPoolAutoScaleThreshold, got)
	}

	// 写一条 SiteConfig 阈值 42
	if err := db.Create(&model.SiteConfig{SGPoolAutoScaleThreshold: 42}).Error; err != nil {
		t.Fatalf("create siteconfig: %v", err)
	}
	if got := effectiveSGPoolThreshold(context.Background()); got != 42 {
		t.Errorf("want 42 got %d", got)
	}

	// 零值/负值时兜底
	db.Model(&model.SiteConfig{}).Where("1=1").Update("sg_pool_auto_scale_threshold", 0)
	if got := effectiveSGPoolThreshold(context.Background()); got != model.DefaultSGPoolAutoScaleThreshold {
		t.Errorf("zero: want default got %d", got)
	}
}

// ------------------------------------------------------------
// SelectSGForNewInstance —— 四路径
// ------------------------------------------------------------

func TestSelectSGForNewInstance_NormalPath(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := seedRuleSetAndSGs(t, db, []model.ManagedSGPool{
		{SGID: "sg-a", Status: model.SGStatusActive, CVMCount: 100},
		{SGID: "sg-b", Status: model.SGStatusActive, CVMCount: 10},
	})

	sgID, usedBuf, err := SelectSGForNewInstance(context.Background(), "", rs.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if sgID != "sg-b" {
		t.Errorf("want sg-b (smallest cvm_count) got %s", sgID)
	}
	if usedBuf {
		t.Error("normal path should not be usedBuffer")
	}
}

func TestSelectSGForNewInstance_AutoScalePath(t *testing.T) {
	// 所有 ACTIVE SG 都已经超阈值 → 进入路径 2 AutoScaleSG → 由于 AutoScaleSG 需要 distlock，
	// 在 SQLite 环境下 AcquireLock 返回 "空壳 DistLock"（直接成功）。然后会走 cloud API。
	// 我们注入 fake vpc client 让 cloud API 返回一个新 sg_id。
	db := setupSGPoolTestDB(t)
	// seedRuleSetAndSGs 内部会写一行 SiteConfig，默认阈值 1800。
	// 设 sg-a cvm_count=2000，高于默认阈值 → 路径 1 查不到。
	rs := seedRuleSetAndSGs(t, db, []model.ManagedSGPool{
		{SGID: "sg-a", Status: model.SGStatusActive, CVMCount: 2000},
	})

	fake := &fakeSGPoolVpcClient{
		createResp: &vpc.CreateSecurityGroupResponse{
			Response: &vpc.CreateSecurityGroupResponseParams{
				SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: common.StringPtr("sg-new")},
			},
		},
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	sgID, usedBuf, err := SelectSGForNewInstance(context.Background(), "acme", rs.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if sgID != "sg-new" {
		t.Errorf("expected sg-new, got %s", sgID)
	}
	if usedBuf {
		t.Error("scale-up path should not be usedBuffer")
	}
}

func TestSelectSGForNewInstance_NoRuleSet(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// 建一个假的 SiteConfig，阈值设 1，强制走路径 2
	db.Create(&model.SiteConfig{SGPoolAutoScaleThreshold: 1})
	// 不建任何 RuleSet 或 SG；传一个不存在的 rule_set_id
	_, _, err := SelectSGForNewInstance(context.Background(), "acme", 999)
	if err == nil {
		t.Fatal("expected error when no rule set")
	}
	if !errors.Is(err, ErrNoBaseConfigured) {
		t.Errorf("want ErrNoBaseConfigured, got %v", err)
	}
}

// ------------------------------------------------------------
// AutoScaleSG 分支
// ------------------------------------------------------------

func TestAutoScaleSG_PoolAtMax(t *testing.T) {
	db := setupSGPoolTestDB(t)
	// 灌 20 个 ACTIVE → 达到 MaxSGPerRuleSet
	var many []model.ManagedSGPool
	for i := range model.MaxSGPerRuleSet {
		many = append(many, model.ManagedSGPool{
			SGID: fmt.Sprintf("sg-m%d", i), Status: model.SGStatusActive,
		})
	}
	rs := seedRuleSetAndSGs(t, db, many)

	_, err := AutoScaleSG(context.Background(), "acme", rs)
	if !errors.Is(err, ErrPoolAtMaxSize) {
		t.Errorf("expected ErrPoolAtMaxSize, got %v", err)
	}
}

func TestAutoScaleSG_Success(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := seedRuleSetAndSGs(t, db, []model.ManagedSGPool{
		{SGID: "sg-one", Status: model.SGStatusActive, CVMCount: 50},
	})

	fake := &fakeSGPoolVpcClient{
		createResp: &vpc.CreateSecurityGroupResponse{
			Response: &vpc.CreateSecurityGroupResponseParams{
				SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: common.StringPtr("sg-brand-new")},
			},
		},
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	newSG, err := AutoScaleSG(context.Background(), "acme", rs)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if newSG.SGID != "sg-brand-new" {
		t.Errorf("want sg-brand-new got %s", newSG.SGID)
	}
	// 新 SG 必须被 INSERT 到 DB
	var row model.ManagedSGPool
	if err := db.Where("sg_id = ?", "sg-brand-new").First(&row).Error; err != nil {
		t.Fatalf("new sg not in db: %v", err)
	}
	if row.Status != model.SGStatusActive || row.RuleSetID != rs.ID {
		t.Errorf("unexpected fields: %+v", row)
	}
	// create 请求名字应该是 clawpro-sg-acme-{name}-02（已有 sg-one 一条，ordinal=2）
	wantName := buildManagedSGName("acme", rs.Name, 2)
	if fake.createName != wantName {
		t.Errorf("create name = %q want %q", fake.createName, wantName)
	}
}

func TestAutoScaleSG_CreateCloudSGFails(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := seedRuleSetAndSGs(t, db, nil)

	fake := &fakeSGPoolVpcClient{createErr: errors.New("no quota")}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	_, err := AutoScaleSG(context.Background(), "acme", rs)
	if err == nil || !errors.Is(err, hcommon.I18nError(i18n.MsgSGPoolCreateCloudSGFailed)) {
		t.Errorf("expected create cloud sg error, got %v", err)
	}
}

func TestAutoScaleSG_ApplyRulesFails_CleansUp(t *testing.T) {
	db := setupSGPoolTestDB(t)
	rs := seedRuleSetAndSGs(t, db, nil)
	// 覆盖 rs.Rules 为非空，确保 AutoScaleSG 内部走 ModifySecurityGroupPolicies
	// 路径（空规则会改走 clearAllRulesForSG，无法触发 modifyErr）
	rs.Rules = `[{"direction":"INGRESS","protocol":"TCP","port":"22","cidr_block":"0.0.0.0/0","action":"ACCEPT"}]`
	if err := db.Save(rs).Error; err != nil {
		t.Fatalf("update rs.Rules: %v", err)
	}

	fake := &fakeSGPoolVpcClient{
		createResp: &vpc.CreateSecurityGroupResponse{
			Response: &vpc.CreateSecurityGroupResponseParams{
				SecurityGroup: &vpc.SecurityGroup{SecurityGroupId: common.StringPtr("sg-tmp")},
			},
		},
		modifyErr: errors.New("rules bad"),
	}
	restore := withFakeSGPoolVpcClient(fake)
	defer restore()

	_, err := AutoScaleSG(context.Background(), "acme", rs)
	if err == nil || !errors.Is(err, hcommon.I18nError(i18n.MsgSGPoolApplyRulesFailed)) {
		t.Errorf("expected apply rules error, got %v", err)
	}
	// 应当回滚调用 DeleteSecurityGroup
	if fake.deletedSGID != "sg-tmp" {
		t.Errorf("cleanup DeleteSecurityGroup not called on sg-tmp, got %q", fake.deletedSGID)
	}
	// 且 DB 里不应该有 sg-tmp
	var cnt int64
	db.Model(&model.ManagedSGPool{}).Where("sg_id = ?", "sg-tmp").Count(&cnt)
	if cnt != 0 {
		t.Errorf("failed scale should not leave row, got count=%d", cnt)
	}
}

// ============================================================================
// Change: sg-rule-support-sg-and-address-template
// ----------------------------------------------------------------------------
// 覆盖 ruleToPolicy 按前缀路由到腾讯云 SDK 互斥字段：
//   sg-    → p.SecurityGroupId
//   ipmg-  → p.AddressTemplate.AddressGroupId  （必须先于 ipm- 判断）
//   ipm-   → p.AddressTemplate.AddressId
// ============================================================================

// TestRuleToPolicy_SourceSGGoesToSecurityGroupId 验证 sg-xxx 前缀的 Rule 被正确路由到
// SDK 的 SecurityGroupId 字段，其他来源字段（CidrBlock / Ipv6CidrBlock / AddressTemplate）
// 必须全部 nil。这是腾讯云 ModifySecurityGroupPolicies API 互斥约束的体现。
func TestRuleToPolicy_SourceSGGoesToSecurityGroupId(t *testing.T) {
	r := &Rule{
		Direction: "INGRESS",
		Protocol:  "TCP",
		Port:      "22",
		CidrBlock: "sg-2f25udyn",
		Action:    "ACCEPT",
	}
	p := ruleToPolicy(r)
	if p.SecurityGroupId == nil || *p.SecurityGroupId != "sg-2f25udyn" {
		t.Errorf("SecurityGroupId = %v; want sg-2f25udyn", p.SecurityGroupId)
	}
	if p.CidrBlock != nil {
		t.Errorf("CidrBlock should be nil when SecurityGroupId set; got %q", *p.CidrBlock)
	}
	if p.Ipv6CidrBlock != nil {
		t.Errorf("Ipv6CidrBlock should be nil; got %q", *p.Ipv6CidrBlock)
	}
	if p.AddressTemplate != nil {
		t.Errorf("AddressTemplate should be nil; got %+v", p.AddressTemplate)
	}
}

// TestRuleToPolicy_AddressTemplateRoutesByPrefix 验证 ipm- 和 ipmg- 前缀分别路由到
// AddressTemplate 的两个子字段。
//
// ⚠️ 核心回归点：ipmg- 必须先被识别为 AddressGroupId，而非被当作 ipm- 的匹配对象
// 错误写入 AddressId。本测试通过显式断言"另一个子字段为 nil"锁定分类顺序。
func TestRuleToPolicy_AddressTemplateRoutesByPrefix(t *testing.T) {
	t.Run("ipm- goes to AddressId", func(t *testing.T) {
		r := &Rule{Direction: "INGRESS", Protocol: "TCP", Port: "80", CidrBlock: "ipm-aw8st7ni", Action: "ACCEPT"}
		p := ruleToPolicy(r)
		if p.AddressTemplate == nil {
			t.Fatal("AddressTemplate should not be nil")
		}
		if p.AddressTemplate.AddressId == nil || *p.AddressTemplate.AddressId != "ipm-aw8st7ni" {
			t.Errorf("AddressId = %v; want ipm-aw8st7ni", p.AddressTemplate.AddressId)
		}
		if p.AddressTemplate.AddressGroupId != nil {
			t.Errorf("AddressGroupId should be nil when ipm- prefix; got %q", *p.AddressTemplate.AddressGroupId)
		}
		// 其他来源字段也必须全 nil
		if p.CidrBlock != nil || p.Ipv6CidrBlock != nil || p.SecurityGroupId != nil {
			t.Errorf("other source fields must be nil: %+v", p)
		}
	})

	t.Run("ipmg- goes to AddressGroupId (not AddressId)", func(t *testing.T) {
		r := &Rule{Direction: "EGRESS", Protocol: "ALL", Port: "ALL", CidrBlock: "ipmg-abcdef12", Action: "ACCEPT"}
		p := ruleToPolicy(r)
		if p.AddressTemplate == nil {
			t.Fatal("AddressTemplate should not be nil")
		}
		if p.AddressTemplate.AddressGroupId == nil || *p.AddressTemplate.AddressGroupId != "ipmg-abcdef12" {
			t.Errorf("AddressGroupId = %v; want ipmg-abcdef12", p.AddressTemplate.AddressGroupId)
		}
		// ⚠️ 核心断言：绝不能被误识别为 AddressId
		if p.AddressTemplate.AddressId != nil {
			t.Errorf("AddressId must be nil when ipmg- prefix (prefix order bug); got %q", *p.AddressTemplate.AddressId)
		}
		if p.CidrBlock != nil || p.Ipv6CidrBlock != nil || p.SecurityGroupId != nil {
			t.Errorf("other source fields must be nil: %+v", p)
		}
	})
}

// TestRuleToPolicy_SourceFieldsMutuallyExclusive 五种来源类型遍历，每种只 set 一个字段、其他都 nil。
// 这是腾讯云 API 硬约束，一旦有两个非 nil 云端会直接返回 InvalidParameterValue。
func TestRuleToPolicy_SourceFieldsMutuallyExclusive(t *testing.T) {
	type check func(p *vpc.SecurityGroupPolicy) bool
	cases := []struct {
		name      string
		cidrBlock string
		wantSet   check
	}{
		{
			"IPv4 CIDR only sets CidrBlock",
			"10.0.0.0/8",
			func(p *vpc.SecurityGroupPolicy) bool {
				return p.CidrBlock != nil && p.Ipv6CidrBlock == nil && p.SecurityGroupId == nil && p.AddressTemplate == nil
			},
		},
		{
			"IPv6 CIDR only sets Ipv6CidrBlock",
			"::/0",
			func(p *vpc.SecurityGroupPolicy) bool {
				return p.Ipv6CidrBlock != nil && p.CidrBlock == nil && p.SecurityGroupId == nil && p.AddressTemplate == nil
			},
		},
		{
			"sg- only sets SecurityGroupId",
			"sg-2f25udyn",
			func(p *vpc.SecurityGroupPolicy) bool {
				return p.SecurityGroupId != nil && p.CidrBlock == nil && p.Ipv6CidrBlock == nil && p.AddressTemplate == nil
			},
		},
		{
			"ipm- only sets AddressTemplate.AddressId",
			"ipm-aw8st7ni",
			func(p *vpc.SecurityGroupPolicy) bool {
				return p.AddressTemplate != nil && p.AddressTemplate.AddressId != nil &&
					p.AddressTemplate.AddressGroupId == nil &&
					p.CidrBlock == nil && p.Ipv6CidrBlock == nil && p.SecurityGroupId == nil
			},
		},
		{
			"ipmg- only sets AddressTemplate.AddressGroupId",
			"ipmg-abcdef12",
			func(p *vpc.SecurityGroupPolicy) bool {
				return p.AddressTemplate != nil && p.AddressTemplate.AddressGroupId != nil &&
					p.AddressTemplate.AddressId == nil &&
					p.CidrBlock == nil && p.Ipv6CidrBlock == nil && p.SecurityGroupId == nil
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: c.cidrBlock, Action: "ACCEPT"}
			p := ruleToPolicy(r)
			if !c.wantSet(p) {
				t.Errorf("mutual exclusion violated: %+v", p)
			}
		})
	}
}

// TestRuleToPolicy_RoundTrip 正反向往返一致性：
// Rule(sg-xxx) → Policy(SecurityGroupId=sg-xxx) → Rule(CidrBlock=sg-xxx)。
// 这是 Guardian 不误报 drift 的关键——云端反向读回的规则指纹必须和 DB 正向下发的指纹一致。
func TestRuleToPolicy_RoundTrip(t *testing.T) {
	cases := []struct {
		name     string
		original Rule
	}{
		{"sg reference", Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "sg-2f25udyn", Action: "ACCEPT"}},
		{"address tpl", Rule{Direction: "INGRESS", Protocol: "ALL", Port: "ALL", CidrBlock: "ipm-aw8st7ni", Action: "ACCEPT"}},
		{"address group", Rule{Direction: "EGRESS", Protocol: "TCP", Port: "443", CidrBlock: "ipmg-abcdef12", Action: "DROP"}},
		{"ipv4 cidr", Rule{Direction: "INGRESS", Protocol: "TCP", Port: "22", CidrBlock: "0.0.0.0/0", Action: "ACCEPT"}},
		{"ipv6 cidr", Rule{Direction: "EGRESS", Protocol: "ALL", Port: "ALL", CidrBlock: "::/0", Action: "ACCEPT"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// 正向：Rule → Policy
			p := ruleToPolicy(&c.original)
			// 反向：Policy → Rule（补齐 Protocol/Port/Action，模拟云端返回）
			// ⚠️ policyToRule 不会还原被 ALL→nil 归一化掉的 Protocol/Port，因此对比指纹时
			// 直接用原规则的 Protocol/Port 填回（模拟云端保留 "ALL" 的真实响应）。
			if p.Protocol == nil && c.original.Protocol != "" {
				proto := c.original.Protocol
				p.Protocol = &proto
			}
			if p.Port == nil && c.original.Port != "" {
				port := c.original.Port
				p.Port = &port
			}
			r := policyToRule(p, c.original.Direction)
			if r.Fingerprint() != c.original.Fingerprint() {
				t.Errorf("round-trip fingerprint mismatch:\n  original: %s\n  roundtrip: %s", c.original.Fingerprint(), r.Fingerprint())
			}
		})
	}
}
