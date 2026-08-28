/**
 * 同步结果弹窗
 *
 * 展示同步后检测到的异常：
 *   1. 组织异常（上方）：被删除的组织架构仍有配置绑定
 *   2. 用户异常（下方）：主部门失效等
 */
import React from "react";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import { StatusTag } from "@/components/ui/status-tag";
import { CircleAlert, FolderX } from "lucide-react";
import { BodyMedium, HelperText } from "@/components/ui/Typography";
import type { SyncResult } from "./types";

interface SyncResultDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  result: SyncResult | null;
  onConfirm: () => void;
}

export default function SyncResultDialog({
  open,
  onOpenChange,
  result,
  onConfirm,
}: SyncResultDialogProps) {
  if (!result) return null;

  const hasGroupAnomalies = result.anomalousGroups.length > 0;
  const hasUserAnomalies = result.anomalousUsers.length > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[720px]">
        <DialogHeader>
          <DialogTitle>
            同步结果
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2 max-h-[60vh] overflow-y-auto">
          {/* 组织异常区块 */}
          {hasGroupAnomalies && (
            <div>
              <div className="flex items-center gap-2 mb-2.5">
                <FolderX className="w-4 h-4 text-red-500" />
                <BodyMedium as="h4" tone="primary">
                  组织异常
                </BodyMedium>
                <HelperText as="span" className="tabular-nums">
                  ({result.anomalousGroups.length})
                </HelperText>
              </div>

              <Alert variant="warning" className="mb-3">
                <CircleAlert />
                <AlertDescription>
                  以下组织对应的部门已在腾讯统一身份管理平台被删除，组织内用户已被移出。由于组织仍有正在应用的配置，需管理员将配置与组织解绑后，组织才会被彻底删除。
                </AlertDescription>
              </Alert>

              {/* 组织异常表格 */}
              <div className="border border-[#e5e5e5] rounded-[4px] overflow-hidden">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>组织名称</TableHead>
                      <TableHead className="text-center">组织总人数</TableHead>
                      <TableHead>已应用配置</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {result.anomalousGroups.map((group) => (
                      <TableRow key={group.groupId}>
                        <TableCell className="font-medium">
                          {group.groupName}
                        </TableCell>
                        <TableCell className="text-center tabular-nums text-[#737373]">
                          {group.memberCount}
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-wrap gap-1.5">
                            {group.boundConfigs.map((config) => (
                              <StatusTag key={config} mode="soft" variant="red">
                                {config}
                              </StatusTag>
                            ))}
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          )}

          {/* 用户异常区块 */}
          {hasUserAnomalies && (
            <div>
              <Alert variant="warning" className="mb-3">
                <CircleAlert />
                <AlertTitle>用户异常</AlertTitle>
                <AlertDescription>
                  以下用户的主部门在腾讯统一身份管理平台已失效，需管理员关注其配置生效状态。
                </AlertDescription>
              </Alert>

              {/* 用户异常表格 */}
              <div className="border border-[#e5e5e5] rounded-[4px] overflow-hidden">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>用户</TableHead>
                      <TableHead>异常原因</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {result.anomalousUsers.map((user) => (
                      <TableRow key={user.userId}>
                        <TableCell className="font-medium">
                          {user.displayName}
                        </TableCell>
                        <TableCell className="text-[#737373]">
                          {user.reason}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="dialog-confirm"
            onClick={() => {
              onConfirm();
              onOpenChange(false);
            }}
          >
            我知道了
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
