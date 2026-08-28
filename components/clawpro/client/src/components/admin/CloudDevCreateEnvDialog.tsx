/**
 * CloudDevCreateEnvDialog - 新建云开发环境弹窗（共享组件）
 * 供 CloudDevManagement（管控端）和 AgentChat（用户端）共同使用
 *
 * 视觉规范：
 *   - 4px 圆角（管控端铁律），颜色统一走 --cp-* / --text-* token
 *   - 套餐选中态 bg-[var(--cp-brand-blue)]/5（品牌蓝 5% 透明度）
 *   - Header / Footer 固定，Body 限高 + overflow-y-auto（spec §12.4 长表单）
 *   - 主操作按钮 variant="dialog-confirm"（弹窗专用），次级 variant="claw-outline"
 *
 * 校验：
 *   - 环境名称：必填、长度 ≤ 24、正则 ^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$
 *
 * 字段说明：
 *   - dbType / region 不在 UI 中暴露，使用 DEFAULT_CREATE_ENV_FORM 的默认值（cloud / ap-guangzhou）。
 *     外部仍以 form 字段形式回传，确保下游调用方接口稳定。
 */
import { useEffect, useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import { Check } from "lucide-react";

/* ── 共享类型 ─────────────────────────────────────────────── */
export type CloudDevPackageId = "personal" | "standard";
export type CloudDevDbType = "cloud" | "postgresql";

export interface CreateEnvForm {
  name: string;
  region: string;
  pkg: CloudDevPackageId;
  dbType: CloudDevDbType;
  overflowBilling: boolean;
  autoRenewal: boolean;
}

/** 默认表单值 */
export const DEFAULT_CREATE_ENV_FORM: CreateEnvForm = {
  name: "",
  region: "ap-guangzhou",
  pkg: "standard",
  dbType: "cloud",
  overflowBilling: false,
  autoRenewal: true,
};

/** 环境名校验正则：小写字母开头，字母数字结尾，可含 -，不允许连续 - */
export const ENV_NAME_REGEX = /^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/;

/* ── 组件 Props ───────────────────────────────────────────── */
interface CloudDevCreateEnvDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 预设的初始表单值（如"领取免费体验"预选 standard）；不传则用 DEFAULT_CREATE_ENV_FORM */
  initialForm?: Partial<CreateEnvForm>;
  /** 确认创建：参数为校验通过的完整表单；调用方负责真正创建环境 + 关闭弹窗 */
  onConfirm: (form: CreateEnvForm) => void;
}

export default function CloudDevCreateEnvDialog({
  open,
  onOpenChange,
  initialForm,
  onConfirm,
}: CloudDevCreateEnvDialogProps) {
  const [nf, setNf] = useState<CreateEnvForm>({ ...DEFAULT_CREATE_ENV_FORM, ...initialForm });
  const [nameError, setNameError] = useState(false);

  // 每次重新打开弹窗时重置表单（同时尊重新传入的 initialForm）
  useEffect(() => {
    if (open) {
      setNf({ ...DEFAULT_CREATE_ENV_FORM, ...initialForm });
      setNameError(false);
    }
  // 仅在 open 变 true 时重置；initialForm 引用变化通过外部控制 open 实现
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const handleSubmit = () => {
    const name = nf.name.trim();
    if (!name) { setNameError(true); return; }
    if (name.length > 24 || !ENV_NAME_REGEX.test(name)) { setNameError(true); return; }
    onConfirm({ ...nf, name });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>新建云开发环境</DialogTitle>
        </DialogHeader>
        <div className="flex-1 overflow-y-auto scrollbar-hide -mx-6 px-6 space-y-6 py-2">
          {/* 环境名称 */}
          <div className="space-y-2">
            <Label className="text-sm font-medium text-[var(--text-title)]">环境名称</Label>
            <div className="relative">
              <Input
                placeholder="请输入环境名称"
                value={nf.name}
                onChange={e => { setNf({ ...nf, name: e.target.value }); if (nameError) setNameError(false); }}
                aria-invalid={nameError || undefined}
                className={`bg-white pr-12 ${nameError ? "border-[var(--text-danger)] ring-1 ring-[var(--text-danger)]/20" : ""}`}
                maxLength={24}
              />
              <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-[var(--text-weak)]">{nf.name.length}/24</span>
            </div>
            {nameError && (
              <p className="text-xs text-[var(--text-danger)]">由小写字母、数字及-组成，且仅支持字母开头、字母数字结尾，不支持连续-，最大长度为24个字符。</p>
            )}
          </div>

          {/* 套餐版本
              当前档位：个人版 / 标准版（企业版通过单独的「领取免费体验」入口分发，不在创建环境弹窗暴露） */}
          <div className="space-y-3">
            <Label className="text-sm font-medium text-[var(--text-title)]">套餐版本</Label>
            <div className="grid grid-cols-2 gap-3">
              {([
                {
                  id: "personal" as const,
                  name: "个人版",
                  tag: "适合个人开发使用",
                  tagColor: "gray" as const,
                  features: [
                    "资源点配额4万点",
                    "Serverless云数据库、云存储、云函数",
                    "原生打通微信生态、免鉴权API调用",
                  ],
                },
                {
                  id: "standard" as const,
                  name: "标准版",
                  tag: "适合团队开发使用",
                  tagColor: "blue" as const,
                  features: [
                    "资源点配额33万点",
                    "对接自有数据库和CDN",
                    "更好的可用性保障(数据库回档、扩容速度)",
                  ],
                },
              ]).map(pkg => {
                const sel = nf.pkg === pkg.id;
                return (
                  <button
                    key={pkg.id}
                    type="button"
                    onClick={() => setNf((prev) => ({ ...prev, pkg: pkg.id }))}
                    className={`relative rounded-[var(--radius-lg)] border p-4 text-left transition-all flex flex-col h-full ${sel ? "border-[var(--cp-brand-blue)] bg-[var(--cp-surface)]" : "border-[var(--cp-border)] bg-[var(--cp-surface)] hover:border-[var(--cp-border-control)]"}`}
                  >
                    {sel && (
                      <div className="absolute top-3 right-3 w-4 h-4 rounded-full bg-[var(--cp-brand-blue)] flex items-center justify-center">
                        <Check className="w-2.5 h-2.5 text-white" strokeWidth={3} />
                      </div>
                    )}
                    <div className="flex items-center gap-2 mb-2">
                      <p className={`text-sm font-semibold ${sel ? "text-[var(--text-brand)]" : "text-[var(--text-title)]"}`}>{pkg.name}</p>
                      {pkg.tagColor === "gray"
                        ? <Badge variant="secondary">{pkg.tag}</Badge>
                        : <Badge color={pkg.tagColor}>{pkg.tag}</Badge>}
                    </div>
                    <div className="flex-1">
                      <ul className="space-y-1">
                        {pkg.features.map((f) => (
                          <li key={f} className="flex items-start gap-1.5 text-xs text-[var(--text-secondary)]">
                            <Check className="w-3 h-3 text-[var(--text-secondary)] mt-0.5 flex-shrink-0" />
                            <span>{f}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  </button>
                );
              })}
            </div>
          </div>

          {/* 超限设置 */}
          <Separator />
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-[var(--text-title)]">超限设置</p>
              <p className="text-xs text-[var(--text-muted)] mt-0.5 leading-relaxed">
                开启超限设置后，不限制用户访问，超过套餐额度部分按量计费使用
                <a href="#" target="_blank" rel="noreferrer" className="text-[var(--text-brand)] hover:underline ml-1">详细介绍</a>
              </p>
            </div>
            <Switch checked={nf.overflowBilling} onCheckedChange={v => setNf({ ...nf, overflowBilling: v })} />
          </div>

          {/* 自动续期 */}
          <Separator />
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-[var(--text-title)]">自动续期</p>
              <p className="text-xs text-[var(--text-muted)] mt-0.5">环境到期后按月自动续期</p>
            </div>
            <Switch checked={nf.autoRenewal} onCheckedChange={v => setNf({ ...nf, autoRenewal: v })} />
          </div>
        </div>
        <DialogFooter>
          <Button variant="claw-outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button variant="dialog-confirm" onClick={handleSubmit}>确认创建</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
