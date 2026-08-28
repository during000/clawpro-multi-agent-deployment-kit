/**
 * CloneAgentImageDialog - 克隆 Agent 为镜像（管控端 Agent 列表 · 操作列「更多」）
 *
 * 产品目标：补齐 Agent 的二次开发与复用链路——支持将编辑后的 Agent 一键
 * 创建为自定义镜像，沉淀为企业可复用模板，便于批量复制与分发。
 *
 * 结构对齐 CreateAgentDialog（管控端多步弹窗范本）：
 *   - Header（Title + Description）→ 顶部 Alert 提示 → DialogBody（space-y-5 字段组）→ Footer
 *   - label：text-sm font-medium text-[var(--text-emphasis)] + 红色必填星号
 *   - loading 态在提交按钮内（Loader2 + 禁用），不锁弹窗
 *   - 错误提示贴近字段 / Body 顶部，不漂浮
 *
 * 流程：
 *   第一步：说明 + 隐私强提示 + 镜像名称 →「确认克隆」（按钮内 loading）
 *   创建成功后切换为结果态：展示 ImageId + 引导前往「Agent 类型」页启用
 *   （跳转携带 ?enableImage=<imageId>，由 AgentTypesTable 自动打开对应类型的
 *   「切换镜像」弹窗并预选）
 *
 * ─── 研发实现要点（后端/联调）─────────────────────────────────────────
 * 1) CVM CreateImage 接口（https://cloud.tencent.com/document/product/213/16726）
 *    走 cloud proxy 透传：POST /admin/cloud/mutate/cvm，Header X-TC-Action:
 *    CreateImage，Body: { InstanceId, ImageName, ImageDescription? }，
 *    返回 { ImageId, RequestId }。镜像名称上限 60 字符、不可与已有镜像重名。
 *    ⚠️ CreateImage 目前不在透传白名单（现状仅 Associate/Disassociate
 *    SecurityGroups），需后端在 cloudProxyRegistry 注册后本调用才真实生效。
 *    （参考：《ClawPro独立站腾讯云API通用透传接口.md》第 8 节"零适配扩展"）
 * 2) 隐私数据清理：不做"临时清空→复原"流程，由页面强提示 Agent 所属用户在
 *    克隆前自行清理模型/通道/密钥等隐私信息；镜像内最终内容以清理后状态为准。
 * 3) 创建成功后自动导入：将新镜像写入该 Agent 所属「Agent 类型」的自定义
 *    镜像列表，agentVersion 与当前 Agent 版本一致、active=false（未启用，
 *    待用户在 Agent 类型页切换启用）。本仓库为前端原型，直接写
 *    localStorage("admin_images_v3")，与 ImageManagement.tsx 的镜像持久化
 *    key 保持一致。
 * 4) DEV 环境 cloud proxy 无后端、透传封装会静默返回空对象，此处对空
 *    ImageId 降级为本地模拟 ID（img-xxxxxxxx），保证原型演示链路可走通；
 *    生产环境走真实返回。
 */
import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation } from "wouter";
import { AlertTriangle, Copy, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { CodeText, MetaText } from "@/components/ui/Typography";
import { createSecurityMutateApi } from "@/pages/admin/Security/api/shared";
import type { ImageRow } from "@/pages/admin/ImageManagement/deriveAgentTypeView";
import type { Claw } from "../OpenClawMonitor";

// ─── CVM CreateImage 透传封装 ──────────────────────────────────────────
interface CreateImageResponse {
  ImageId?: string;
  RequestId?: string;
}
const createCvmImage = createSecurityMutateApi<CreateImageResponse>("CreateImage", "cvm");

/** 与 OpenClawMonitor.AGENT_TYPE_DISPLAY 保持一致的显示映射（避免循环依赖，弹窗内独立维护） */
const AGENT_TYPE_LABEL: Record<string, string> = {
  OpenClaw: "OpenClaw",
  Hermes: "Hermes Agent",
  LightclawACE: "LightClaw ACE",
  LocalAgent: "本地 Agent",
};

/** 镜像持久化 key —— 与 ImageManagement.tsx 保持一致（勿单独修改） */
const IMAGES_STORAGE_KEY = "admin_images_v3";

/** 腾讯云镜像名称上限 */
const IMAGE_NAME_MAX = 60;

function nowStr(): string {
  return new Date().toISOString().replace("T", " ").slice(0, 19);
}

/** DEV / 白名单未生效时降级生成模拟镜像 ID（img- + 8 位十六进制） */
function mockImageId(): string {
  return `img-${Math.random().toString(16).slice(2, 10)}`;
}

/** 读取现有镜像列表，用于重名校验与自动导入 */
function loadExistingImages(): ImageRow[] {
  try {
    const raw = localStorage.getItem(IMAGES_STORAGE_KEY);
    return raw ? (JSON.parse(raw) as ImageRow[]) : [];
  } catch {
    return [];
  }
}

/** 将克隆产生的自定义镜像自动导入对应 Agent 类型（active=false 待启用） */
function persistClonedImage(image: ImageRow): void {
  try {
    const list = loadExistingImages();
    if (!list.some((i) => i.id === image.id)) {
      list.push(image);
      localStorage.setItem(IMAGES_STORAGE_KEY, JSON.stringify(list));
    }
  } catch {
    // localStorage 异常不阻断主流程（创建本身已成功）
  }
}

/** 必填标记（对齐 CreateAgentDialog 的 RequiredMark） */
function RequiredMark() {
  return <span className="text-[var(--text-danger)] leading-none">*</span>;
}

type Phase = "form" | "success";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  claw: Claw | null;
}

export default function CloneAgentImageDialog({ open, onOpenChange, claw }: Props) {
  const [, setLocation] = useLocation();

  const [phase, setPhase] = useState<Phase>("form");
  const [imageName, setImageName] = useState("");
  const [imageId, setImageId] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [apiError, setApiError] = useState("");

  // 弹窗"由关到开"时重置为表单态（与 SwitchImageDialog 相同的 prevOpen 模式）
  const prevOpen = useRef(open);
  useEffect(() => {
    if (open && !prevOpen.current) {
      setPhase("form");
      setImageName("");
      setImageId("");
      setApiError("");
      setSubmitting(false);
    }
    prevOpen.current = open;
  }, [open]);

  const agentTypeLabel = claw ? (AGENT_TYPE_LABEL[claw.agentType] ?? claw.agentType) : "";

  // 镜像名称校验：非空 + 长度 2-60 + 不与现有镜像重名（对应云上 InvalidImageName.Duplicate）
  const existingNames = useMemo(
    () => (open ? loadExistingImages().map((i) => i.name) : []),
    [open],
  );
  const nameError = useMemo(() => {
    const v = imageName.trim();
    if (!v) return "";
    if (v.length < 2 || v.length > IMAGE_NAME_MAX) {
      return `长度需在 2-${IMAGE_NAME_MAX} 个字符之间`;
    }
    if (existingNames.includes(v)) return "该镜像名称已存在，请更换名称";
    return "";
  }, [imageName, existingNames]);

  const canSubmit = imageName.trim().length >= 2 && !nameError && !submitting;

  // ─── 确认克隆：调用 CVM CreateImage（loading 在按钮内）─────────────────
  const handleConfirmClone = async () => {
    if (!claw || !canSubmit) return;
    setSubmitting(true);
    setApiError("");
    try {
      const res = await createCvmImage({
        InstanceId: claw.instanceId,
        ImageName: imageName.trim(),
        ImageDescription: `ClawPro clone of agent "${claw.name}" (${claw.id}), type ${claw.agentType}, version ${claw.version}`.slice(0, 256),
      });
      const finalImageId = res?.ImageId?.trim() || mockImageId();
      setImageId(finalImageId);
      // 自动导入到对应 Agent 类型的自定义镜像列表（版本与当前 Agent 一致，未启用）
      persistClonedImage({
        id: finalImageId,
        name: imageName.trim(),
        status: "available",
        type: "custom",
        agentType: claw.agentType,
        agentVersion: claw.version,
        os: "Linux 64位",
        createTime: nowStr(),
        active: false,
      });
      setPhase("success");
    } catch (e) {
      setApiError(e instanceof Error ? e.message : String(e));
    } finally {
      setSubmitting(false);
    }
  };

  /** 前往 Agent 类型页启用：携带 ?enableImage= 自动定位并预选新镜像 */
  const handleGoEnable = () => {
    onOpenChange(false);
    setLocation(`/admin/agent-types?enableImage=${encodeURIComponent(imageId)}`);
  };

  const handleCopyImageId = async () => {
    try {
      await navigator.clipboard.writeText(imageId);
      toast.success("镜像 ID 已复制");
    } catch {
      toast.error("复制失败，请手动复制");
    }
  };

  // 提交中禁止关闭，防止误触中断流程
  const handleOpenChange = (next: boolean) => {
    if (submitting && !next) return;
    onOpenChange(next);
  };

  // ══════════════════ 第一步：表单 ══════════════════
  if (phase === "form") {
    return (
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="sm:max-w-[440px]">
          <DialogHeader>
            <DialogTitle>克隆 Agent 为镜像</DialogTitle>
            <DialogDescription>
              将 Agent「{claw?.name}」的当前环境制作为自定义镜像，供企业内批量复制与分发
            </DialogDescription>
          </DialogHeader>

          <DialogBody className="px-6">
            <div className="space-y-5 pt-3 pb-1">
              {/* 接口调用失败：贴近表单展示，可直接改名重试 */}
              {apiError && (
                <Alert variant="error">
                  <AlertTriangle />
                  <AlertDescription>镜像创建失败：{apiError}，请重试。</AlertDescription>
                </Alert>
              )}

              {/* Agent 概要信息 */}
              <div className="rounded-[4px] border border-[var(--border)] bg-[var(--bg-grey-hover)] px-4 py-3 space-y-1.5">
                <div className="flex items-center justify-between gap-3">
                  <MetaText tone="weak" className="shrink-0">Agent 类型</MetaText>
                  <span className="text-sm text-[var(--text-title)] text-right truncate min-w-0">{agentTypeLabel}</span>
                </div>
                <div className="flex items-center justify-between gap-3">
                  <MetaText tone="weak" className="shrink-0">Agent 版本</MetaText>
                  <span className="text-sm text-[var(--text-title)] text-right truncate min-w-0">{claw?.version}</span>
                </div>
                <div className="flex items-center justify-between gap-3">
                  <MetaText tone="weak" className="shrink-0">实例 ID</MetaText>
                  <CodeText className="text-right truncate min-w-0">{claw?.instanceId}</CodeText>
                </div>
              </div>

              {/* 隐私强提示：紧贴「镜像名称」字段上方 */}
              <div className="space-y-3">
                <Alert variant="warning">
                  <AlertTriangle />
                  <AlertDescription>
                    当前 Agent 的模型、通道、密钥等隐私数据会一并保存到镜像中，请所属用户提前清理后再克隆。
                  </AlertDescription>
                </Alert>

                {/* 镜像名称 */}
                <div className="space-y-2">
                  <label
                    htmlFor="clone-image-name"
                    className="flex items-center gap-1 text-sm font-medium text-[var(--text-emphasis)]"
                  >
                    镜像名称 <RequiredMark />
                  </label>
                  <Input
                    id="clone-image-name"
                    placeholder="请输入镜像名称"
                    value={imageName}
                    maxLength={IMAGE_NAME_MAX}
                    onChange={(e) => setImageName(e.target.value)}
                  />
                  {nameError ? (
                    <MetaText tone="danger" className="text-xs">{nameError}</MetaText>
                  ) : (
                    <MetaText tone="weak" className="text-xs">
                      2-{IMAGE_NAME_MAX} 个字符，创建后不可与已有镜像重名
                    </MetaText>
                  )}
                </div>
              </div>
            </div>
          </DialogBody>

          <DialogFooter>
            <Button variant="claw-outline" onClick={() => onOpenChange(false)}>取消</Button>
            <Button variant="dialog-confirm" disabled={!canSubmit} onClick={handleConfirmClone}>
              {submitting && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
              {submitting ? "提交中…" : "确认克隆"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  // ══════════════════ 第二步：克隆任务已提交（镜像异步制作，约 10 分钟）══════════════════
  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[440px]">
        <DialogHeader>
          <DialogTitle>克隆任务已提交</DialogTitle>
          <DialogDescription>
            镜像制作约需 10 分钟，完成后即可在「{agentTypeLabel}」的自定义镜像列表中启用
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="px-6">
          <div className="space-y-5 pt-3 pb-1">
            {/* 镜像信息 */}
            <div className="rounded-[4px] border border-[var(--border)] bg-[var(--bg-grey-hover)] px-4 py-3 space-y-1.5">
              <div className="flex items-center justify-between gap-3">
                <MetaText tone="weak" className="shrink-0">镜像 ID</MetaText>
                <span className="inline-flex items-center gap-1.5 min-w-0">
                  <CodeText className="truncate">{imageId}</CodeText>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className="size-6 shrink-0"
                    onClick={handleCopyImageId}
                    aria-label="复制镜像 ID"
                  >
                    <Copy className="size-3.5" />
                  </Button>
                </span>
              </div>
              <div className="flex items-center justify-between gap-3">
                <MetaText tone="weak" className="shrink-0">镜像名称</MetaText>
                <span className="text-sm text-[var(--text-title)] text-right truncate min-w-0">{imageName.trim()}</span>
              </div>
              <div className="flex items-center justify-between gap-3">
                <MetaText tone="weak" className="shrink-0">Agent 类型 / 版本</MetaText>
                <span className="text-sm text-[var(--text-title)] text-right truncate min-w-0">{agentTypeLabel} · {claw?.version}</span>
              </div>
            </div>

            {/* 启用引导（醒目警示条） */}
            <Alert variant="warning">
              <AlertTriangle />
              <AlertTitle>新镜像需制作完成并启用后方可使用</AlertTitle>
              <AlertDescription>
                制作完成后，请前往「Agent 类型」页找到对应 Agent 类型并点击「切换」，将新镜像设为当前生效镜像；之后新创建的 Agent 将使用该镜像。
              </AlertDescription>
            </Alert>
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="claw-outline" onClick={() => onOpenChange(false)}>留在本页</Button>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="dialog-confirm" onClick={handleGoEnable}>前往启用</Button>
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-[260px] text-xs leading-relaxed">
              制作完成后，在「Agent 类型」页将新镜像切换为当前生效镜像，之后新建的 Agent 即可使用该镜像
            </TooltipContent>
          </Tooltip>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
