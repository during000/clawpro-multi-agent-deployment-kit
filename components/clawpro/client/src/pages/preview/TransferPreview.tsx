/**
 * Transfer Preview
 * 路由：/preview/transfer
 * 展示穿梭框组件的 instant/batch 模式、搜索、禁用项、空态
 */
import { useState } from "react";
import { Transfer } from "@/components/ui/transfer";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { MetaText, SectionTitle } from "@/components/ui/Typography";

function DemoBlock({ title, desc, children }: { title: string; desc?: string; children: React.ReactNode }) {
  return (
    <section className="rounded-[8px] border border-[#e5e5e5] overflow-hidden">
      <header className="flex items-baseline justify-between px-5 py-3 border-b border-[#f0f0f0] bg-[#fafafa]">
        <SectionTitle as="h3" className="!text-sm">{title}</SectionTitle>
        {desc && <MetaText tone="weak">{desc}</MetaText>}
      </header>
      <div className="p-5">{children}</div>
    </section>
  );
}

type MockItem = {
  key: string;
  name: string;
  ip: string;
  version: string;
  disabled?: boolean;
};

const MOCK_DATA: MockItem[] = Array.from({ length: 20 }, (_, i) => ({
  key: `item-${i}`,
  name: `Agent-${String(i + 1).padStart(3, "0")}`,
  ip: `10.0.${Math.floor(i / 10)}.${(i % 254) + 1}`,
  version: i % 3 === 0 ? "基础版" : "旗舰版",
  disabled: i % 3 === 0,
}));

export default function TransferPreview() {
  const [selected1, setSelected1] = useState<string[]>(["item-1", "item-4"]);
  const [selected2, setSelected2] = useState<string[]>([]);

  return (
    <div className="min-h-screen bg-[#F8FAFC] p-8">
      <div className="max-w-5xl mx-auto space-y-8">
        <header className="space-y-1">
          <h1 className="text-xl font-semibold text-[#0F172A]">Transfer 穿梭框</h1>
          <p className="text-sm text-[#64748B]">
            instant 模式（默认推荐） · 搜索 · 分页 · 禁用项 Tooltip · Table compact 密度
          </p>
        </header>

        <DemoBlock title="Instant 模式（默认）" desc="左侧勾选立即搬到右侧">
          <Transfer<MockItem>
            dataSource={MOCK_DATA}
            rowKey="key"
            targetKeys={selected1}
            onChange={(nextKeys) => setSelected1(nextKeys)}
            showSearch
            searchPlaceholder={["搜索资产名称 / IP", "搜索已选资产"]}
            pagination={{ pageSize: 8 }}
            height={320}
            titles={["全部 AI Agent 资产", "已选 AI Agent 资产"]}
            isItemDisabled={(h) => !!h.disabled}
            renderDisabledTrigger={(_h, defaultCheckbox) => (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="inline-flex">{defaultCheckbox}</span>
                </TooltipTrigger>
                <TooltipContent>基础版资产请升级到旗舰版以使用该能力</TooltipContent>
              </Tooltip>
            )}
            filterOption={(input, h) => {
              const needle = input.toLowerCase();
              return [h.name, h.ip].some((v) => v.toLowerCase().includes(needle));
            }}
            columns={[
              {
                key: "name",
                header: "名称",
                render: (h) => (
                  <div className="min-w-0">
                    <div className="truncate text-[var(--text-emphasis)]">{h.name}</div>
                  </div>
                ),
              },
              { key: "version", header: "版本", width: 80, render: (h) => h.version },
              { key: "ip", header: "IP", width: 120, render: (h) => h.ip },
            ]}
          />
        </DemoBlock>

        <DemoBlock title="空态" desc="右侧无数据时的空状态展示">
          <Transfer<MockItem>
            dataSource={MOCK_DATA.slice(0, 5)}
            rowKey="key"
            targetKeys={selected2}
            onChange={(nextKeys) => setSelected2(nextKeys)}
            showSearch
            searchPlaceholder="搜索"
            pagination={{ pageSize: 5 }}
            height={240}
            titles={["候选项", "已选择"]}
            columns={[
              { key: "name", header: "名称", render: (h) => h.name },
              { key: "ip", header: "IP", width: 120, render: (h) => h.ip },
            ]}
          />
        </DemoBlock>

        <DemoBlock title="视觉规范">
          <div className="grid grid-cols-2 gap-6 text-xs text-[#64748B]">
            <div className="space-y-1">
              <p className="font-medium text-[#0A0A0A]">面板</p>
              <p>• 等宽双面板，圆角 4px</p>
              <p>• 描边：var(--border) #EAEEF4</p>
              <p>• 表头底色：#FAFBFD</p>
              <p>• 行高：40px（Table compact）</p>
              <p>• 字号：12px</p>
            </div>
            <div className="space-y-1">
              <p className="font-medium text-[#0A0A0A]">交互</p>
              <p>• instant：勾选即搬，行末 X 移除</p>
              <p>• batch：中间 &gt; &lt; 按钮批量穿梭</p>
              <p>• 禁用项：60% 透明度 + Tooltip</p>
              <p>• 分页：simple size="small"</p>
              <p>• 空态：纯文字（不用插画）</p>
            </div>
          </div>
        </DemoBlock>
      </div>
    </div>
  );
}
