package model

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupAgentCommandTestDB 在内存 SQLite 中 AutoMigrate Agent 命令执行 4 张表。
func setupAgentCommandTestDB(t *testing.T) func() {
	t.Helper()
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := testDB.AutoMigrate(
		&AgentCommand{}, &AgentCommandDispatch{},
		&AgentCommandInvocation{}, &AgentCommandTask{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return UseDBForTest(testDB)
}

func TestAgentCommand_TableName(t *testing.T) {
	if name := (AgentCommand{}).TableName(); name != "agent_commands" {
		t.Errorf("AgentCommand.TableName() = %q, want agent_commands", name)
	}
	if name := (AgentCommandInvocation{}).TableName(); name != "agent_command_invocations" {
		t.Errorf("AgentCommandInvocation.TableName() = %q, want agent_command_invocations", name)
	}
	if name := (AgentCommandTask{}).TableName(); name != "agent_command_tasks" {
		t.Errorf("AgentCommandTask.TableName() = %q, want agent_command_tasks", name)
	}
}

func TestGenerateAgentCommandSlug_Format(t *testing.T) {
	for i := 0; i < 50; i++ {
		s := GenerateAgentCommandSlug()
		if !strings.HasPrefix(s, AgentCommandSlugPrefix) {
			t.Errorf("slug %q missing prefix %q", s, AgentCommandSlugPrefix)
		}
		rand := strings.TrimPrefix(s, AgentCommandSlugPrefix)
		if len(rand) != AgentCommandSlugRandLen {
			t.Errorf("slug %q random part len = %d, want %d", s, len(rand), AgentCommandSlugRandLen)
		}
		for _, b := range []byte(rand) {
			if !((b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')) {
				t.Errorf("slug %q contains invalid char %q", s, b)
			}
		}
	}
}

func TestGenerateAgentDispatchSlug_Format(t *testing.T) {
	s := GenerateAgentDispatchSlug()
	if !strings.HasPrefix(s, AgentCommandDispatchSlugPrefix) {
		t.Errorf("dispatch slug %q missing prefix", s)
	}
	if len(strings.TrimPrefix(s, AgentCommandDispatchSlugPrefix)) != AgentCommandDispatchSlugRandLen {
		t.Errorf("dispatch slug random part length wrong: %q", s)
	}
}

func TestValidateAgentCommandName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		ok   bool
	}{
		{"empty", "", false},
		{"english", "cleanup-logs", true},
		{"chinese", "清理临时日志", true},
		{"underscores", "rotate_log_v2", true},
		{"dot", "v1.0.script", true},
		{"too long", strings.Repeat("a", AgentCommandNameMaxChars+1), false},
		{"space", "bad name", false},
		{"slash", "bad/name", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateAgentCommandName(c.in)
			if c.ok && err != nil {
				t.Errorf("%q expected ok, got %v", c.in, err)
			}
			if !c.ok && err == nil {
				t.Errorf("%q expected error, got nil", c.in)
			}
		})
	}
}

func TestValidateAgentCommandContent(t *testing.T) {
	if err := ValidateAgentCommandContent(""); err == nil {
		t.Error("empty content should fail")
	}
	if err := ValidateAgentCommandContent("   \n\t"); err == nil {
		t.Error("whitespace-only content should fail")
	}
	if err := ValidateAgentCommandContent("#!/bin/bash\necho hello"); err != nil {
		t.Errorf("normal content failed: %v", err)
	}
	if err := ValidateAgentCommandContent(strings.Repeat("a", AgentCommandContentMaxChars+1)); err == nil {
		t.Error("oversize content should fail")
	}
}

func TestValidateAgentCommandTimeout(t *testing.T) {
	cases := []struct {
		sec  uint
		ok   bool
		desc string
	}{
		{0, false, "0 越下界"},
		{1, true, "下界 1"},
		{60, true, "默认 60"},
		{3600, true, "1 小时"},
		{86400, true, "上界 86400 (1 天，与 TAT API 上限对齐)"},
		{86401, false, "越上界"},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := ValidateAgentCommandTimeout(c.sec)
			if c.ok && err != nil {
				t.Errorf("%d should pass: %v", c.sec, err)
			}
			if !c.ok && err == nil {
				t.Errorf("%d should fail, got nil", c.sec)
			}
		})
	}
	// sanity check: 常量值与 TAT API 对齐
	if AgentCommandTimeoutMax != 86400 {
		t.Errorf("AgentCommandTimeoutMax = %d, want 86400 (TAT API max)", AgentCommandTimeoutMax)
	}
	if AgentCommandTimeoutDefault != 60 {
		t.Errorf("AgentCommandTimeoutDefault = %d, want 60", AgentCommandTimeoutDefault)
	}
}

// TestValidateAgentCommandRunUserAndWorkdir 校验 run_user / workdir 长度上限。
//
// 与 DB 列宽（varchar(64) / varchar(255)）对齐；按字符（rune）计，不是字节。
func TestValidateAgentCommandRunUserAndWorkdir(t *testing.T) {
	t.Run("run_user empty ok (will fall back to default)", func(t *testing.T) {
		if err := ValidateAgentCommandRunUser(""); err != nil {
			t.Errorf("empty run_user should pass, got %v", err)
		}
	})
	t.Run("run_user at limit", func(t *testing.T) {
		if err := ValidateAgentCommandRunUser(strings.Repeat("a", AgentCommandRunUserMaxChars)); err != nil {
			t.Errorf("at limit (%d) should pass, got %v", AgentCommandRunUserMaxChars, err)
		}
	})
	t.Run("run_user chinese at limit", func(t *testing.T) {
		// 64 个中文 = 192 字节 utf8，但字符数 = 64 应通过
		if err := ValidateAgentCommandRunUser(strings.Repeat("中", AgentCommandRunUserMaxChars)); err != nil {
			t.Errorf("64 chinese chars should pass, got %v", err)
		}
	})
	t.Run("run_user over limit", func(t *testing.T) {
		err := ValidateAgentCommandRunUser(strings.Repeat("a", AgentCommandRunUserMaxChars+1))
		if err != ErrAgentCommandRunUserTooLong {
			t.Errorf("over limit should return ErrAgentCommandRunUserTooLong, got %v", err)
		}
	})

	t.Run("workdir empty ok", func(t *testing.T) {
		if err := ValidateAgentCommandWorkdir(""); err != nil {
			t.Errorf("empty workdir should pass, got %v", err)
		}
	})
	t.Run("workdir at limit", func(t *testing.T) {
		if err := ValidateAgentCommandWorkdir(strings.Repeat("a", AgentCommandWorkdirMaxChars)); err != nil {
			t.Errorf("at limit (%d) should pass, got %v", AgentCommandWorkdirMaxChars, err)
		}
	})
	t.Run("workdir chinese at limit", func(t *testing.T) {
		if err := ValidateAgentCommandWorkdir(strings.Repeat("中", AgentCommandWorkdirMaxChars)); err != nil {
			t.Errorf("255 chinese chars should pass, got %v", err)
		}
	})
	t.Run("workdir over limit", func(t *testing.T) {
		err := ValidateAgentCommandWorkdir(strings.Repeat("a", AgentCommandWorkdirMaxChars+1))
		if err != ErrAgentCommandWorkdirTooLong {
			t.Errorf("over limit should return ErrAgentCommandWorkdirTooLong, got %v", err)
		}
	})

	// sanity check: 常量值与 DB schema 列宽对齐
	if AgentCommandRunUserMaxChars != 64 {
		t.Errorf("AgentCommandRunUserMaxChars = %d, want 64 (matches varchar(64))", AgentCommandRunUserMaxChars)
	}
	if AgentCommandWorkdirMaxChars != 255 {
		t.Errorf("AgentCommandWorkdirMaxChars = %d, want 255 (matches varchar(255))", AgentCommandWorkdirMaxChars)
	}
}

func TestValidateAgentCommandType(t *testing.T) {
	if err := ValidateAgentCommandType(AgentCommandTypeShell); err != nil {
		t.Errorf("SHELL should pass: %v", err)
	}
	if err := ValidateAgentCommandType("PYTHON"); err == nil {
		t.Error("unknown type should fail")
	}
}

func TestValidateAgentCommandParams(t *testing.T) {
	t.Run("empty ok", func(t *testing.T) {
		if _, err := ValidateAgentCommandParams(nil); err != nil {
			t.Errorf("nil should pass: %v", err)
		}
	})
	t.Run("normal", func(t *testing.T) {
		params := []AgentCommandParam{
			{Name: "port", Default: "8080", Description: "端口"},
			{Name: "log_dir", Default: "/var/log", Description: ""},
		}
		if _, err := ValidateAgentCommandParams(params); err != nil {
			t.Errorf("normal should pass: %v", err)
		}
	})
	t.Run("too many", func(t *testing.T) {
		params := make([]AgentCommandParam, AgentCommandParamsMax+1)
		for i := range params {
			params[i] = AgentCommandParam{Name: "p"}
		}
		if _, err := ValidateAgentCommandParams(params); err == nil {
			t.Error("over-cap should fail")
		}
	})
	t.Run("invalid name format", func(t *testing.T) {
		params := []AgentCommandParam{{Name: "1bad"}} // can't start with digit
		bad, err := ValidateAgentCommandParams(params)
		if err == nil {
			t.Error("bad name should fail")
		}
		if bad != "1bad" {
			t.Errorf("returned bad name = %q, want 1bad", bad)
		}
	})
	t.Run("duplicate names", func(t *testing.T) {
		params := []AgentCommandParam{
			{Name: "x"}, {Name: "x"},
		}
		bad, err := ValidateAgentCommandParams(params)
		if err == nil || bad != "x" {
			t.Errorf("duplicate should fail, got bad=%q err=%v", bad, err)
		}
	})
}

func TestAgentCommand_ParamsRoundTrip(t *testing.T) {
	c := AgentCommand{}
	params := []AgentCommandParam{
		{Name: "port", Default: "8080", Description: "端口号"},
		{Name: "host", Default: "127.0.0.1", Description: ""},
	}
	if err := c.SetParams(params); err != nil {
		t.Fatalf("SetParams: %v", err)
	}
	got := c.Params()
	if len(got) != 2 || got[0].Name != "port" || got[1].Default != "127.0.0.1" {
		t.Errorf("round-trip params mismatch: %+v", got)
	}
}

func TestCreateAgentCommandWithSlugRetry_ContentStoredAsRaw(t *testing.T) {
	defer setupAgentCommandTestDB(t)()
	ctx := context.Background()

	rawContent := "#!/bin/bash\necho 'hello {{name}}'\n# 中文注释\n"
	c := &AgentCommand{
		Identifier:      "",
		Name:            "test",
		Description:     "ut",
		Type:            AgentCommandTypeShell,
		Content:         rawContent,
		TimeoutSec:      60,
		RunUser:         "root",
		Workdir:         "/root",
		ParamsJSON:      `[]`,
		VisibilityType:  AgentCommandVisibilityTenant,
		CreatedByUserID: 1,
	}
	if err := CreateAgentCommandWithSlugRetry(ctx, c, 5); err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected primary key set")
	}
	if !strings.HasPrefix(c.Slug, AgentCommandSlugPrefix) {
		t.Errorf("slug not generated: %q", c.Slug)
	}

	// 关键断言：DB 取出的 content 等于原始 raw 文本，无 base64 编码（spec.md "content 与 snapshot 存 raw"）
	got, err := FindAgentCommandByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if got.Content != rawContent {
		t.Errorf("Content stored as base64 or modified! got=%q want=%q", got.Content, rawContent)
	}
}

func TestCountActiveAgentCommands_SoftDelete(t *testing.T) {
	defer setupAgentCommandTestDB(t)()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		c := &AgentCommand{Name: "n", Content: "echo", TimeoutSec: 60}
		if err := CreateAgentCommandWithSlugRetry(ctx, c, 5); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	n, err := CountActiveAgentCommands(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 3 {
		t.Errorf("active count = %d, want 3", n)
	}

	// 软删 1 条
	var first AgentCommand
	if err := DB(ctx).Order("id asc").First(&first).Error; err != nil {
		t.Fatalf("find first: %v", err)
	}
	if err := DB(ctx).Delete(&first).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	n2, err := CountActiveAgentCommands(ctx)
	if err != nil {
		t.Fatalf("count2: %v", err)
	}
	if n2 != 2 {
		t.Errorf("active count after soft delete = %d, want 2", n2)
	}

	// Unscoped 查询能看到软删行
	var allRows []AgentCommand
	if err := DB(ctx).Unscoped().Find(&allRows).Error; err != nil {
		t.Fatalf("unscoped find: %v", err)
	}
	if len(allRows) != 3 {
		t.Errorf("unscoped count = %d, want 3", len(allRows))
	}
}

func TestFindAgentCommandBySlug_IncludeDeleted(t *testing.T) {
	defer setupAgentCommandTestDB(t)()
	ctx := context.Background()

	c := &AgentCommand{Name: "n", Content: "echo"}
	if err := CreateAgentCommandWithSlugRetry(ctx, c, 5); err != nil {
		t.Fatalf("create: %v", err)
	}
	slug := c.Slug
	if err := DB(ctx).Delete(&AgentCommand{}, c.ID).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	// 普通查找：找不到（因为软删过滤）
	if _, err := FindAgentCommandBySlug(ctx, slug, false); err == nil {
		t.Error("normal lookup of soft-deleted slug should miss")
	}
	// includeDeleted：能看到
	got, err := FindAgentCommandBySlug(ctx, slug, true)
	if err != nil {
		t.Errorf("include-deleted lookup: %v", err)
	}
	if got != nil && got.Slug != slug {
		t.Errorf("got slug %q, want %q", got.Slug, slug)
	}
}

func TestHasInProgressDispatches(t *testing.T) {
	defer setupAgentCommandTestDB(t)()
	ctx := context.Background()

	cmd := &AgentCommand{Name: "n", Content: "echo"}
	if err := CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("create cmd: %v", err)
	}

	t.Run("none", func(t *testing.T) {
		has, slugs, err := HasInProgressDispatches(ctx, cmd.ID)
		if err != nil || has || len(slugs) != 0 {
			t.Errorf("got has=%v slugs=%v err=%v, want false/empty", has, slugs, err)
		}
	})

	// 创建一个 in_progress 的 dispatch
	dispatchSlug := GenerateAgentDispatchSlug()
	d := &AgentCommandDispatch{
		Slug:      dispatchSlug,
		CommandID: cmd.ID,
		Status:    AgentDispatchStatusInProgress,
		StartedAt: time.Now(),
	}
	if err := DB(ctx).Create(d).Error; err != nil {
		t.Fatalf("create dispatch: %v", err)
	}

	t.Run("has in_progress", func(t *testing.T) {
		has, slugs, err := HasInProgressDispatches(ctx, cmd.ID)
		if err != nil || !has || len(slugs) != 1 || slugs[0] != dispatchSlug {
			t.Errorf("got has=%v slugs=%v err=%v", has, slugs, err)
		}
	})

	// 推进到终态
	if err := DB(ctx).Model(d).Update("status", AgentDispatchStatusSuccess).Error; err != nil {
		t.Fatalf("update status: %v", err)
	}
	t.Run("after terminal", func(t *testing.T) {
		has, _, err := HasInProgressDispatches(ctx, cmd.ID)
		if err != nil || has {
			t.Errorf("after terminal has=%v err=%v, want false", has, err)
		}
	})

	// awaiting_confirmation 也算「未到终态」（用户尚未决定，dispatch 还活跃）
	if err := DB(ctx).Model(d).Update("status", AgentDispatchStatusAwaitingConfirmation).Error; err != nil {
		t.Fatalf("update status: %v", err)
	}
	t.Run("awaiting_confirmation counts as in_progress", func(t *testing.T) {
		has, slugs, err := HasInProgressDispatches(ctx, cmd.ID)
		if err != nil || !has || len(slugs) != 1 {
			t.Errorf("got has=%v slugs=%v err=%v, want true/[slug]", has, slugs, err)
		}
	})
}

func TestIsTerminalAgentTaskStatus(t *testing.T) {
	cases := map[string]bool{
		AgentTaskStatusPending:     false,
		AgentTaskStatusInProgress:  false,
		AgentTaskStatusSuccess:     true,
		AgentTaskStatusFailed:      true,
		AgentTaskStatusTimeout:     true,
		AgentTaskStatusUnreachable: true,
		"unknown":                  false,
	}
	for s, want := range cases {
		if got := IsTerminalAgentTaskStatus(s); got != want {
			t.Errorf("IsTerminalAgentTaskStatus(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestIsFailureAgentTaskStatus(t *testing.T) {
	cases := map[string]bool{
		AgentTaskStatusFailed:      true,
		AgentTaskStatusTimeout:     true,
		AgentTaskStatusUnreachable: true,
		AgentTaskStatusSuccess:     false,
		AgentTaskStatusPending:     false,
		AgentTaskStatusInProgress:  false,
	}
	for s, want := range cases {
		if got := IsFailureAgentTaskStatus(s); got != want {
			t.Errorf("IsFailureAgentTaskStatus(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestInvocation_IsTerminal(t *testing.T) {
	cases := map[string]bool{
		AgentInvocationStatusPending:    false,
		AgentInvocationStatusInProgress: false,
		AgentInvocationStatusSuccess:    true,
		AgentInvocationStatusPartial:    true,
		AgentInvocationStatusFailed:     true,
	}
	for s, want := range cases {
		inv := &AgentCommandInvocation{Status: s}
		if got := inv.IsTerminal(); got != want {
			t.Errorf("IsTerminal(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestBatchCommandExecutionStats(t *testing.T) {
	defer setupAgentCommandTestDB(t)()
	ctx := context.Background()

	cmd := &AgentCommand{Name: "n", Content: "echo"}
	if err := CreateAgentCommandWithSlugRetry(ctx, cmd, 5); err != nil {
		t.Fatalf("create cmd: %v", err)
	}
	// v2: 一次 dispatch = 1 行；ExecutedCount 直接看 dispatch 行数。
	d1 := GenerateAgentDispatchSlug()
	d2 := GenerateAgentDispatchSlug()
	now := time.Now()
	dispatches := []*AgentCommandDispatch{
		{Slug: d1, CommandID: cmd.ID, Status: AgentDispatchStatusSuccess, StartedAt: now},
		{Slug: d2, CommandID: cmd.ID, Status: AgentDispatchStatusFailed, StartedAt: now.Add(time.Second)},
	}
	for _, d := range dispatches {
		if err := DB(ctx).Create(d).Error; err != nil {
			t.Fatalf("create dispatch: %v", err)
		}
	}
	got, err := BatchCommandExecutionStats(ctx, []uint{cmd.ID})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	stat, ok := got[cmd.ID]
	if !ok {
		t.Fatal("missing stat for cmd")
	}
	if stat.ExecutedCount != 2 {
		t.Errorf("ExecutedCount = %d, want 2 (two distinct dispatches)", stat.ExecutedCount)
	}
}

func TestRandomLowerAlnum(t *testing.T) {
	if got := randomLowerAlnum(0); got != "" {
		t.Errorf("len 0 should be empty, got %q", got)
	}
	if got := randomLowerAlnum(16); len(got) != 16 {
		t.Errorf("len 16 expected, got %d", len(got))
	}
}

// TestValidateAgentCommandName_ChineseLength 6 个汉字（18 字节）+ 53 ASCII 共 59 字符。
// 修改前按字节算 71 > 60 报 name_too_long；修改后按字符 59 ≤ 60 通过。
func TestValidateAgentCommandName_ChineseLength(t *testing.T) {
	// 59 字符（6 中文 + 53 数字），71 字节
	name := "查看磁盘空间" + strings.Repeat("0", 53)
	if err := ValidateAgentCommandName(name); err != nil {
		t.Errorf("59 chars name (71 bytes) should pass under rune-based limit, got %v", err)
	}
	// 60 字符 = 上限
	name60 := "查看磁盘空间" + strings.Repeat("0", 54)
	if err := ValidateAgentCommandName(name60); err != nil {
		t.Errorf("60 chars name should pass at limit, got %v", err)
	}
	// 61 字符 = 越界
	name61 := "查看磁盘空间" + strings.Repeat("0", 55)
	if err := ValidateAgentCommandName(name61); err != ErrAgentCommandNameTooLong {
		t.Errorf("61 chars name should be too long, got %v", err)
	}
	// 纯中文 60 字
	allChinese := strings.Repeat("中", 60)
	if err := ValidateAgentCommandName(allChinese); err != nil {
		t.Errorf("60 Chinese chars (180 bytes) should pass, got %v", err)
	}
	allChinese61 := strings.Repeat("中", 61)
	if err := ValidateAgentCommandName(allChinese61); err != ErrAgentCommandNameTooLong {
		t.Errorf("61 Chinese chars should fail, got %v", err)
	}
}

// TestValidateAgentCommandContent_ChineseLength 内容也按字符计。
func TestValidateAgentCommandContent_ChineseLength(t *testing.T) {
	// 8192 中文字符（24576 字节，远超原来 8192 字节限制）
	content := strings.Repeat("中", AgentCommandContentMaxChars)
	if err := ValidateAgentCommandContent(content); err != nil {
		t.Errorf("8192 Chinese chars content should pass under rune-based limit, got %v", err)
	}
	// 越界 1 字符
	content2 := strings.Repeat("中", AgentCommandContentMaxChars+1)
	if err := ValidateAgentCommandContent(content2); err != ErrAgentCommandContentTooLong {
		t.Errorf("8193 Chinese chars should fail, got %v", err)
	}
}

// TestTruncateRunes 按字符截断且不在多字节中间切断。
func TestTruncateRunes(t *testing.T) {
	cases := []struct {
		in       string
		maxRunes int
		want     string
	}{
		{"", 5, ""},
		{"abc", 5, "abc"},
		{"abcdef", 3, "abc"},
		{"中文测试", 2, "中文"},          // 截 2 个中文 = 6 字节
		{"中文测试", 0, ""},            // 0 字符
		{"中文测试", 100, "中文测试"},      // 不动
		{"hello 中国", 7, "hello 中"}, // 混合截断
		{"中aaa", 3, "中aa"},         // 中文 + 部分英文
	}
	for _, c := range cases {
		got := TruncateRunes(c.in, c.maxRunes)
		if got != c.want {
			t.Errorf("TruncateRunes(%q, %d) = %q, want %q", c.in, c.maxRunes, got, c.want)
		}
	}
}

// TestValidateAgentCommandParams_DescTooLong 参数 description 超长直接报错（不再静默截断）。
func TestValidateAgentCommandParams_DescTooLong(t *testing.T) {
	// 200 个中文 = 上限，应通过
	at := strings.Repeat("中", AgentCommandParamDescMax)
	params := []AgentCommandParam{{Name: "x", Description: at}}
	if _, err := ValidateAgentCommandParams(params); err != nil {
		t.Errorf("at limit (%d chars) should pass, got %v", AgentCommandParamDescMax, err)
	}

	// 201 个中文 = 越界 1 字符，应报错
	over := strings.Repeat("中", AgentCommandParamDescMax+1)
	params2 := []AgentCommandParam{{Name: "x", Description: over}}
	offender, err := ValidateAgentCommandParams(params2)
	if err != ErrAgentCommandParamDescTooLong {
		t.Errorf("over limit should return ErrAgentCommandParamDescTooLong, got %v", err)
	}
	if offender != "x" {
		t.Errorf("offender = %q, want x", offender)
	}
	// description 字段不应被修改
	if params2[0].Description != over {
		t.Error("ValidateAgentCommandParams must not mutate description on error")
	}
}

// TestValidateAgentCommandParams_DefaultTooLong 参数 default 超长报错。
func TestValidateAgentCommandParams_DefaultTooLong(t *testing.T) {
	// 128 字符上限通过
	at := strings.Repeat("中", AgentCommandParamDefaultMax)
	if _, err := ValidateAgentCommandParams([]AgentCommandParam{{Name: "x", Default: at}}); err != nil {
		t.Errorf("at limit (%d chars) should pass, got %v", AgentCommandParamDefaultMax, err)
	}

	// 129 字符越界
	over := strings.Repeat("中", AgentCommandParamDefaultMax+1)
	offender, err := ValidateAgentCommandParams([]AgentCommandParam{{Name: "y", Default: over}})
	if err != ErrAgentCommandParamDefaultTooLong {
		t.Errorf("over limit should return ErrAgentCommandParamDefaultTooLong, got %v", err)
	}
	if offender != "y" {
		t.Errorf("offender = %q, want y", offender)
	}
}

// TestValidateAgentCommandParams_MaxItems 参数项数上限调整为 10，触发 ErrAgentCommandParamsTooMany。
func TestValidateAgentCommandParams_MaxItems(t *testing.T) {
	if AgentCommandParamsMax != 10 {
		t.Errorf("AgentCommandParamsMax = %d, want 10 (sanity check)", AgentCommandParamsMax)
	}
	// 10 项恰好上限，应通过
	good := make([]AgentCommandParam, AgentCommandParamsMax)
	for i := range good {
		good[i] = AgentCommandParam{Name: "p" + string(rune('A'+i))}
	}
	if _, err := ValidateAgentCommandParams(good); err != nil {
		t.Errorf("at limit (%d items) should pass, got %v", AgentCommandParamsMax, err)
	}
	// 11 项越界
	over := make([]AgentCommandParam, AgentCommandParamsMax+1)
	for i := range over {
		over[i] = AgentCommandParam{Name: "q" + string(rune('A'+i))}
	}
	if _, err := ValidateAgentCommandParams(over); err != ErrAgentCommandParamsTooMany {
		t.Errorf("over limit should return ErrAgentCommandParamsTooMany, got %v", err)
	}
}

// TestValidateAgentCommandParams_WorstCaseFitsColumn 验证 10 项参数在所有字段
// 跑满（包含 4 字节 utf8mb4 字符）时，序列化后字符数 ≤ 8192（agent_commands.params_json
// 的 varchar(8192) 容量），保证 DB 不会写入失败。
func TestValidateAgentCommandParams_WorstCaseFitsColumn(t *testing.T) {
	// 用一个 4-byte utf8mb4 字符（emoji 𝄞 = U+1D11E）做 worst case 字节数。
	const fourByteRune = "𝄞"
	const ColumnMaxChars = 8192 // agent_commands.params_json varchar(8192)

	params := make([]AgentCommandParam, AgentCommandParamsMax)
	for i := range params {
		params[i] = AgentCommandParam{
			Name:        "param_" + string(rune('A'+i)) + strings.Repeat("a", AgentCommandParamNameMax-8), // 32 字符 ASCII
			Default:     strings.Repeat(fourByteRune, AgentCommandParamDefaultMax),                        // 128 个 4-byte rune
			Description: strings.Repeat(fourByteRune, AgentCommandParamDescMax),                           // 200 个 4-byte rune
		}
	}
	if _, err := ValidateAgentCommandParams(params); err != nil {
		t.Fatalf("max-filled params should validate, got %v", err)
	}

	// 序列化（与 SetParams 同路径）
	b, err := jsonMarshalParams(params)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	chars := 0
	for range string(b) {
		chars++
	}
	t.Logf("worst-case params_json chars=%d bytes=%d", chars, len(b))
	if chars > ColumnMaxChars {
		t.Errorf("worst-case JSON chars=%d > column varchar(%d) — DB will reject INSERT",
			chars, ColumnMaxChars)
	}
}

// jsonMarshalParams 与 AgentCommand.SetParams 同等路径，单独可调便于测试。
func jsonMarshalParams(p []AgentCommandParam) ([]byte, error) {
	c := &AgentCommand{}
	if err := c.SetParams(p); err != nil {
		return nil, err
	}
	return []byte(c.ParamsJSON), nil
}

// TestValidateAgentCommandDescription 命令描述长度按字符校验，超出报错（无 in-place 修改）。
func TestValidateAgentCommandDescription(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want error
	}{
		{"empty ok", "", nil},
		{"english under limit", strings.Repeat("a", AgentCommandDescMaxChars), nil},
		{"english over limit", strings.Repeat("a", AgentCommandDescMaxChars+1), ErrAgentCommandDescTooLong},
		{"chinese at limit", strings.Repeat("中", AgentCommandDescMaxChars), nil},
		{"chinese over limit", strings.Repeat("中", AgentCommandDescMaxChars+1), ErrAgentCommandDescTooLong},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateAgentCommandDescription(c.in)
			if err != c.want {
				t.Errorf("err = %v, want %v", err, c.want)
			}
		})
	}
}
