/**
 * AgentAvatar - 按角色身份渲染头像
 *
 * 对齐 Figma node 517:4002「助手」组件集合。
 * 头像素材来自「agent 头像」目录，已包含完整的渐变背景+角色图片+圆形裁切效果。
 * 素材命名规则：Property 1=角色名.png → 项目内路径 /assets/avatars/avatar-{英文名}.png
 *
 * 角色与头像映射：
 *   ┌────────────┬──────────────────────┐
 *   │ 角色        │ 头像文件              │
 *   ├────────────┼──────────────────────┤
 *   │ 默认/通用助手│ avatar-default.png   │
 *   │ 设计师      │ avatar-designer.png  │
 *   │ 行业分析师  │ avatar-analyst.png   │
 *   │ 内容创作者  │ avatar-creator.png   │
 *   │ 开发工程师  │ avatar-developer.png │
 *   │ 项目经理    │ avatar-pm.png        │
 *   │ 办公能手    │ avatar-operator.png  │
 *   │ 运营        │ avatar-operator.png  │
 *   └────────────┴──────────────────────┘
 *
 * 更新头像时，直接替换 /public/assets/avatars/ 下对应文件即可。
 */

/** 角色名 → 头像路径 */
const ROLE_AVATAR: Record<string, string> = {
  默认: "/assets/avatars/avatar-default.png",
  通用助手: "/assets/avatars/avatar-default.png",
  设计师: "/assets/avatars/avatar-designer.png",
  行业分析师: "/assets/avatars/avatar-analyst.png",
  内容创作者: "/assets/avatars/avatar-creator.png",
  开发工程师: "/assets/avatars/avatar-developer.png",
  项目经理: "/assets/avatars/avatar-pm.png",
  办公能手: "/assets/avatars/avatar-operator.png",
  运营: "/assets/avatars/avatar-operator.png",
};

/** 业务侧角色别名 → 标准角色名 */
const ROLE_ALIAS: Record<string, keyof typeof ROLE_AVATAR> = {
  数据分析师: "行业分析师",
  数据分析: "行业分析师",
  数据师: "行业分析师",
  初始角色: "通用助手",
  程序员: "开发工程师",
  理财助理: "行业分析师",
  美食家: "内容创作者",
  运营助手: "运营",
  产品经理: "项目经理",
  客服助手: "通用助手",
  技术顾问: "开发工程师",
  文案创作: "内容创作者",
  开发: "开发工程师",
  报告: "行业分析师",
  报告生成: "行业分析师",
};

/** 默认兜底：通用助手 */
const DEFAULT_ROLE: keyof typeof ROLE_AVATAR = "通用助手";

interface AgentAvatarProps {
  /** 角色名，如「设计师」「通用助手」 */
  roleName?: string;
  /** Agent 名称（仅用于 aria-label 兜底） */
  agentName?: string;
  /** 尺寸，默认 48 */
  size?: number;
  /** 是否灰显（停用/失败/已关机） */
  grayed?: boolean;
  className?: string;
}

export const AgentAvatar = ({
  roleName,
  agentName,
  size = 48,
  grayed = false,
  className = "",
}: AgentAvatarProps) => {
  // 解析角色名 → 头像路径
  const resolvedRole: keyof typeof ROLE_AVATAR = roleName
    ? ROLE_AVATAR[roleName]
      ? (roleName as keyof typeof ROLE_AVATAR)
      : ROLE_ALIAS[roleName] ?? DEFAULT_ROLE
    : DEFAULT_ROLE;
  const src = ROLE_AVATAR[resolvedRole];

  return (
    <img
      src={src}
      alt=""
      aria-label={roleName || agentName || "Agent 头像"}
      draggable={false}
      className={`flex-shrink-0 transition-opacity ${
        grayed ? "opacity-40" : ""
      } ${className}`}
      style={{
        width: size,
        height: size,
        borderRadius: "50%",
        objectFit: "cover",
        pointerEvents: "none",
        userSelect: "none",
      }}
    />
  );
};
