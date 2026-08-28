/**
 * CloudDevClaimTrialDialog - 「云开发 × Clawpro 免费体验」领取环境弹窗
 *
 * 业务规则（产品确认）：
 *   - 领取的环境固定为「标准版」
 *   - 固定免费使用 6 个月
 *   - 每个云账号最多领取 3 个
 *   - 用户仅需填写「环境名」即可领取，其他参数（规格 / 数据库 / 区域 / 计费）均不可选
 *
 * 视觉规范：
 *   - 管控端 4px 圆角铁律（rounded-[var(--radius-lg)]）
 *   - 主操作按钮 variant="dialog-confirm"，次级 "claw-outline"
 *   - 权益卡使用浅蓝渐变 + 品牌色强调（与活动横幅视觉联动）
 *
 * 校验：
 *   - 环境名：必填、长度 ≤ 24、正则 ENV_NAME_REGEX（复用 CloudDevCreateEnvDialog 导出）
 *   - 名称重复：由调用方通过 existingNames 传入做唯一性校验
 *   - 已达上限（claimedCount >= MAX_CLAIM）时禁用确认按钮 + 头部提示
 */
import { useEffect, useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Sparkles, Gift, Clock, Layers, Check } from "lucide-react";
import { ENV_NAME_REGEX } from "./CloudDevCreateEnvDialog";

/** 单云账号最多可领取数量 */
export const MAX_TRIAL_CLAIM = 3;

interface CloudDevClaimTrialDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 已领取的免费体验环境数 */
  claimedCount: number;
  /** 已存在的环境名集合，用于做唯一性校验（不区分大小写） */
  existingNames: string[];
  /** 确认领取：参数为校验通过的环境名；调用方负责创建环境 + 关闭弹窗 */
  onConfirm: (envName: string) => void;
}

export default function CloudDevClaimTrialDialog({
  open,
  onOpenChange,
  claimedCount,
  existingNames,
  onConfirm,
}: CloudDevClaimTrialDialogProps) {
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);

  const remaining = Math.max(0, MAX_TRIAL_CLAIM - claimedCount);
  const reachedLimit = remaining <= 0;

  useEffect(() => {
    if (open) {
      setName("");
      setError(null);
    }
  }, [open]);

  const validate = (raw: string): string | null => {
    const trimmed = raw.trim();
    if (!trimmed) return "请输入环境名";
    if (trimmed.length > 24) return "环境名长度不能超过 24 个字符";
    if (!ENV_NAME_REGEX.test(trimmed)) return "仅支持小写字母开头，可含数字与中划线（不允许连续中划线）";
    if (existingNames.some(n => n.trim().toLowerCase() === trimmed.toLowerCase())) {
      return "已存在同名环境，请更换名称";
    }
    return null;
  };

  const handleSubmit = () => {
    if (reachedLimit) return;
    const err = validate(name);
    if (err) {
      setError(err);
      return;
    }
    onConfirm(name.trim());
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-[var(--cp-brand-blue)]" />
            领取免费体验环境
          </DialogTitle>
          <DialogDescription>
            云开发 × Clawpro 限时活动：标准版环境，免费使用 6 个月，每个云账号最多领取 {MAX_TRIAL_CLAIM} 个。
          </DialogDescription>
        </DialogHeader>

        {/* Body：三段（权益卡 / 表单 / 上限提示），用 space-y-4 控制间距 */}
        <div className="space-y-4">
          {/* 权益卡片：浅蓝底 + 品牌色边 */}
          <div className="rounded-[var(--radius-lg)] border border-[var(--cp-brand-blue)]/30 bg-[var(--cp-brand-blue)]/5 px-4 py-3 space-y-2.5">
            <div className="flex items-center gap-2 text-[13px] font-medium text-[var(--text-title)]">
              <Gift className="w-3.5 h-3.5 text-[var(--cp-brand-blue)]" />
              本次领取你将获得
            </div>
            <ul className="space-y-1.5 text-[12px] text-[var(--text-secondary)] leading-relaxed">
              <li className="flex items-start gap-2">
                <Layers className="w-3.5 h-3.5 text-[var(--cp-brand-blue)] mt-0.5 flex-shrink-0" />
                <span>
                  <span className="font-semibold text-[var(--text-title)]">标准版</span> 云开发环境（适合小团队场景）
                </span>
              </li>
              <li className="flex items-start gap-2">
                <Clock className="w-3.5 h-3.5 text-[var(--cp-brand-blue)] mt-0.5 flex-shrink-0" />
                <span>
                  免费使用 <span className="font-semibold text-[var(--text-title)]">6 个月</span>，到期后可续费继续使用
                </span>
              </li>
              <li className="flex items-start gap-2">
                <Check className="w-3.5 h-3.5 text-[var(--cp-brand-blue)] mt-0.5 flex-shrink-0" />
                <span>
                  当前云账号最多领取 <span className="font-semibold text-[var(--text-title)]">{MAX_TRIAL_CLAIM} 个</span>，已领{" "}
                  <span className="font-semibold text-[var(--text-title)]">{claimedCount}</span> /{" "}
                  {MAX_TRIAL_CLAIM}
                </span>
              </li>
            </ul>
          </div>

          {/* 表单：仅环境名（必填红点采用 Label 内嵌 span，与现有 Field 写法一致；Input aria-invalid 由错误态触发） */}
          <div className="space-y-2">
            <Label htmlFor="claim-env-name" className="text-sm font-medium text-[var(--text-title)]">
              环境名 <span className="text-[var(--text-danger)]" aria-hidden="true">*</span>
            </Label>
            <Input
              id="claim-env-name"
              value={name}
              onChange={e => {
                setName(e.target.value);
                if (error) setError(null);
              }}
              onBlur={e => {
                const err = validate(e.target.value);
                setError(err);
              }}
              placeholder="例如：team-prod-01"
              disabled={reachedLimit}
              maxLength={24}
              required
              aria-invalid={!!error}
              aria-describedby={error ? "claim-env-name-error" : "claim-env-name-helper"}
            />
            {error ? (
              <p id="claim-env-name-error" className="text-xs text-[var(--text-danger)]">
                {error}
              </p>
            ) : (
              <p id="claim-env-name-helper" className="text-xs text-[var(--text-muted)]">
                小写字母开头，可含数字与中划线，长度不超过 24 个字符
              </p>
            )}
          </div>

          {reachedLimit && (
            <div className="rounded-[var(--radius-lg)] bg-[var(--bg-grey-normal)] px-3 py-2 text-xs text-[var(--text-secondary)]">
              当前云账号已领取满 {MAX_TRIAL_CLAIM} 个免费体验环境。如需更多环境，请前往「资源管理」购买或释放已有环境后再次领取。
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="claw-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="dialog-confirm"
            onClick={handleSubmit}
            disabled={reachedLimit || !name.trim() || !!error}
          >
            确认领取
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
