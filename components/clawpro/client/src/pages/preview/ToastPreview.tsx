/**
 * Toast Preview
 * 路由：/preview/toast
 * 展示 Toast 消息提示的所有类型和状态
 */
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { SectionTitle, MetaText } from "@/components/ui/Typography";

function DemoBlock({ title, desc, children }: { title: string; desc?: string; children: React.ReactNode }) {
  return (
    <section className="rounded-[8px] border border-[#e5e5e5] overflow-hidden">
      <header className="flex items-baseline justify-between px-5 py-3 border-b border-[#f0f0f0] bg-[#fafafa]">
        <SectionTitle as="h3" className="!text-sm">{title}</SectionTitle>
        {desc && <MetaText tone="weak">{desc}</MetaText>}
      </header>
      <div className="p-5 flex flex-wrap gap-3">{children}</div>
    </section>
  );
}

export default function ToastPreview() {
  return (
    <div className="min-h-screen bg-[#F8FAFC] p-8">
      <div className="max-w-3xl mx-auto space-y-8">
        <header className="space-y-1">
          <h1 className="text-xl font-semibold text-[#0F172A]">Toast 消息提示</h1>
          <p className="text-sm text-[#64748B]">
            基于 sonner · 白底统一风格 · 关闭按钮右侧 · 顶部居中
          </p>
        </header>

        <DemoBlock title="类型" desc="点击触发对应类型 toast">
          <Button variant="claw-outline" size="claw-sm" onClick={() => toast.success("操作成功")}>
            Success
          </Button>
          <Button variant="claw-outline" size="claw-sm" onClick={() => toast.error("请输入用户 ID")}>
            Error
          </Button>
          <Button variant="claw-outline" size="claw-sm" onClick={() => toast.info("系统将于 10 分钟后维护")}>
            Info
          </Button>
          <Button variant="claw-outline" size="claw-sm" onClick={() => toast.warning("配额即将用尽")}>
            Warning
          </Button>
          <Button variant="claw-outline" size="claw-sm" onClick={() => toast("普通提示消息")}>
            Default
          </Button>
        </DemoBlock>

        <DemoBlock title="长文本" desc="验证换行和关闭按钮位置">
          <Button variant="claw-outline" size="claw-sm" onClick={() => toast.error("操作失败：当前用户没有权限执行此操作，请联系管理员授权后重试。")}>
            触发长文本 Toast
          </Button>
        </DemoBlock>

        <DemoBlock title="视觉规范">
          <div className="text-xs text-[#64748B] space-y-1">
            <p>• 背景：白色 #FFFFFF</p>
            <p>• 边框：#EAEEF4（蓝灰 token）</p>
            <p>• 圆角：12px</p>
            <p>• 内边距：12px 16px</p>
            <p>• 字号：14px / font-medium</p>
            <p>• 关闭按钮：右上角（悬浮在卡片角外）</p>
            <p>• 定位：页面顶部居中</p>
            <p>• z-index：99999</p>
          </div>
        </DemoBlock>
      </div>
    </div>
  );
}
