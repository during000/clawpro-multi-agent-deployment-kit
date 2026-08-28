/**
 * RemoveUserFromGroupDialog - 「从组织 / 项目中移除用户」共享弹窗
 *
 * 从 NodeContentPanel 内嵌弹窗抽取而来，供以下两处复用（内容与逻辑完全一致）：
 *   - 用户管理页-组织视图 / 项目视图（NodeContentPanel）
 *   - 项目资产管理页（AssetPanel）
 *
 * 行为：
 *   1. 移除确认弹窗：warning Alert + 用户 ID / 组织(项目)名称信息卡；
 *   2. 组织场景下，若该用户在当前节点存在存量 Agent 实例，确认后弹出「存量 Agent 实例处理」二次弹窗
 *      （保留原配置 / 删除实例）；项目场景不处理存量实例，直接移除。
 *
 * 视觉规范：遵循 ClawPro 可移植设计规范（管控端）：信息卡 4px 圆角、危险确认按钮 variant="destructive"。
 */
import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { BodyText, MetaMedium } from "@/components/ui/Typography";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui/table";
import { CircleAlert } from "lucide-react";
import type { UserGroup } from "./types";
import { getPrimaryDeptPath, MOCK_USER_GROUP_AGENTS } from "./mock";

export interface RemoveUserFromGroupDialogProps {
  /** 待移除用户 id（为 null 时弹窗关闭） */
  userId: string | null;
  /** 目标节点 id（用于检测存量实例） */
  nodeId: string;
  /** 目标节点名称（信息卡展示） */
  nodeName: string;
  /** 组织全集（用于展示节点路径） */
  groups: UserGroup[];
  /** 是否项目场景（项目：文案用"项目"，且不处理存量 Agent 实例） */
  isProject?: boolean;
  /** 关闭弹窗 */
  onClose: () => void;
  /** 确认移除回调 */
  onConfirm: (userId: string) => void;
}

export function RemoveUserFromGroupDialog({
  userId,
  nodeId,
  nodeName,
  groups,
  isProject = false,
  onClose,
  onConfirm,
}: RemoveUserFromGroupDialogProps) {
  const term = isProject ? "项目" : "组织";

  // 存量 Agent 实例处理弹窗（仅组织场景）
  const [agentInstanceDialog, setAgentInstanceDialog] = useState<{
    userId: string;
    groupName: string;
    instances: Array<{ id: string; name: string }>;
  } | null>(null);
  const [agentInstanceChoice, setAgentInstanceChoice] = useState<"keep" | "delete">("keep");

  const handleRemoveConfirm = () => {
    if (!userId) return;
    // 检测该用户在当前节点是否有 Agent 实例
    const userAgents = MOCK_USER_GROUP_AGENTS[userId];
    const instances = userAgents?.[nodeId] ?? [];

    if (instances.length > 0 && !isProject) {
      // 有存量实例，弹出二次确认（仅组织场景；项目场景不处理存量实例，直接移除）
      setAgentInstanceDialog({
        userId,
        groupName: getPrimaryDeptPath(nodeId, groups),
        instances,
      });
      onClose();
    } else {
      onConfirm(userId);
      onClose();
    }
  };

  return (
    <>
      {/* 从组织 / 项目中移除确认弹窗 */}
      <Dialog
        open={!!userId}
        onOpenChange={(open) => {
          if (!open) onClose();
        }}
      >
        <DialogContent
          className="sm:max-w-[560px]"
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>从{term}中移除</DialogTitle>
          </DialogHeader>
          <div className="py-2 space-y-4">
            {/* 警示提示 - Alert 位于内容区最上方 */}
            <Alert variant="warning">
              <CircleAlert />
              <AlertDescription>
                移除后，该用户在此{term}下的可见范围和权限将被收回。用户不会被删除，
                <span className="font-medium">仅解除与该{term}的关联</span>。
              </AlertDescription>
            </Alert>

            {/* 信息卡 - 规范样式：白底 #E5E5E5 边 4px 圆角，label #525252 / value #0A0A0A */}
            <div className="rounded-[4px] border border-[#e5e5e5] bg-white px-4 py-3 space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-[#525252]">用户 ID</span>
                <span className="text-sm font-medium text-[#0A0A0A]">{userId}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-[#525252]">{term}名称</span>
                <span className="text-sm font-medium text-[#0A0A0A]">{nodeName}</span>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={onClose}>
              取消
            </Button>
            <Button variant="destructive" onClick={handleRemoveConfirm}>
              确认移除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 存量 Agent 实例处理弹窗（仅组织场景） */}
      <Dialog
        open={!!agentInstanceDialog}
        onOpenChange={(open) => {
          if (!open) setAgentInstanceDialog(null);
        }}
      >
        <DialogContent className="sm:max-w-[720px]" onOpenAutoFocus={(e) => e.preventDefault()}>
          <DialogHeader>
            <DialogTitle>存量 Agent 实例处理</DialogTitle>
          </DialogHeader>
          <div className="py-2 space-y-3">
            <BodyText as="p" tone="secondary">
              用户在该组织下创建了 Agent 实例，用户已从该组织中移除，请选择如何处理存量实例：
            </BodyText>
            <div className="rounded-[4px] border border-[#e5e5e5] overflow-hidden">
              <Table density="compact">
                <TableHeader>
                  <TableRow>
                    <TableHead>用户 ID</TableHead>
                    <TableHead>Agent 实例名称 / ID</TableHead>
                    <TableHead>组织</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {agentInstanceDialog?.instances.map((inst) => (
                    <TableRow key={inst.id}>
                      <TableCell>{agentInstanceDialog.userId}</TableCell>
                      <TableCell>
                        {inst.name}
                        <span className="text-[#A3A3A3] ml-1">({inst.id})</span>
                      </TableCell>
                      <TableCell>{agentInstanceDialog.groupName}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>
          <div className="py-2 space-y-2">
            <MetaMedium as="p" tone="secondary" className="mb-1">
              处理方式
            </MetaMedium>
            {[
              {
                value: "keep",
                title: "保留原配置",
                desc: "存量 Agent 实例保留在原组织名下，可继续使用原组织的配置和权限，但无法在原组织创建新的 Agent",
              },
              {
                value: "delete",
                title: "删除实例",
                desc: "确认后将跳转到 Agent 列表页面，系统会帮您自动筛选出这些实例，您可以全选并批量删除",
              },
            ].map((opt) => (
              <label
                key={opt.value}
                className={`flex items-start gap-2.5 p-3 rounded-[4px] border cursor-pointer transition-colors ${
                  agentInstanceChoice === opt.value
                    ? "border-[#1447E6] bg-[#F0F3FC]"
                    : "border-[#e5e5e5] hover:border-[#A3A3A3]"
                }`}
                onClick={() => setAgentInstanceChoice(opt.value as "keep" | "delete")}
              >
                <span
                  className={`mt-0.5 w-4 h-4 rounded-full border-2 flex items-center justify-center shrink-0 ${
                    agentInstanceChoice === opt.value ? "border-[#1447E6]" : "border-[#C8CFDA]"
                  }`}
                >
                  {agentInstanceChoice === opt.value && (
                    <span className="w-2 h-2 rounded-full bg-[#355EF1]" />
                  )}
                </span>
                <div>
                  <p className="text-sm font-medium text-[#0A0A0A]">{opt.title}</p>
                  <p className="text-xs text-[#737373] mt-0.5">{opt.desc}</p>
                </div>
              </label>
            ))}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAgentInstanceDialog(null)}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              onClick={() => {
                if (agentInstanceDialog) {
                  onConfirm(agentInstanceDialog.userId);
                }
                const ids = agentInstanceDialog?.instances.map((i) => i.id).join(",") ?? "";
                setAgentInstanceDialog(null);
                if (agentInstanceChoice === "delete") {
                  window.location.href = `/admin/openclaw-monitor?filter=pending-delete&ids=${ids}`;
                }
              }}
            >
              确认
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

export default RemoveUserFromGroupDialog;
