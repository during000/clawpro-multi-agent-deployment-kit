/**
 * UpdateRecordsTab - 「项目资产管理」某组织的更新记录（版本历史）
 * 每次编辑保存或工具库联动都会新增一条记录。
 */
import { useState } from 'react';
import { ChevronDown, Clock } from 'lucide-react';
import { StatusTag, type StatusTagColor } from '@/components/ui/status-tag';
import { Empty, EmptyHeader, EmptyDescription } from '@/components/ui/empty';
import { BodyMedium, MetaText } from '@/components/ui/Typography';
import type { ProjectAssetTagKind, ProjectAssetUpdateRecord } from './types';

/** tag 只有两种，用颜色区分：手动编辑=蓝、自动同步=灰 */
const TAG_META: Record<ProjectAssetTagKind, { label: string; color: StatusTagColor }> = {
  manual: { label: '手动编辑', color: 'blue' },
  auto: { label: '自动同步', color: 'gray' },
};

function formatTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export default function UpdateRecordsTab({ records }: { records: ProjectAssetUpdateRecord[] }) {
  // 每个 section 的明细小字默认折叠，点击主文案行后展开；key = `${record.id}-${sectionIndex}`
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const toggle = (key: string) =>
    setExpanded((prev) => ({ ...prev, [key]: !prev[key] }));

  if (records.length === 0) {
    return (
      <Empty className="py-16">
        <EmptyHeader>
          <EmptyDescription>暂无更新记录，对项目资产进行编辑并保存后将生成版本记录</EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <div className="space-y-2">
      {records.map((record) => {
        const meta = TAG_META[record.tagKind];
        return (
          <div
            key={record.id}
            className="flex items-start gap-3 px-4 py-3 rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)]"
          >
            <div className="flex flex-col items-center gap-1 shrink-0 w-[72px]">
              <BodyMedium>v{record.version}</BodyMedium>
              <StatusTag variant={meta.color} mode="soft">{meta.label}</StatusTag>
            </div>
            <div className="min-w-0 flex-1 space-y-2">
              {record.sections.map((section, si) => {
                const hasItems = !!section.items && section.items.length > 0;
                const key = `${record.id}-${si}`;
                const open = !!expanded[key];
                return (
                  <div key={si}>
                    {hasItems ? (
                      <button
                        type="button"
                        onClick={() => toggle(key)}
                        aria-expanded={open}
                        className="group -mx-1.5 flex items-center gap-1 rounded-[4px] px-1.5 py-0.5 transition-colors hover:bg-[var(--bg-grey-hover)]"
                      >
                        <BodyMedium className="text-left text-[var(--text-body)]">{section.title}</BodyMedium>
                        <ChevronDown
                          className={`w-3.5 h-3.5 shrink-0 text-[var(--text-weak)] transition-transform duration-200 ${open ? 'rotate-180' : ''}`}
                        />
                      </button>
                    ) : (
                      <BodyMedium className="text-[var(--text-body)]">{section.title}</BodyMedium>
                    )}
                    {hasItems && open && (
                      <ul className="mt-1 space-y-1">
                        {section.items!.map((item, i) => (
                          <li key={i} className="flex items-start gap-1.5">
                            <span className="mt-[7px] w-1 h-1 rounded-full bg-[var(--text-weak)] shrink-0" />
                            <span className="text-[13px] leading-5 text-[var(--text-secondary)]">{item}</span>
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                );
              })}
              <div className="flex items-center gap-3 pt-0.5">
                <span className="inline-flex items-center gap-1">
                  <Clock className="w-3.5 h-3.5 text-[var(--text-weak)]" />
                  <MetaText tone="secondary">{formatTime(record.createdAt)}</MetaText>
                </span>
                {record.operator && <MetaText tone="secondary">操作人：{record.operator}</MetaText>}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
