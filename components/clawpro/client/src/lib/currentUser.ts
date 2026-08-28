/**
 * currentUser - 当前登录用户与共享权限判定的统一工具
 *
 * NOTE：当前项目尚无真实登录态，用户端「当前登录用户」统一以此常量为准（原先散落在
 *       TenantLayout 等处的硬编码 alice@acompany.com 收敛到这里，避免口径不一致）。
 */

/** 当前登录用户邮箱（Mock 登录态，全站唯一口径） */
export const CURRENT_USER_EMAIL = "alice@acompany.com";

/** 判定「共享/归属」相关字段的最小结构（兼容 AgentItem / OpenClawItem） */
export interface ShareOwnershipLike {
  shareScope?: "private" | "shared" | string;
  /** 创建人邮箱 */
  creator?: string;
  /** 归属人邮箱（代建场景可能与 creator 不同；缺省时回退 creator） */
  owner?: string;
}

/**
 * 判断某个 Agent 对「当前用户」是否为「他人共享给我」。
 * 这是详情页只读、列表页操作受限的唯一判定依据。
 *
 * 规则：shareScope === "shared" 且 归属人/创建人 ≠ 当前用户。
 *  - 仅影响真正被共享过来的 Agent；自己创建后共享出去的仍可正常编辑。
 */
export function isSharedToMe(claw?: ShareOwnershipLike | null): boolean {
  if (!claw) return false;
  if (claw.shareScope !== "shared") return false;
  const ownerEmail = claw.owner ?? claw.creator;
  return !!ownerEmail && ownerEmail !== CURRENT_USER_EMAIL;
}
