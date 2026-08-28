import { ChevronDown, FileCode2 } from "lucide-react";

import { BodyMedium, MetaText } from "@/components/ui/Typography";
import type { WorkflowNodeConfigAsset } from "./multiAgentDevelopmentAssets";

const ASSET_TYPE_LABEL: Record<WorkflowNodeConfigAsset["type"], string> = {
  rules: "Rules",
  skill: "Skill",
  contract: "Contract",
};

export function NodeConfigAssets({
  assets,
}: {
  assets: WorkflowNodeConfigAsset[];
}) {
  if (assets.length === 0) {
    return (
      <div className="rounded-[var(--radius-md)] border border-dashed border-[var(--cp-border)] bg-white px-4 py-5 text-center">
        <MetaText tone="weak">该节点暂未配置资产。</MetaText>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-[var(--radius-md)] border border-[var(--cp-border)] bg-white">
      {assets.map((asset, index) => (
        <details
          key={asset.id}
          className={`group ${index > 0 ? "border-t border-[var(--cp-border)]" : ""}`}
        >
          <summary className="flex cursor-pointer list-none items-start gap-3 px-4 py-3.5 transition-colors hover:bg-[var(--bg-subtle)] [&::-webkit-details-marker]:hidden">
            <span className="mt-0.5 inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-[4px] bg-[var(--bg-brand-subtle)] text-[var(--cp-brand-blue)]">
              <FileCode2 className="h-4 w-4" />
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <BodyMedium>{asset.name}</BodyMedium>
                <span className="rounded-[4px] bg-[var(--bg-subtle)] px-1.5 py-0.5 text-[11px] text-[var(--text-muted)]">
                  {ASSET_TYPE_LABEL[asset.type]}
                </span>
                <MetaText tone="weak">v{asset.version}</MetaText>
              </div>
              <p className="mb-0 mt-1 text-xs leading-5 text-[var(--text-muted)]">
                {asset.summary}
              </p>
            </div>
            <ChevronDown className="mt-1 h-4 w-4 shrink-0 text-[var(--text-muted)] transition-transform group-open:rotate-180" />
          </summary>
          <div className="border-t border-[var(--cp-border)] bg-[var(--bg-subtle)] px-4 py-4">
            <pre className="m-0 max-h-[420px] overflow-auto whitespace-pre-wrap break-words font-sans text-sm leading-6 text-[var(--text-body)]">
              {asset.content}
            </pre>
            <MetaText className="mt-3 block break-all font-mono" tone="weak">
              来源：{asset.source}
            </MetaText>
          </div>
        </details>
      ))}
    </div>
  );
}
