package common

// FixedSnapshot 承载当前进程应服务的单个租户快照（非 universe 模式）。
// 由 main() 启动时构造。非 universe 模式下始终不为 nil（SQLite 模式下 Identifier 为空）。
// Universe 模式下为 nil，表示需要从 Host 动态解析租户。
var FixedSnapshot *TenantSnapshot

// IsUniverseMode 返回当前进程是否以 universe 多租户模式运行。
// 判断依据：FixedSnapshot 为 nil 即 universe 模式。
func IsUniverseMode() bool {
	return FixedSnapshot == nil
}
