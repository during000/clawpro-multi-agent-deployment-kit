import { useEffect, useState } from "react";
import {
  ArrowRight,
  Check,
  CheckCircle2,
  Database,
  Loader2,
  MessageSquareText,
  PackageCheck,
  ServerCog,
  Sparkles,
  UserRound,
  Wrench,
} from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

type MigrationStage = "setup" | "migrating" | "success";
export type MigratableAgentType = "openclaw" | "hermes";

const MIGRATION_ASSET_COUNT = 6;

const RUNTIME_META: Record<
  MigratableAgentType,
  { label: string; detailId: string }
> = {
  openclaw: { label: "OpenClaw", detailId: "oc-003" },
  hermes: { label: "Hermes", detailId: "oc-010" },
};

const TARGETS_BY_TYPE: Record<
  MigratableAgentType,
  Array<{
    id: string;
    name: string;
    instanceId: string;
    version: string;
    status: string;
    recommended: boolean;
    detailId: string;
  }>
> = {
  hermes: [
    {
      id: "hermes-prod-01",
      name: "Hermes 生产助手",
      instanceId: "ins-hermes01",
      version: "v0.12.0",
      status: "运行中",
      recommended: true,
      detailId: "oc-010",
    },
    {
      id: "hermes-test-02",
      name: "Hermes 测试环境",
      instanceId: "ins-hermes02",
      version: "v0.12.0",
      status: "运行中",
      recommended: false,
      detailId: "oc-011",
    },
  ],
  openclaw: [
    {
      id: "openclaw-prod-01",
      name: "OpenClaw 生产助手",
      instanceId: "ins-running01",
      version: "2026.4.23",
      status: "运行中",
      recommended: true,
      detailId: "oc-003",
    },
    {
      id: "openclaw-team-02",
      name: "OpenClaw 团队助手",
      instanceId: "ins-grpdemo02",
      version: "2026.4.23",
      status: "运行中",
      recommended: false,
      detailId: "oc-015",
    },
  ],
};

export function AgentTypeMigrationDialog({
  open,
  onOpenChange,
  sourceName,
  sourceInstanceId,
  sourceType,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sourceName: string;
  sourceInstanceId: string;
  sourceType: MigratableAgentType;
}) {
  const targetType: MigratableAgentType =
    sourceType === "openclaw" ? "hermes" : "openclaw";
  const targetLabel = RUNTIME_META[targetType].label;
  const targets = TARGETS_BY_TYPE[targetType];
  const [stage, setStage] = useState<MigrationStage>("setup");
  const [targetId, setTargetId] = useState(targets[0].id);
  const [includeSecrets, setIncludeSecrets] = useState(false);
  const [migrationIndex, setMigrationIndex] = useState(0);
  const target = targets.find(item => item.id === targetId) ?? targets[0];
  const migrationSteps = [
    {
      label: "基础配置",
      detail: "转换运行参数与偏好设置",
      icon: Wrench,
    },
    {
      label: "模型配置",
      detail: "转换供应商、模型与路由策略",
      icon: Sparkles,
    },
    {
      label: "技能",
      detail: "迁移已安装技能及配置",
      icon: PackageCheck,
    },
    {
      label: "记忆",
      detail: "迁移长期记忆与知识上下文",
      icon: Database,
    },
    {
      label: "会话历史",
      detail: "迁移历史对话与附件索引",
      icon: MessageSquareText,
    },
    {
      label: "角色设定",
      detail: "迁移人设、提示词与行为偏好",
      icon: UserRound,
    },
    {
      label: "目标实例验证",
      detail: `应用配置并检查 ${target.name}`,
      icon: ServerCog,
    },
  ];
  const currentMigrationStep =
    migrationIndex < migrationSteps.length
      ? migrationSteps[migrationIndex]
      : null;
  const migrationProgress = Math.min(
    100,
    Math.round(((migrationIndex + 1) / migrationSteps.length) * 100)
  );
  const CurrentMigrationIcon = currentMigrationStep?.icon ?? CheckCircle2;

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen && stage === "migrating") {
      toast("迁移将在后台继续，完成后会通知你");
    }
    onOpenChange(nextOpen);
  };

  useEffect(() => {
    if (!open) return;
    setStage("setup");
    setTargetId(targets[0].id);
    setIncludeSecrets(false);
    setMigrationIndex(0);
  }, [open, targets]);

  useEffect(() => {
    if (stage !== "migrating") return;
    if (migrationIndex >= migrationSteps.length) {
      const timer = window.setTimeout(() => setStage("success"), 520);
      return () => window.clearTimeout(timer);
    }
    const timer = window.setTimeout(
      () => setMigrationIndex(index => index + 1),
      560
    );
    return () => window.clearTimeout(timer);
  }, [stage, migrationIndex, migrationSteps.length]);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        size="lg"
        className="max-h-[92vh] overflow-hidden"
        data-testid="agent-type-migration-dialog"
      >
        <DialogHeader className="border-b border-[var(--border)] px-6 pb-5 pt-6">
          <DialogTitle className="text-lg">迁移到 {targetLabel}</DialogTitle>
          <DialogDescription>
            将 {sourceName} 的配置、技能和记忆等内容复制并转换到现有{" "}
            {targetLabel} 实例，源实例数据不受影响。
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="max-h-[calc(92vh-150px)] overflow-y-auto px-6 py-5">
          {stage === "setup" && (
            <div className="space-y-4">
              <section
                aria-label={`从 ${sourceName}（${sourceInstanceId}）选择迁移目标`}
              >
                <div className="mb-3">
                  <h3 className="text-sm font-semibold text-[var(--text-title)]">
                    选择目标实例
                  </h3>
                  <p className="mt-1 text-xs text-[var(--text-muted)]">
                    仅展示可迁移的 {targetLabel} 实例
                  </p>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  {targets.map(item => {
                    const selected = targetId === item.id;
                    return (
                      <button
                        key={item.id}
                        type="button"
                        onClick={() => setTargetId(item.id)}
                        className={`rounded-[var(--radius-lg)] border p-3 text-left transition-colors ${selected ? "border-[#355EF1] bg-[#F5F8FF] shadow-[0_0_0_1px_rgba(53,94,241,0.12)]" : "border-[var(--border)] bg-white hover:border-[#9DB4FF]"}`}
                        data-testid={`migration-target-${item.id}`}
                      >
                        <div className="flex items-start justify-between gap-3">
                          <div>
                            <div className="flex items-center gap-2">
                              <span className="text-sm font-medium text-[var(--text-title)]">
                                {item.name}
                              </span>
                              {item.recommended && (
                                <Badge
                                  variant="secondary"
                                  className="rounded-sm text-[10px]"
                                >
                                  推荐
                                </Badge>
                              )}
                            </div>
                            <div className="mt-1.5 text-xs text-[var(--text-muted)]">
                              {item.instanceId} · {targetLabel} {item.version}
                            </div>
                          </div>
                          <div
                            className={`flex size-4 items-center justify-center rounded-full border ${selected ? "border-[#355EF1] bg-[#355EF1] text-white" : "border-[var(--border-control)]"}`}
                          >
                            {selected && (
                              <Check className="size-3" strokeWidth={3} />
                            )}
                          </div>
                        </div>
                        <div className="mt-2 flex items-center gap-1.5 text-xs text-[var(--text-success)]">
                          <span className="size-1.5 rounded-full bg-[var(--text-success)]" />
                          {item.status} · 迁移环境检查通过
                        </div>
                      </button>
                    );
                  })}
                </div>
              </section>

              <div className="flex items-start gap-3 rounded-[var(--radius-lg)] bg-[var(--bg-grey-light,#FAFAFB)] px-4 py-3">
                <PackageCheck className="mt-0.5 size-4 shrink-0 text-[var(--text-brand)]" />
                <div>
                  <p className="text-sm text-[var(--text-title)]">
                    将迁移配置、模型、技能等 {MIGRATION_ASSET_COUNT} 项内容
                  </p>
                  <p className="mt-1 text-xs leading-5 text-[var(--text-muted)]">
                    ClawPro 将自动完成迁移，并在完成后验证目标实例状态。
                  </p>
                </div>
              </div>

              <label className="flex cursor-pointer items-start gap-3 px-1 py-1">
                <Checkbox
                  checked={includeSecrets}
                  onCheckedChange={checked =>
                    setIncludeSecrets(checked === true)
                  }
                  aria-label="同时迁移密钥和登录凭证"
                />
                <span>
                  <span className="block text-sm font-medium text-[var(--text-title)]">
                    同时迁移密钥和登录凭证
                  </span>
                  <span className="mt-1 block text-xs text-[var(--text-muted)]">
                    敏感信息将加密传输，不会在页面或普通日志中展示。
                  </span>
                </span>
              </label>
            </div>
          )}

          {stage === "migrating" && (
            <div className="mx-auto max-w-[620px] py-6">
              <div className="text-center">
                <div className="mx-auto mb-3 flex size-12 items-center justify-center rounded-full bg-[#EDF2FF] text-[var(--text-brand)]">
                  <Loader2 className="size-6 animate-spin" />
                </div>
                <h3 className="text-base font-semibold text-[var(--text-title)]">
                  正在迁移到 {target.name}
                </h3>
                <p className="mt-1 text-sm text-[var(--text-muted)]">
                  ClawPro 正在自动迁移并转换 Agent 内容
                </p>
              </div>

              <div className="mt-5">
                <div className="mb-2 flex items-center justify-between text-xs text-[var(--text-muted)]">
                  <span>迁移进度</span>
                  <span>{migrationProgress}%</span>
                </div>
                <div className="h-1.5 overflow-hidden rounded-full bg-[var(--bg-grey-light,#F1F3F5)]">
                  <div
                    className="h-full rounded-full bg-[var(--text-brand)] transition-all duration-500"
                    style={{ width: `${migrationProgress}%` }}
                  />
                </div>
              </div>

              <div className="mt-5 overflow-hidden rounded-[var(--radius-card)] border border-[var(--border)] bg-white p-4">
                <div className="text-xs text-[var(--text-muted)]">正在处理</div>
                <div className="mt-2 min-h-[64px] overflow-hidden">
                  {currentMigrationStep ? (
                    <div
                      key={currentMigrationStep.label}
                      className="flex animate-in items-center gap-3 rounded-[var(--radius-lg)] bg-[#F5F8FF] px-3 py-3 fade-in slide-in-from-bottom-2 duration-300"
                    >
                      <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-white text-[var(--text-brand)] shadow-sm">
                        <CurrentMigrationIcon className="size-4.5" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="text-sm font-medium text-[var(--text-title)]">
                          {currentMigrationStep.label}
                        </div>
                        <div className="mt-0.5 text-xs text-[var(--text-muted)]">
                          {currentMigrationStep.detail}
                        </div>
                      </div>
                      <Loader2 className="size-4 shrink-0 animate-spin text-[var(--text-brand)]" />
                    </div>
                  ) : (
                    <div className="flex items-center gap-3 rounded-[var(--radius-lg)] bg-[#F3FBF7] px-3 py-3 text-[var(--text-success)]">
                      <CheckCircle2 className="size-5" />
                      <span className="text-sm font-medium">
                        迁移内容处理完成，正在生成结果
                      </span>
                    </div>
                  )}
                </div>
              </div>

              <p className="mt-5 text-center text-xs text-[var(--text-muted)]">
                可关闭弹窗，迁移任务会在后台继续执行
              </p>
            </div>
          )}

          {stage === "success" && (
            <div className="mx-auto max-w-[560px] py-3">
              <div className="text-center">
                <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-[#E8F7EF] text-[var(--text-success)]">
                  <Check className="size-6" strokeWidth={3} />
                </div>
                <h3 className="mt-3 text-lg font-semibold text-[var(--text-title)]">
                  迁移完成
                </h3>
                <p className="mt-1 text-sm text-[var(--text-secondary)]">
                  Agent 内容已成功迁移到 {target.name}
                </p>
              </div>

              <div className="mt-5 overflow-hidden rounded-[var(--radius-card)] border border-[var(--border)] bg-white">
                <div className="flex items-center gap-3 px-4 py-3.5">
                  <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-[#EDF2FF] text-[var(--text-brand)]">
                    <ServerCog className="size-4.5" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium text-[var(--text-title)]">
                      {target.name}
                    </div>
                    <div className="mt-0.5 text-xs text-[var(--text-muted)]">
                      {target.instanceId} · {targetLabel} {target.version}
                    </div>
                  </div>
                  <span className="text-xs text-[var(--text-success)]">
                    {target.status}
                  </span>
                </div>

                <div className="flex items-start gap-3 border-t border-[var(--border)] bg-[var(--bg-grey-light,#FAFAFB)] px-4 py-3.5">
                  <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-[var(--text-success)]" />
                  <div>
                    <div className="text-sm font-medium text-[var(--text-title)]">
                      目标实例验证通过
                    </div>
                    <div className="mt-1 text-xs leading-5 text-[var(--text-muted)]">
                      实例运行正常，可前往目标实例查看迁移内容
                    </div>
                  </div>
                </div>
              </div>

              <p className="mt-4 rounded-[var(--radius-lg)] bg-[#F5F8FF] px-4 py-3 text-xs leading-5 text-[var(--text-secondary)]">
                Agent 资产已迁移至目标实例。源实例 {sourceName}（
                {sourceInstanceId}）及原有数据不受影响。
              </p>
            </div>
          )}
        </DialogBody>

        {stage !== "migrating" && (
          <DialogFooter className="border-t border-[var(--border)] px-6 py-4">
            {stage === "setup" && (
              <>
                <Button
                  variant="tenant-outline"
                  onClick={() => onOpenChange(false)}
                >
                  取消
                </Button>
                <Button
                  variant="tenant-primary"
                  onClick={() => {
                    setMigrationIndex(0);
                    setStage("migrating");
                  }}
                  data-testid="start-managed-type-migration"
                >
                  开始迁移
                  <ArrowRight className="size-3.5" />
                </Button>
              </>
            )}
            {stage === "success" && (
              <>
                <Button
                  variant="tenant-outline"
                  onClick={() => onOpenChange(false)}
                >
                  完成
                </Button>
                <Button
                  variant="tenant-primary"
                  onClick={() => {
                    onOpenChange(false);
                    window.location.href = `/openclaw/${target.detailId}`;
                  }}
                  data-testid="open-type-migration-target"
                >
                  前往 {targetLabel} 验证
                  <ArrowRight className="size-3.5" />
                </Button>
              </>
            )}
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}
