import { CardTitle, MetaText, HelperText } from '@/components/ui/Typography';
import { SurfaceCard } from '@/components/ui/Surface';
import { Switch } from '@/components/ui/switch';
import { Checkbox } from '@/components/ui/checkbox';
import { StatusTag } from '@/components/ui/status-tag';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';

interface SecurityScanCardProps {
  /** 安全检测服务是否激活 */
  securityServiceActive: boolean;
  /** 是否启用安全检测 */
  enableSecurityScan: boolean;
  /** 启用状态变化回调 */
  onEnableSecurityScanChange: (checked: boolean) => void;
  /** 是否默认提交安全检测 */
  defaultSecurityScan: boolean;
  /** 默认提交状态变化回调 */
  onDefaultSecurityScanChange: (checked: boolean) => void;
  /** Switch 是否禁用（可选，默认 !securityServiceActive） */
  switchDisabled?: boolean;
  /** Checkbox 的 id（可选，默认 "skill-default-security-scan"） */
  checkboxId?: string;
  /** 容器样式变体（可选，默认 'surface'） */
  variant?: 'surface' | 'border';
  /**
   * 是否隐藏「设置上传/更新时默认提交安全检测」勾选框。
   * 该开关属于全局默认设置，仅管控端有意义；用户端（tenant/SkillSquare 员工发布 Skill）
   * 应传 true 隐藏。默认 false，管控端行为完全不变。
   */
  hideDefaultSetting?: boolean;
}

/**
 * 提交后进行安全检测卡片组件
 * 用于插件/技能上传、更新弹窗中的安全检测配置
 */
export function SecurityScanCard({
  securityServiceActive,
  enableSecurityScan,
  onEnableSecurityScanChange,
  defaultSecurityScan,
  onDefaultSecurityScanChange,
  switchDisabled,
  checkboxId = 'skill-default-security-scan',
  variant = 'surface',
  hideDefaultSetting = false,
}: SecurityScanCardProps) {
  const isSwitchDisabled = switchDisabled ?? !securityServiceActive;

  const cardContent = (
    <div className="p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <CardTitle as="h4">提交后进行安全检测</CardTitle>
          <Tooltip delayDuration={300}>
            <TooltipTrigger asChild>
              <span className="inline-flex items-center">
                <StatusTag mode="fill" variant="blue" className="cursor-default">限免</StatusTag>
              </span>
            </TooltipTrigger>
            <TooltipContent side="top" className="text-xs max-w-[260px] leading-relaxed">
              限时免费，该检测能力正在公测中，暂不收费，后续如需收费，仅对增量检测收费，并及时与您同步收费方式。
            </TooltipContent>
          </Tooltip>
          {!securityServiceActive && (
            <span className="inline-flex items-center">
              <StatusTag mode="fill" variant="gray">未开通</StatusTag>
            </span>
          )}
        </div>
        <Tooltip delayDuration={300}>
          <TooltipTrigger asChild>
            <span className="inline-flex items-center">
              <Switch
                checked={enableSecurityScan}
                onCheckedChange={onEnableSecurityScanChange}
                disabled={isSwitchDisabled}
              />
            </span>
          </TooltipTrigger>
          {!securityServiceActive && (
            <TooltipContent side="top" className="text-xs max-w-[280px]">
              安全检测服务尚未开通，请前往技能库列表页右上角免费开通试用（26年6月30日前1000次免费试用）。
            </TooltipContent>
          )}
        </Tooltip>
      </div>
      <HelperText as="p" className="mt-2 leading-relaxed">
        {!securityServiceActive
          ? '安全检测服务尚未开通，请前往技能库列表页右上角免费开通试用（26年6月30日前1000次免费试用）。'
          : '开启后将由腾讯云 AI Agent 安全对技能文件进行安全分析，包括代码结构、依赖安全、命令执行、网络请求、文件操作、Prompt 注入等维度的全面审查。检测通常在几分钟内完成。'}
      </HelperText>
      {securityServiceActive && !hideDefaultSetting && (
        <label htmlFor={checkboxId} className={`mt-3 flex items-center gap-2 ${enableSecurityScan ? 'cursor-pointer' : 'cursor-not-allowed'}`}>
          <Checkbox
            id={checkboxId}
            checked={defaultSecurityScan}
            onCheckedChange={(checked) => {
              onDefaultSecurityScanChange(checked === true);
            }}
            disabled={!enableSecurityScan}
          />
          <MetaText as="p" className="self-center">{`设置上传/更新时默认提交安全检测`}</MetaText>
        </label>
      )}
    </div>
  );

  if (variant === 'border') {
    return (
      <div className="border border-gray-200 rounded-[4px]">
        {cardContent}
      </div>
    );
  }

  return <SurfaceCard>{cardContent}</SurfaceCard>;
}
