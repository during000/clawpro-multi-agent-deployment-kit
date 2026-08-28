import { Button } from '@/components/ui/button';
import { Download } from 'lucide-react';
import { CardTitle, MetaText } from '@/components/ui/Typography';
import { toast } from 'sonner';
import { downloadSampleSkillZip, downloadSamplePluginZip } from './downloadUtils';

type UploadRequirementsCardVariant = 'skill' | 'plugin-upload' | 'plugin-update' | 'standard' | 'hook-manifest';

interface UploadRequirementsCardProps {
  variant?: UploadRequirementsCardVariant;
  /** 是否在气泡/Popover 内使用：去掉边框和 padding */
  borderless?: boolean;
}

export function UploadRequirementsCard({ variant = 'skill', borderless = false }: UploadRequirementsCardProps) {
  const showDownloadButton = variant === 'skill' || variant === 'plugin-upload';

  const handleDownload = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (variant === 'plugin-upload') {
      downloadSamplePluginZip();
    } else {
      downloadSampleSkillZip();
    }
    toast.success('样例文件下载中...');
  };

  return (
    <div className={borderless ? 'text-left' : 'border border-gray-200 rounded-[4px] p-4 text-left bg-white'}>
      <div className="flex items-center justify-between mb-2">
        <CardTitle as="h4">上传要求</CardTitle>
        {showDownloadButton && (
          <Button type="button" variant="outline" size="sm" onClick={handleDownload}>
            <Download className="w-3.5 h-3.5" />
            下载样例
          </Button>
        )}
      </div>

      {/* Skill 变体：SKILL.md 要求 */}
      {variant === 'skill' && (
        <div className="space-y-2">
          <div className="flex gap-1">
            <MetaText as="span" tone="secondary">1.</MetaText>
            <MetaText tone="secondary">ZIP 包/文件夹 <strong>根目录</strong> 必须包含 SKILL.md 文件（建议 SKILL 大写）</MetaText>
          </div>
          <div className="space-y-1.5">
            <div className="flex gap-1 leading-relaxed">
              <MetaText as="span" tone="secondary">2.</MetaText>
              <MetaText tone="secondary">SKILL.md 文件需包含 YAML 格式的技能名称和描述，name 和 description 后必须有空格</MetaText>
            </div>
            <pre className="bg-[#FAFAFA] border border-gray-200 rounded-[4px] px-3 py-2 text-xs text-[var(--text-secondary)] font-mono whitespace-pre leading-relaxed">
{`---
name: skill-creator
description: this is a skill creator.
---`}
            </pre>
          </div>
          <div className="flex gap-1">
            <MetaText as="span" tone="secondary">3.</MetaText>
            <MetaText tone="secondary">建议文件夹/ZIP 包名称和 name 名称保持一致</MetaText>
          </div>
        </div>
      )}

      {variant === 'standard' && (
        <div className="space-y-2">
          <div className="flex gap-1">
            <MetaText as="span" tone="secondary">1.</MetaText>
            <MetaText tone="secondary">仅支持上传 <strong>.md</strong> Markdown 文件。</MetaText>
          </div>
          <div className="flex gap-1">
            <MetaText as="span" tone="secondary">2.</MetaText>
            <MetaText tone="secondary">上传后填写显示名称、slug、版本号和应用范围，文件名会作为默认显示名称。</MetaText>
          </div>
          <div className="flex gap-1">
            <MetaText as="span" tone="secondary">3.</MetaText>
            <MetaText tone="secondary">选择 CLAUDE.md 类用于全局指令文件，选择 rules 用于企业规范文件。</MetaText>
          </div>
        </div>
      )}

      {variant === 'hook-manifest' && (
        <div className="space-y-2">
          <div className="flex gap-1">
            <MetaText as="span" tone="secondary">1.</MetaText>
            <MetaText tone="secondary">支持 <strong>.yaml</strong> 或 <strong>.yml</strong> 文件，无需命名为 hooks.yaml。</MetaText>
          </div>
          <div className="flex gap-1">
            <MetaText as="span" tone="secondary">2.</MetaText>
            <MetaText tone="secondary">顶层需包含 <strong>hooks</strong> 数组，每个 Hook 需配置 id、description、event 和 command。</MetaText>
          </div>
        </div>
      )}

      {/* 插件变体（发布/更新）：agent.plugin.json + package.json 要求 */}
      {(variant === 'plugin-upload' || variant === 'plugin-update') && (
        <ol className="text-xs text-[#737373] space-y-2 list-decimal pl-5">
          <li className="leading-relaxed">
            插件 ZIP 包<strong>根目录</strong>必须包含
            <code className="mx-1 px-1 py-0.5 bg-[#FAFAFA] border border-[#E5E5E5] rounded text-[11px] font-mono text-[#334155]">agent.plugin.json</code>
            与
            <code className="mx-1 px-1 py-0.5 bg-[#FAFAFA] border border-[#E5E5E5] rounded text-[11px] font-mono text-[#334155]">package.json</code>
            文件，系统据此识别插件
          </li>
          <li className="leading-relaxed">
            建议压缩包（或内部文件夹）名称与下方"唯一标识"保持一致
          </li>
        </ol>
      )}
    </div>
  );
}
