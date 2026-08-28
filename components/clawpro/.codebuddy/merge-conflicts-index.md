# 合并冲突解决进度索引

- 操作：`git merge origin/main` 合并进 `feature/design-jingsujiang`
- 分叉点：`c752143c`
- 解决原则：功能以主干(origin/main)为准；新功能套用分支(HEAD)样式且样式以分支为准
- 模式：严格逐文件，每个文件解决后展示 diff 确认

## 状态图例
- [ ] 待处理  [~] 处理中  [x] 已解决并 add

## ① 构建产物（取主干）
- [x] client/public/__manus__/version.json  → 取主干版本号/时间戳

## ② 全局设计组件（样式以分支为准）
- [x] client/src/components/ui/button.tsx  → 保留分支新增 SmallIconStateButton
- [x] client/src/components/ui/checkbox.tsx  → 保留分支样式 + 补回 group 类(修复 indeterminate)
- [x] client/src/components/ScopePopover.tsx  → 保留分支 SegmentGroup + 合并主干滚动穿透修复
- [x] client/src/components/AdminNoticeBar.tsx  → 分支箭头图标+getAdminNotices；主干停服告警(useServiceStatus/RENEW_URL)+补 AlertTriangle 导入
- [x] client/src/components/SsoLoginDialog.tsx  → 整体取主干(新增账号/邮箱/重置密码视图)，phone表单样式待走查
- [x] client/src/pages/SsoLoginDemo.tsx  → 整体取主干(配合 SsoLoginDialog)
- [x] client/src/components/AdminLayout.tsx  → 取分支 renderNavItem+ADMIN_NAV_GROUPS 架构(导航数据在 config/adminNav)
- [x] client/src/components/TenantLayout.tsx  → 取主干注释+NotificationPanel 组件(分支主体已引用)

### 待确认事项
- AdminLayout 导航：分支 config/adminNav 缺主干的 2 个入口：
  - /admin/resource-management（资源管理）
  - /admin/knowledge-management（知识管理）
  分支另有 /admin/agent-template。是否补齐主干入口到分支导航配置？（待用户确认）

## ③ tenant 业务页面（功能主干+样式分支）
- [x] client/src/pages/tenant/MyOpenClaw.tsx  → B方案：扩展 AgentCard 公共组件(开关机/重命名/分组变更/移交/迁移)+用 AgentCard 渲染；header/创建按钮/空状态融合主干停服禁用
- [x] (衍生) client/src/components/agent/AgentCard.tsx  → 扩展可选 props，向后兼容
- [x] client/src/pages/tenant/OpenClawDetail.tsx  → 取分支 SegmentList 导航+mt-1容器；补回主干 backup(数据备份) 入口
- [x] client/src/pages/tenant/ModelQuota.tsx  → 块1取主干 HoverCard 导入；块2取主干(当前周期配额+HoverCard详情+无限制态)
- [x] client/src/pages/tenant/ToolsMcpPanel.tsx  → doAddMCP 取主干增强签名；Input 取分支 tenant 样式；取主干双按钮(确认但不重启/确认并重启)+套分支 tenant-primary

## ③ admin 业务页面（功能主干+样式分支）
（ImageManagement* / MemberManagement* / MemoryManagement* / SkillLibrary* / SessionManagement / OpenClawMonitor / OpsObservation / TokensMonitor / ModelConfig / PlatformPolicy / ResourceManagement / BasicInfo / BatchUpdateNotice / CloudDevManagement / AgentToolLibrary / SkillRolesTab / SecurityGroupManagement / SessionDetail）

## ④ Security/AIAgent 模块（功能主干+样式分支，约40文件）
- [x] 全部 client/src/pages/admin/Security/**（含 AIAgent/api/index/Logs）→ add/add 同源，批量取分支端(ours)：保留同源功能 + 分支设计系统UI规范化；编译无 error

## ⑥ 杂项
- [ ] client/src/lib/mockData.ts
