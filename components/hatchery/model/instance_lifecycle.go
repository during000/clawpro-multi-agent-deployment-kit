package model

// 操作类型常量
const (
	OpNone               = ""                     // 无操作
	OpCreate             = "create"               // 创建
	OpReboot             = "reboot"               // 重启
	OpReinstall          = "reinstall"            // 重装
	OpUpgrade            = "upgrade"              // 升级
	OpDelete             = "delete"               // 删除
	OpMigrate            = "migrate"              // 迁移
	OpAdjustInstanceType = "adjust_instance_type" // CVM 规格升配
	OpAdjustSystemDisk   = "adjust_system_disk"   // CVM 系统盘扩容

	// 本地 Agent 卸载（uninstall_teamai 任务下发后标记，避免复用 CVM 删除状态机）
	LocalAgentOpUninstall = "uninstall_local_agent"
)

// 操作状态常量
const (
	OpStateNone       = ""           // 无操作状态
	OpStateProcessing = "processing" // 处理中
	OpStateSuccess    = "success"    // 成功
	OpStateFailed     = "failed"     // 失败
)

// OpenClaw 状态常量
const (
	StatusCreating      = "creating"       // 创建中
	StatusCreateFailed  = "create_failed"  // 创建失败
	StatusRunning       = "running"        // 运行中
	StatusStopped       = "stopped"        // 已关机
	StatusLoading       = "loading"        // 加载中
	StatusLoadFailed    = "load_failed"    // 加载失败
	StatusMaintaining   = "maintaining"    // 维护中
	StatusPending       = "pending"        // 待处理
	StatusDestroying    = "destroying"     // 销毁中
	StatusDestroyed     = "destroyed"      // 已销毁
	StatusUpgrading     = "upgrading"      // 升级中
	StatusUpgradeFailed = "upgrade_failed" // 升级失败
)

// OpenClaw 状态定义
type OpenClawStatusDef struct {
	Status    string
	Label     string
	LabelEn   string
	Tooltip   string
	TooltipEn string
	Transient bool
	Actions   []string
}

// 用户端状态映射（状态 → 展示信息）
var UserStatusMap = map[string]OpenClawStatusDef{
	StatusCreating:      {StatusCreating, "创建中", "Creating", "正在创建中，请稍候", "Creating, please wait", true, []string{}},
	StatusCreateFailed:  {StatusCreateFailed, "创建失败", "Create failed", "创建失败，可删除后重新创建", "Create failed, please delete and try again", false, []string{"delete"}},
	StatusRunning:       {StatusRunning, "运行中", "Running", "", "", false, []string{"stop", "restart_gateway", "reboot", "reinstall", "delete", "terminal"}},
	StatusStopped:       {StatusStopped, "已关机", "Stopped", "已关机，可开机恢复", "Stopped, please start", false, []string{"start", "delete"}},
	StatusLoading:       {StatusLoading, "加载中", "Loading", "加载中，请稍候", "Loading, please wait", true, []string{}},
	StatusLoadFailed:    {StatusLoadFailed, "加载失败", "Loading failed", "加载失败，可点击重试恢复", "Loading failed, please retry", false, []string{"retry", "delete"}},
	StatusMaintaining:   {StatusMaintaining, "维护中", "Maintaining", "维护中，请稍候", "Maintaining, please wait", true, []string{"delete"}},
	StatusPending:       {StatusPending, "待处理", "Pending", "已停用，请联系管理员处理", "Pending, please contact the administrator", true, []string{}},
	StatusDestroying:    {StatusDestroying, "销毁中", "Destroying", "正在销毁中，请稍候", "Destroying, please wait", true, []string{}},
	StatusDestroyed:     {StatusDestroyed, "已销毁", "Destroyed", "", "", false, []string{"delete"}},
	StatusUpgrading:     {StatusUpgrading, "升级中", "Upgrading", "正在升级，请稍候", "Upgrading, please wait", true, []string{}},
	StatusUpgradeFailed: {StatusUpgradeFailed, "升级失败", "Upgrade failed", "升级失败，请重试或联系管理员", "Upgrade failed, please retry or contact the administrator", false, []string{}},
}

// 管控端状态映射（状态 → 管理员操作）
var AdminStatusMap = map[string]OpenClawStatusDef{
	StatusCreating:      {StatusCreating, "创建中", "Creating", "正在创建中，请稍候", "Creating, please wait", true, []string{"monitor"}},
	StatusCreateFailed:  {StatusCreateFailed, "创建失败", "Create failed", "创建失败，可删除后重新创建", "Create failed, please delete and try again", false, []string{"delete"}},
	StatusRunning:       {StatusRunning, "运行中", "Running", "", "", false, []string{"terminal", "stop", "delete", "restart_gateway", "reboot", "reinstall", "monitor"}},
	StatusStopped:       {StatusStopped, "已关机", "Stopped", "已关机，可开机恢复", "Stopped, please start", false, []string{"start", "delete", "reinstall", "monitor"}},
	StatusLoading:       {StatusLoading, "加载中", "Loading", "加载中，请稍候", "Loading, please wait", true, []string{"monitor"}},
	StatusLoadFailed:    {StatusLoadFailed, "加载失败", "Loading failed", "加载失败，可点击重试恢复", "Loading failed, please retry", false, []string{"delete", "monitor"}},
	StatusMaintaining:   {StatusMaintaining, "维护中", "Maintaining", "维护中，请稍候", "Maintaining, please wait", true, []string{"delete", "monitor"}},
	StatusPending:       {StatusPending, "待处理", "Pending", "已停用，需要处理", "Pending, please handle", true, []string{"monitor"}},
	StatusDestroying:    {StatusDestroying, "销毁中", "Destroying", "正在销毁中，请稍候", "Destroying, please wait", true, []string{"monitor"}},
	StatusDestroyed:     {StatusDestroyed, "已销毁", "Destroyed", "", "", false, []string{"delete"}},
	StatusUpgrading:     {StatusUpgrading, "升级中", "Upgrading", "正在升级，请稍候", "Upgrading, please wait", true, []string{"monitor"}},
	StatusUpgradeFailed: {StatusUpgradeFailed, "升级失败", "Upgrade failed", "升级失败，请重试或联系管理员", "Upgrade failed, please retry or contact the administrator", false, []string{"monitor"}},
}

// 操作超时阈值（秒）
var OperationTimeouts = map[string]int{
	OpCreate:             600,  // 创建：10 分钟
	OpReboot:             300,  // 重启：5 分钟
	OpReinstall:          900,  // 重装：15 分钟
	OpUpgrade:            2700, // 升级：45 分钟（备份+上传+重装+等待就绪+恢复+后置hook）
	OpDelete:             300,  // 删除：5 分钟
	OpMigrate:            1800, // 迁移：30 分钟（下载+解压+重启）
	OpAdjustInstanceType: 900,  // 规格升配：15 分钟
	OpAdjustSystemDisk:   900,  // 系统盘扩容：15 分钟
}

// OperationTransitStatus 操作 → 对应的过渡态映射，用于操作即时写 last_known_status。
// 供 UpdateInstanceCachedStatus 使用，确保各 handler 不重复硬编码。
var OperationTransitStatus = map[string]string{
	OpCreate:    StatusCreating,
	OpReboot:    StatusLoading,
	OpReinstall: StatusLoading,
	OpUpgrade:   StatusUpgrading,
	OpDelete:    StatusDestroying,
	OpMigrate:   StatusLoading,
}

// CVM 状态分类

// 过渡态：CVM 可能在此期间状态变化
var CVMTransientStates = map[string]bool{
	"PENDING":      true,
	"LAUNCHING":    true,
	"STOPPING":     true,
	"STARTING":     true,
	"REBOOTING":    true,
	"REINSTALLING": true,
	"SHUTDOWN":     true,
	"TERMINATING":  true,
}

// 稳定态：CVM 不会自动变化
var CVMStableStates = map[string]bool{
	"RUNNING":       true,
	"STOPPED":       true,
	"LAUNCH_FAILED": true,
}

// 热迁移态
// 热迁移态：实例仍在运行，映射为 running
var CVMLiveMigrateStates = map[string]bool{
	"ENTER_SERVICE_LIVE_MIGRATE": true, // 进入热迁移
	"SERVICE_LIVE_MIGRATE":       true, // 热迁移中
	"EXIT_SERVICE_LIVE_MIGRATE":  true, // 退出热迁移
}

// 平台限制态：实例被平台限制，映射为 pending
var CVMPlatformLimitStates = map[string]bool{
	"FREEZING":  true, // 冻结中
	"BANNING":   true, // 封禁中
	"CORRUPTED": true, // 已损坏
	// 注意：ISOLATING/ISOLATED 不再通过 InstanceState 判断，改用 RestrictState 字段
}

// 维护态（救援模式）：映射为 maintaining
var CVMRescueModeStates = map[string]bool{
	"ENTER_RESCUE_MODE": true, // 进入救援模式
	"RESCUE_MODE":       true, // 救援模式中
	"EXIT_RESCUE_MODE":  true, // 退出救援模式
}

// IsResourceAdjustmentOperation reports whether operation is managed by the
// dedicated CVM resource-adjustment worker instead of the generic lifecycle
// transition map.
func IsResourceAdjustmentOperation(operation string) bool {
	return operation == OpAdjustInstanceType || operation == OpAdjustSystemDisk
}
