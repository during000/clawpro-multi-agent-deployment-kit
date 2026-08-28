/**
 * 用户查看配置抽屉
 * 展示用户的基本信息 + 归属组织 + 加法型资源 + 唯一型资源 + 最终生效值
 *
 * v2.0：数据模型改为 UserOrg { groupIds, primaryGroupId, primaryGroupValid }
 *   - 主部门：来自 primaryGroupId 对应的 oneid-dept 节点（全路径）
 *   - 兼任：groupIds 里剩余的 oneid-dept 节点
 *   - 用户组：groupIds 里的 oneid-group 节点
 *   - 自建组织：groupIds 里的 manual 节点
 */
import React from "react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { StatusTag } from "@/components/ui/status-tag";
import { CircleAlert } from "lucide-react";
import type { UserOrg, UserOverrideInfo, UserGroup } from "./types";
import {
  MOCK_EFFECTIVE_CONFIG,
  MOCK_GROUPS,
  getPrimaryDeptPath,
} from "./mock";

interface MemberDrawerProps {
  user: UserOrg | null;
  info?: UserOverrideInfo;
  open: boolean;
  onOpenChange: (v: boolean) => void;
  /** 可传入当前模式的组织集合（OneID 或 manual），默认为 MOCK_GROUPS */
  groups?: UserGroup[];
}

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="mb-5">
      <div className="text-xs font-semibold text-[#A3A3A3] uppercase tracking-wider mb-2">
        {title}
      </div>
      <div
        className="bg-white rounded-[4px] border border-[#e5e5e5] overflow-hidden"
      >
        {children}
      </div>
    </div>
  );
}

function Row({
  label,
  value,
  hint,
}: {
  label: string;
  value: React.ReactNode;
  hint?: React.ReactNode;
}) {
  return (
    <div className="flex items-start gap-3 px-4 py-3 border-b border-[#f5f5f5] last:border-b-0">
      <div className="w-20 shrink-0 text-sm text-[#737373]">{label}</div>
      <div className="flex-1 min-w-0 text-sm text-[#0A0A0A] break-words">
        {value}
      </div>
      {hint && (
        <div className="text-xs text-[#A3A3A3] shrink-0 tabular-nums">{hint}</div>
      )}
    </div>
  );
}

function getGroupPath(id: string, groups: UserGroup[]): string {
  const map = new Map(groups.map((g) => [g.id, g]));
  const chain: string[] = [];
  let cur = map.get(id);
  while (cur) {
    chain.unshift(cur.name);
    cur = cur.parentId ? map.get(cur.parentId) : undefined;
  }
  return chain.join(" / ");
}

export default function MemberDrawer({
  user,
  info,
  open,
  onOpenChange,
  groups = MOCK_GROUPS,
}: MemberDrawerProps) {
  if (!user) return null;

  const cfg = MOCK_EFFECTIVE_CONFIG[user.userId] ?? {};

  // 主部门（仅 OneID 模式）
  const primaryPath = getPrimaryDeptPath(user.primaryGroupId, groups);

  // 分离兼任部门 / 用户组 / 自建组织
  const groupMap = new Map(groups.map((g) => [g.id, g]));
  const secondaryDepts = user.groupIds
    .filter((gid) => {
      const g = groupMap.get(gid);
      return (
        g?.source === "oneid-dept" && gid !== user.primaryGroupId
      );
    })
    .map((gid) => getGroupPath(gid, groups));
  const oneidGroups = user.groupIds
    .filter((gid) => groupMap.get(gid)?.source === "oneid-group")
    .map((gid) => groupMap.get(gid)?.name ?? gid);
  const manualGroups = user.groupIds
    .filter((gid) => groupMap.get(gid)?.source === "manual")
    .map((gid) => groupMap.get(gid)?.name ?? gid);

  const hasConflict = info?.status === "groupConflict";
  const hasMissing = info?.status === "primaryDeptMissing";
  const isMultiGroup = user.groupIds.length >= 2;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="!w-[600px] !max-w-none p-0 flex flex-col"
      >
        <SheetHeader className="px-6 py-5 border-b border-[#e5e5e5]">
          <SheetTitle className="text-lg">
            {user.displayName}
            <span className="ml-2 text-sm font-normal text-[#737373]">
              {user.userId}
            </span>
          </SheetTitle>
          <div className="text-xs text-[#737373] mt-1">
            查看该用户的最终生效配置
          </div>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto px-6 py-5 bg-white">
          {(hasConflict || hasMissing) && (
            <Alert variant="warning" className="mb-5">
              <CircleAlert />
              <AlertDescription>
                {hasMissing
                  ? "该用户主部门无效（为空 / 已被删除），请在 OneID 侧修正。"
                  : "该用户存在唯一型资源的组织冲突，当前按「最新绑定」兜底，建议管理员显式裁决。"}
              </AlertDescription>
            </Alert>
          )}

          {/* 基本信息 */}
          <Section title="基本信息">
            {user.primaryGroupId !== null && (
              <Row
                label="主部门"
                value={
                  <div className="flex items-center gap-2">
                    <span>{primaryPath}</span>
                    {!user.primaryGroupValid && (
                      <StatusTag mode="text" variant="red">无效</StatusTag>
                    )}
                  </div>
                }
              />
            )}
            {secondaryDepts.length > 0 && (
              <Row
                label="兼任部门"
                value={
                  <div className="space-y-1">
                    {secondaryDepts.map((p, i) => (
                      <div key={i} className="text-sm text-[#334155]">
                        {p}
                      </div>
                    ))}
                  </div>
                }
              />
            )}
            <Row
              label="所在组织"
              value={
                oneidGroups.length + manualGroups.length === 0 ? (
                  <span className="text-[#A3A3A3]">未加入组织</span>
                ) : (
                  <div className="flex flex-wrap gap-1.5">
                    {oneidGroups.map((g) => (
                      <StatusTag key={"og-" + g} mode="fill" variant="blue">
                        {g}
                      </StatusTag>
                    ))}
                    {manualGroups.map((g) => (
                      <StatusTag key={"mg-" + g} mode="soft" variant="purple">
                        {g}
                      </StatusTag>
                    ))}
                  </div>
                )
              }
              hint={isMultiGroup ? "多归属" : undefined}
            />
          </Section>

          {/* 加法型资源 */}
          <Section title="加法型资源（并集）">
            <Row
              label="可见模型"
              value={
                (cfg.models ?? []).length === 0 ? (
                  <span className="text-[#A3A3A3]">—</span>
                ) : (
                  <div className="flex flex-wrap gap-1.5">
                    {(cfg.models ?? []).map((m) => (
                      <StatusTag key={m} mode="soft" variant="gray">{m}</StatusTag>
                    ))}
                  </div>
                )
              }
              hint={`${(cfg.models ?? []).length} 项`}
            />
            <Row
              label="可见通道"
              value={
                (cfg.channels ?? []).length === 0 ? (
                  <span className="text-[#A3A3A3]">—</span>
                ) : (
                  <div className="flex flex-wrap gap-1.5">
                    {(cfg.channels ?? []).map((c) => (
                      <StatusTag key={c} mode="soft" variant="gray">{c}</StatusTag>
                    ))}
                  </div>
                )
              }
              hint={`${(cfg.channels ?? []).length} 项`}
            />
            <Row
              label="技能"
              value={<span className="text-[#A3A3A3]">未配置</span>}
            />
            <Row
              label="工具"
              value={<span className="text-[#A3A3A3]">未配置</span>}
            />
          </Section>

          {/* 唯一型资源 */}
          <Section title="唯一型资源（按优先级取首个）">
            <Row
              label="安全组"
              value={
                <span className="font-mono text-xs text-[#0A0A0A]">
                  {cfg.securityGroup ?? "—"}
                </span>
              }
            />
            <Row
              label="VPC"
              value={
                <div className="flex items-center gap-2">
                  <span className="font-mono text-xs text-[#0A0A0A]">
                    {cfg.vpc ?? "—"}
                  </span>
                  {hasConflict && (
                    <StatusTag mode="text" variant="orange">未裁决</StatusTag>
                  )}
                </div>
              }
            />
            <Row
              label="记忆"
              value={
                <span className="font-mono text-xs text-[#0A0A0A]">
                  {cfg.memory ?? "—"}
                </span>
              }
            />
            <Row
              label="镜像"
              value={
                <span className="font-mono text-xs text-[#0A0A0A]">
                  {cfg.image ?? "—"}
                </span>
              }
            />
          </Section>
        </div>
      </SheetContent>
    </Sheet>
  );
}
