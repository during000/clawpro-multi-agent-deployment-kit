/**
 * SettingDistributeDialog - 团队设定 下发弹窗
 *
 * 与「技能下发」一致的交互：选择目标工具后下发。
 * 当前仅 Claude Code、CodeBuddy 支持下发；其余工具置灰并标注「暂不支持」。
 * 下发时，平台会把设定写入各工具各自的设定文件（CLAUDE.md / CODEBUDDY.md）。
 */
import { useEffect, useMemo, useState } from 'react';
import { Check } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { StatusTag } from '@/components/ui/status-tag';
import { BodyMedium, MetaText } from '@/components/ui/Typography';
import {
  type TeamSetting,
  DISTRIBUTE_TARGETS,
  getProjectName,
} from './settingsLibraryData';

interface SettingDistributeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  setting: TeamSetting | null;
  /** 确认下发回调，返回所选目标工具的 id 列表 */
  onConfirm: (targetIds: string[]) => void;
}

export default function SettingDistributeDialog({
  open,
  onOpenChange,
  setting,
  onConfirm,
}: SettingDistributeDialogProps) {
  const supportedIds = useMemo(
    () => DISTRIBUTE_TARGETS.filter((t) => t.supported).map((t) => t.id),
    [],
  );

  // 默认全选已支持的工具
  const [selected, setSelected] = useState<string[]>(supportedIds);

  useEffect(() => {
    if (open) setSelected(supportedIds);
  }, [open, supportedIds]);

  const toggle = (id: string) => {
    setSelected((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    );
  };

  const handleConfirm = () => {
    if (selected.length === 0) return;
    onConfirm(selected);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[520px]">
        <DialogHeader>
          <DialogTitle>下发团队设定</DialogTitle>
          <DialogDescription>
            选择要下发的目标工具，平台会将设定写入各工具各自的设定文件。
          </DialogDescription>
        </DialogHeader>

        {setting && (
          <div className="space-y-3">
            {/* 当前设定信息 */}
            <div className="flex items-center gap-2 rounded-[4px] border border-[var(--cp-border)] bg-[var(--bg-grey-hover-subtle)] px-3 py-2">
              <BodyMedium as="span" tone="primary" className="truncate">{setting.name}</BodyMedium>
              <StatusTag mode="fill" variant="gray" className="shrink-0">v{setting.version}</StatusTag>
              <MetaText as="span" tone="weak" className="ml-auto shrink-0 truncate">
                项目：{getProjectName(setting.projectId)}
              </MetaText>
            </div>

            {/* 目标工具列表 */}
            <div className="space-y-2">
              <MetaText as="p" tone="secondary">选择下发目标</MetaText>
              <div className="space-y-2">
                {DISTRIBUTE_TARGETS.map((t) => {
                  const checked = selected.includes(t.id);
                  return (
                    <label
                      key={t.id}
                      className={`flex items-center gap-3 rounded-[4px] border px-3 py-2.5 transition-colors ${
                        !t.supported
                          ? 'cursor-not-allowed border-[var(--cp-border)] bg-[#FAFAFA] opacity-70'
                          : checked
                            ? 'cursor-pointer border-[#C7D7FE] bg-[#E8ECFE]'
                            : 'cursor-pointer border-[var(--cp-border)] bg-[var(--cp-surface)] hover:border-[#1447E6]'
                      }`}
                    >
                      <Checkbox
                        checked={t.supported && checked}
                        disabled={!t.supported}
                        onCheckedChange={() => t.supported && toggle(t.id)}
                      />
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <BodyMedium as="span" tone={t.supported ? 'primary' : 'weak'}>{t.name}</BodyMedium>
                          {!t.supported && (
                            <StatusTag mode="soft" variant="gray" className="shrink-0">暂不支持</StatusTag>
                          )}
                        </div>
                        <MetaText as="p" tone="weak" className="font-mono">{t.file}</MetaText>
                      </div>
                      {t.supported && checked && (
                        <Check className="size-4 shrink-0 text-[#1447E6]" />
                      )}
                    </label>
                  );
                })}
              </div>
              <MetaText as="p" tone="weak" className="leading-relaxed">
                目前仅支持下发到 Claude Code、CodeBuddy，其余工具陆续接入中。
              </MetaText>
            </div>
          </div>
        )}

        <DialogFooter>
          <Button variant="claw-outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button variant="dialog-confirm" onClick={handleConfirm} disabled={selected.length === 0}>
            确认下发{selected.length > 0 ? `（${selected.length}）` : ''}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
