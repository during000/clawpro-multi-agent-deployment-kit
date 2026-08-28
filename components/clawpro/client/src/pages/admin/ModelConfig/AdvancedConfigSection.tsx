/**
 * AdvancedConfigSection
 * 「添加模型」弹窗内的「高级配置」可折叠区域：
 *   - 厂商模型：maxTokens + headers
 *   - 自定义表单：contextWindow + maxTokens + headers
 *
 * 抽出来的原因：原本嵌在 ModelConfig.tsx 顶部，长度 150+ 行；这块只服务
 * AddModelDialog 的折叠面板，没有外部消费者。
 */
import * as React from "react";
import { Plus, Info, ChevronDown, ChevronRight, X } from "lucide-react";

import { Input } from "@/components/ui/input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { BodyMedium, MetaText, MetaMedium } from "@/components/ui/Typography";

export type HeaderEntry = { key: string; value: string };
export type AdvancedConfig = {
  contextWindow: string;
  maxTokens: string;
  headers: HeaderEntry[];
};

export interface AdvancedConfigSectionProps {
  open: boolean;
  onToggle: () => void;
  config: AdvancedConfig;
  onChange: (c: AdvancedConfig) => void;
  /** 仅自定义模型表单模式下显示「上下文长度」字段 */
  showContextWindow: boolean;
}

export function AdvancedConfigSection({
  open,
  onToggle,
  config,
  onChange,
  showContextWindow,
}: AdvancedConfigSectionProps) {
  const updateHeader = (index: number, field: "key" | "value", val: string) => {
    const next = config.headers.map((h, i) => i === index ? { ...h, [field]: val } : h);
    onChange({ ...config, headers: next });
  };

  const addHeader = () => {
    onChange({ ...config, headers: [...config.headers, { key: "", value: "" }] });
  };

  const removeHeader = (index: number) => {
    onChange({ ...config, headers: config.headers.filter((_, i) => i !== index) });
  };

  return (
    <div className="rounded-[var(--radius)] border border-[var(--cp-border)] overflow-hidden">
      {/* 标题行 */}
      <button
        type="button"
        onClick={onToggle}
        className="w-full flex items-center justify-between px-4 py-3 bg-[var(--bg-grey-normal)] hover:bg-[var(--bg-grey-hover)] transition-colors"
      >
        <BodyMedium as="span" tone="secondary">高级配置</BodyMedium>
        <span className="flex items-center gap-1.5">
          <MetaText as="span" tone="weak">非必填</MetaText>
          {open ? (
            <ChevronDown className="w-3.5 h-3.5 text-[var(--text-weak)]" />
          ) : (
            <ChevronRight className="w-3.5 h-3.5 text-[var(--text-weak)]" />
          )}
        </span>
      </button>

      {/* 展开内容 */}
      {open && (
        <div className="px-4 py-3 space-y-3 bg-[var(--cp-surface)]">
          {/* 上下文长度：仅自定义表单模式 */}
          {showContextWindow && (
            <div className="space-y-1.5">
              <div className="flex items-center gap-1">
                <MetaMedium as="label" tone="secondary">上下文长度</MetaMedium>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="cursor-default">
                      <Info className="w-3 h-3 text-[var(--text-weak)]" />
                    </span>
                  </TooltipTrigger>
                  <TooltipContent className="max-w-[240px]">
                    <MetaText tone="inherit">contextWindow，指的是模型总上下文窗口大小。</MetaText>
                  </TooltipContent>
                </Tooltip>
              </div>
              <Input
                type="number"
                placeholder="请输入上下文长度 contextWindow"
                value={config.contextWindow}
                onChange={(e) => onChange({ ...config, contextWindow: e.target.value })}
              />
            </div>
          )}

          {/* 最大输出长度 */}
          <div className="space-y-1.5">
            <div className="flex items-center gap-1">
              <MetaMedium as="label" tone="secondary">最大输出长度</MetaMedium>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="cursor-default">
                    <Info className="w-3 h-3 text-[var(--text-weak)]" />
                  </span>
                </TooltipTrigger>
                <TooltipContent className="max-w-[240px]">
                  <MetaText tone="inherit">maxTokens，即模型单次回复时最多输出的 Token 数。</MetaText>
                </TooltipContent>
              </Tooltip>
            </div>
            <Input
              type="number"
              placeholder="请输入最大输出长度 maxTokens"
              value={config.maxTokens}
              onChange={(e) => onChange({ ...config, maxTokens: e.target.value })}
            />
          </div>

          {/* 请求头 */}
          <div className="space-y-1.5">
            <div className="flex items-center gap-1">
              <MetaMedium as="label" tone="secondary">请求头</MetaMedium>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="cursor-default">
                    <Info className="w-3 h-3 text-[var(--text-weak)]" />
                  </span>
                </TooltipTrigger>
                <TooltipContent className="max-w-[240px]">
                  <MetaText tone="inherit">headers，在 HTTP 请求中用于传递认证信息或数据格式等元数据的参数。</MetaText>
                </TooltipContent>
              </Tooltip>
            </div>
            {/* key-value 行列表 */}
            <div className="space-y-2">
              {config.headers.map((entry, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <Input
                    placeholder="key"
                    value={entry.key}
                    onChange={(e) => updateHeader(idx, "key", e.target.value)}
                    className="w-[36%] shrink-0"
                  />
                  <Input
                    placeholder="value"
                    value={entry.value}
                    onChange={(e) => updateHeader(idx, "value", e.target.value)}
                    className="flex-1"
                  />
                  <button
                    type="button"
                    onClick={() => removeHeader(idx)}
                    className="shrink-0 text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors"
                    aria-label="移除请求头"
                  >
                    <X className="w-3.5 h-3.5" />
                  </button>
                </div>
              ))}
              <button
                type="button"
                onClick={addHeader}
                className="flex items-center gap-1 text-xs text-[var(--text-brand)] hover:opacity-80 transition-opacity mt-1"
              >
                <Plus className="w-3 h-3" />
                添加请求头
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
