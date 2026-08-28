/**
 * Alert Preview
 * 路由：/preview/alert
 * 展示 Alert 的 4 个 variant（info / operation-info / warning / product-news）
 * 以及右侧操作区变体、AdminNoticeAlert（产品动态 / 待配置 / 资源告警）。
 *
 * 文案随机，仅供视觉走查与组合演示。
 */
import { useState } from "react";
import { CircleAlert } from "lucide-react";

import {
  Alert,
  AlertDescription,
  AlertInfoIcon,
  AlertOperationInfoIcon,
  AlertProductNewsIcon,
  AlertTitle,
} from "@/components/ui/alert";
import { AdminNoticeAlert } from "@/components/ui/admin-notice-alert";
import { Button } from "@/components/ui/button";
import { MetaText, SectionTitle } from "@/components/ui/Typography";

function DemoBlock({
  title,
  desc,
  children,
}: {
  title: string;
  desc?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-[8px] border border-[#e5e5e5] overflow-hidden">
      <header className="flex items-baseline justify-between px-5 py-3 border-b border-[#f0f0f0] bg-[#fafafa]">
        <SectionTitle as="h3" className="!text-sm">
          {title}
        </SectionTitle>
        {desc ? <MetaText tone="weak">{desc}</MetaText> : null}
      </header>
      <div className="p-5 space-y-3">{children}</div>
    </section>
  );
}

export default function AlertPreview() {
  const [pageIndex] = useState(1);
  const demoControls = (
    <span className="text-[12px] text-[var(--text-emphasis)] tabular-nums">
      {pageIndex}/5
    </span>
  );

  return (
    <div className="min-h-screen bg-[#F8FAFC] p-8">
      <div className="max-w-3xl mx-auto space-y-8">
        <header className="space-y-1">
          <h1 className="text-xl font-semibold text-[#0F172A]">Alert 提示条</h1>
          <p className="text-sm text-[#64748B]">
            页面常驻信息提示 · 4 个 variant + 右侧操作区 + AdminNoticeAlert（管控端顶部公告）
          </p>
        </header>

        <DemoBlock title="Info · 标准信息提示" desc="蓝底蓝边，用于页面常驻说明 / 功能告知">
          <Alert variant="info">
            <AlertInfoIcon />
            <AlertDescription>
              当前 Agent 仅在「研发联调环境」可见，发布前请前往「Agent 配置 → 可见范围」勾选目标空间。
            </AlertDescription>
          </Alert>

          <Alert variant="info">
            <AlertInfoIcon />
            <AlertTitle>免登录调用已启用</AlertTitle>
            <AlertDescription>
              下一次会话起，企业内成员通过统一登录访问本 Agent 时将不再提示鉴权确认。
            </AlertDescription>
          </Alert>
        </DemoBlock>

        <DemoBlock title="Operation-Info · 操作说明" desc="白底灰边，用于操作上下文 / 表单辅助说明">
          <Alert variant="operation-info">
            <AlertOperationInfoIcon />
            <AlertTitle>批量导入说明</AlertTitle>
            <AlertDescription>
              单次最多上传 200 行，文件需为 UTF-8 编码的 CSV。列顺序请严格按照模板：用户 ID / 邮箱 / 角色。
            </AlertDescription>
          </Alert>

          <Alert variant="operation-info">
            <AlertOperationInfoIcon />
            <AlertDescription>
              已选择 12 项资源，下方操作仅会作用于选中项，未选中项不会被修改。
            </AlertDescription>
          </Alert>
        </DemoBlock>

        <DemoBlock title="Warning · 标准警告" desc="橙底橙边，用于配置缺失 / 风险提示。图标固定为 CircleAlert">
          <Alert variant="warning">
            <CircleAlert />
            <AlertDescription>
              检测到模型配额将在 3 小时内耗尽，请尽快续期，否则租户端的 Agent 调用会被自动拒绝。
            </AlertDescription>
          </Alert>

          <Alert variant="warning">
            <CircleAlert />
            <AlertTitle>有 2 项基础配置未完成</AlertTitle>
            <AlertDescription>
              「企业用户导入」「至少一个模型渠道」尚未配置完毕，未完成前用户端无法正常登录使用。
            </AlertDescription>
          </Alert>
        </DemoBlock>

        <DemoBlock title="Product-News · 产品动态" desc="蓝底蓝边 + 闪光图标，用于版本发布 / 功能上线">
          <Alert variant="product-news">
            <AlertProductNewsIcon />
            <AlertDescription>
              【产品动态】OpenClaw v2.4.0 已发布：新增「记忆管理」与「会话回放」，详见发布说明。
            </AlertDescription>
          </Alert>

          <Alert variant="product-news">
            <AlertProductNewsIcon />
            <AlertDescription>
              【产品动态】Agent 工具库新增 14 款官方工具（含飞书 / 钉钉 / 企微 / Jira），即日起可在「工具配置」启用。
            </AlertDescription>
          </Alert>
        </DemoBlock>

        <DemoBlock title="带右侧操作区" desc="grid 布局扩为 3 列，操作按钮放在末列">
          <Alert
            variant="warning"
            className="has-[>svg]:grid-cols-[16px_minmax(0,1fr)_auto] gap-y-0 items-center"
          >
            <CircleAlert />
            <AlertDescription className="flex min-w-0 items-baseline flex-wrap gap-x-1 leading-[1.5]">
              安全组「prod-default」存在 1 条 0.0.0.0/0 入站规则，建议立即收敛。
            </AlertDescription>
            <div className="col-start-3 shrink-0">
              <Button variant="claw-outline" size="claw-sm">
                前往处理
              </Button>
            </div>
          </Alert>

          <Alert
            variant="info"
            className="has-[>svg]:grid-cols-[16px_minmax(0,1fr)_auto] gap-y-0 items-center"
          >
            <AlertInfoIcon />
            <AlertDescription className="flex min-w-0 items-baseline flex-wrap gap-x-1 leading-[1.5]">
              本月用量已达套餐 80%，可提前升级以避免次月限流。
            </AlertDescription>
            <div className="col-start-3 shrink-0">
              <Button variant="claw-outline" size="claw-sm">
                查看详情
              </Button>
            </div>
          </Alert>
        </DemoBlock>

        <DemoBlock title="AdminNoticeAlert · 管控端顶部常驻公告条" desc="高度 40px / 圆角 4px / 半透白底 / 翻页控件 + 关闭按钮">
          <div className="rounded-[4px] bg-[linear-gradient(180deg,#F7FAFF_0%,#EEF4FB_100%)] px-5 py-4">
            <div className="space-y-3">
              <AdminNoticeAlert type="product-news" controls={demoControls}>
                <span>OpenClaw v2.4.0 已发布：记忆管理 / 会话回放 / 工具库官方工具集 全量上线。</span>
              </AdminNoticeAlert>

              <AdminNoticeAlert type="pending-config" controls={demoControls}>
                <span>有 3 项基础配置未完成（导入企业用户、配置至少一个模型渠道、配置安全组），未完成将影响用户端正常使用，</span>
                <span className="font-medium text-[var(--text-emphasis)] underline underline-offset-2">
                  前往基础信息配置处理
                </span>
              </AdminNoticeAlert>

              <AdminNoticeAlert type="resource-alert" controls={demoControls}>
                <span>私有网络（VPC）配额已耗尽，将影响用户端云设备的正常创建与使用，</span>
                <span className="text-[var(--text-emphasis)] underline underline-offset-2">
                  前往腾讯云控制台提交工单
                </span>
              </AdminNoticeAlert>
            </div>
          </div>
        </DemoBlock>

        <DemoBlock title="视觉规范">
          <div className="text-xs text-[#64748B] space-y-1">
            <p>• 圆角：4px（var(--alert-radius)）</p>
            <p>• 内边距：左右 16px / 上下 10px（px-4 py-2.5）</p>
            <p>• 图标：16px，列宽 16px，与文字间距 8px，translate-y-px 与 12/18 行高首行居中</p>
            <p>• AlertTitle：MetaMedium（12px / medium / 1.5）</p>
            <p>• AlertDescription：MetaText（12px / regular / 1.5 / tone=&quot;inherit&quot;）</p>
            <p>• 标题与描述间距：gap-y-1（4px）</p>
            <p>• 颜色：通过 --alert-* token，禁止硬编码色值</p>
            <p>• Info 图标必须使用 AlertInfoIcon；Warning 图标必须使用 CircleAlert</p>
          </div>
        </DemoBlock>
      </div>
    </div>
  );
}
