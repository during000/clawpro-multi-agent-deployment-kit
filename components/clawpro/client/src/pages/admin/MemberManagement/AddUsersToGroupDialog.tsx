/**
 * AddUsersToGroupDialog - 「添加用户到组织 / 项目」共享弹窗
 *
 * 从 NodeContentPanel 内嵌弹窗抽取而来，供以下两处复用（内容与逻辑完全一致）：
 *   - 用户管理页-组织视图（NodeContentPanel）
 *   - 项目资产管理页（AssetPanel）
 *
 * 行为：表格勾选候选用户（已在当前节点的用户置灰 + Tooltip 提示），
 * 顶部 info 提示多组织规则，底部显示已选数量 / 清除选择，确认时回调所选用户 id。
 */
import { useMemo, useState } from "react";
import { Search, Info } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { StatusTag } from "@/components/ui/status-tag";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { UserOrg, UserGroup } from "./types";
import { getPrimaryDeptPath } from "./mock";

export interface AddUsersToGroupDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 目标节点名称（标题显示） */
  nodeName: string;
  /** 目标节点 id（判断用户是否已在其中） */
  nodeId: string;
  /** 全部候选用户 */
  allUsers: UserOrg[];
  /** 组织全集（用于组织 / 部门路径展示） */
  groups: UserGroup[];
  /** 是否显示「所属组织」（部门）列。默认 false */
  showDept?: boolean;
  /** 是否 OneID 模式（影响「当前组织」列过滤）。默认 false */
  hasOneid?: boolean;
  /** 节点称谓（用于文案，默认"组织"，项目场景传"项目"） */
  term?: string;
  /** 确认回调，返回所选用户 id 列表 */
  onConfirm: (userIds: string[]) => void;
}

export function AddUsersToGroupDialog({
  open,
  onOpenChange,
  nodeName,
  nodeId,
  allUsers,
  groups,
  showDept = false,
  hasOneid = false,
  term = "组织",
  onConfirm,
}: AddUsersToGroupDialogProps) {
  const [addSearch, setAddSearch] = useState("");
  const [addSelected, setAddSelected] = useState<string[]>([]);

  const groupMap = useMemo(() => new Map(groups.map(g => [g.id, g])), [groups]);

  const addFilteredUsers = useMemo(() => {
    let list = allUsers;
    if (addSearch.trim()) {
      const kw = addSearch.trim().toLowerCase();
      list = list.filter(
        u =>
          u.userId.toLowerCase().includes(kw) ||
          u.displayName.toLowerCase().includes(kw)
      );
    }
    return list;
  }, [allUsers, addSearch]);

  const reset = () => {
    setAddSearch("");
    setAddSelected([]);
  };

  const handleClose = () => {
    onOpenChange(false);
    reset();
  };

  const handleConfirm = () => {
    if (addSelected.length === 0) return;
    onConfirm(addSelected);
    onOpenChange(false);
    reset();
  };

  return (
    <Dialog
      open={open}
      onOpenChange={o => {
        if (!o) handleClose();
      }}
    >
      <DialogContent
        className="sm:max-w-[720px] max-h-[85vh] flex flex-col"
        onOpenAutoFocus={e => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>添加用户到「{nodeName}」</DialogTitle>
        </DialogHeader>
        {/* 多组织规则提示 */}
        <Alert variant="info">
          <Info />
          <AlertDescription>
            一个用户支持加入多个{term}，可按{term}设置不同的配置与权限
          </AlertDescription>
        </Alert>
        <div className="py-2 space-y-4 flex-1 min-h-0 flex flex-col">
          <div className="relative shrink-0">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#A3A3A3]" />
            <Input
              placeholder="搜索姓名或用户 ID..."
              value={addSearch}
              onChange={e => setAddSearch(e.target.value)}
              className="pl-9 bg-white border-[#e5e5e5]"
              autoFocus
            />
          </div>
          <div className="flex-1 min-h-0 overflow-y-auto border border-[#e5e5e5] rounded-[4px] bg-white">
            <Table density="compact" autoFixedColumns={false}>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="w-10"></TableHead>
                  <TableHead>员工</TableHead>
                  {showDept && <TableHead>所属组织</TableHead>}
                  <TableHead>当前{term}</TableHead>
                  <TableHead className="w-24">角色</TableHead>
                  <TableHead className="w-20">状态</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {addFilteredUsers.length === 0 ? (
                  <TableRow className="hover:bg-transparent">
                    <TableCell
                      colSpan={showDept ? 6 : 5}
                      className="text-center text-xs text-[#A3A3A3] py-6"
                    >
                      没有可添加的用户
                    </TableCell>
                  </TableRow>
                ) : (
                  addFilteredUsers.map(m => {
                    const isInCurrentGroup = m.groupIds.includes(nodeId);
                    const isDisabled = isInCurrentGroup;
                    // 部门：用户所有 oneid-dept 组织的完整路径（主部门排首位）
                    const deptPaths = showDept
                      ? m.groupIds
                          .filter(
                            gid => groupMap.get(gid)?.source === "oneid-dept"
                          )
                          .map(gid => ({
                            path: getPrimaryDeptPath(gid, groups),
                            isPrimary: gid === m.primaryGroupId,
                          }))
                          .sort((a, b) =>
                            a.isPrimary ? -1 : b.isPrimary ? 1 : 0
                          )
                          .map(d => d.path)
                      : [];
                    // 组织
                    const groupPaths = m.groupIds
                      .filter(gid => {
                        const g = groupMap.get(gid);
                        if (!g) return false;
                        if (term === "项目") return g.source === "project";
                        if (hasOneid) {
                          return showDept
                            ? g.source === "oneid-dept" ||
                                g.source === "oneid-group"
                            : g.source === "oneid-group";
                        }
                        return g.source === "manual";
                      })
                      .map(gid => getPrimaryDeptPath(gid, groups));
                    const tooltipText = isInCurrentGroup
                      ? `该用户已在当前${term}`
                      : "";
                    const isChecked =
                      isInCurrentGroup || addSelected.includes(m.userId);
                    const onToggle = () => {
                      if (isDisabled) return;
                      setAddSelected(prev =>
                        prev.includes(m.userId)
                          ? prev.filter(id => id !== m.userId)
                          : [...prev, m.userId]
                      );
                    };
                    const row = (
                      <TableRow
                        key={m.userId}
                        data-state={
                          isChecked && !isDisabled ? "selected" : undefined
                        }
                        onClick={onToggle}
                        className={
                          isDisabled
                            ? "opacity-50 cursor-not-allowed bg-[#FAFAFA] hover:bg-[var(--bg-grey-hover)]"
                            : "cursor-pointer"
                        }
                      >
                        <TableCell className="w-10">
                          <Checkbox
                            checked={isChecked}
                            disabled={isDisabled}
                            onCheckedChange={onToggle}
                            onClick={e => e.stopPropagation()}
                          />
                        </TableCell>
                        <TableCell>
                          <div className="text-[var(--text-body)]">
                            {m.displayName}
                          </div>
                          <div className="mt-0.5 text-xs text-[#737373]">
                            {m.userId}
                          </div>
                        </TableCell>
                        {showDept && (
                          <TableCell className="text-xs text-[#737373]">
                            {deptPaths.length > 0 ? deptPaths.join("、") : "—"}
                          </TableCell>
                        )}
                        <TableCell className="text-xs text-[#737373]">
                          {groupPaths.length > 0 ? groupPaths.join("、") : "—"}
                        </TableCell>
                        <TableCell className="w-24">
                          <StatusTag
                            mode="fill"
                            variant={m.role === "admin" ? "blue" : "gray"}
                          >
                            {m.role === "admin" ? "管理员" : "用户"}
                          </StatusTag>
                        </TableCell>
                        <TableCell className="w-20">
                          {m.status !== "disabled" ? (
                            <StatusTag mode="text" variant="green">
                              正常
                            </StatusTag>
                          ) : (
                            <StatusTag mode="text" variant="red">
                              禁用
                            </StatusTag>
                          )}
                        </TableCell>
                      </TableRow>
                    );
                    return isDisabled ? (
                      <Tooltip key={m.userId}>
                        <TooltipTrigger asChild>{row}</TooltipTrigger>
                        <TooltipContent>{tooltipText}</TooltipContent>
                      </Tooltip>
                    ) : (
                      row
                    );
                  })
                )}
              </TableBody>
            </Table>
          </div>
          {addSelected.length > 0 && (
            <div className="flex items-center justify-between">
              <span className="text-xs text-[#737373]">
                已选择 {addSelected.length} 名用户
              </span>
              <button
                className="text-xs text-[#355EF1] hover:text-[#1447E6] hover:underline"
                onClick={() => setAddSelected([])}
              >
                清除选择
              </button>
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            取消
          </Button>
          <Button
            variant="dialog-confirm"
            onClick={handleConfirm}
            disabled={addSelected.length === 0}
          >
            确认添加
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default AddUsersToGroupDialog;
