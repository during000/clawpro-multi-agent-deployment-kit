/**
 * TenantSection — 用户端「段」标准容器
 *
 * 统一规范（修订版：标题进卡内）：
 *   1. 标题 + 操作按钮 + 内容**都**在 TenantCard 里
 *   2. 标题在卡内顶部、操作按钮跟标题同一行（左标题、右按钮）
 *   3. 标题与内容之间无分割线，仅靠间距区分
 *
 * 用法：
 *   ```tsx
 *   <TenantSection
 *     title="MCP 配置"
 *     actions={
 *       <>
 *         <Input tenant placeholder="搜索…" />
 *         <Button>添加 MCP</Button>
 *       </>
 *     }
 *   >
 *     <Alert>…</Alert>
 *     <Grid>…</Grid>
 *   </TenantSection>
 *   ```
 *
 * 进阶：
 *   - `bare`        ：不包 TenantCard，children 直接渲染（少数特殊场景）
 *   - `cardPadding` ：透传给 TenantCard 的 padding（默认 "default"）
 *   - `headingLevel`：标题层级，默认 "section"（SectionTitle 18px）；可选 "panel"（PanelTitle 16px）
 */
import { ReactNode } from "react";
import { TenantCard } from "@/components/ui/Surface";
import { SectionTitle, PanelTitle } from "@/components/ui/Typography";
import { cn } from "@/lib/utils";

interface TenantSectionProps {
  /** 段标题，默认渲染为 SectionTitle（18px Medium） */
  title?: ReactNode;
  /** 标题右侧操作区：搜索框 / 按钮 / 视图切换等 */
  actions?: ReactNode;
  /** 标题层级：section=18px Medium（默认）/ panel=16px Semibold */
  headingLevel?: "section" | "panel";
  /** 不包 TenantCard，children 直接裸渲染（少数特殊场景） */
  bare?: boolean;
  /** 透传给 TenantCard 的 padding（仅 bare=false 生效） */
  cardPadding?: "default" | "compact" | "none";
  /** 标题行与下方内容的间距（标题进卡内后不再需要"段—卡"间距），默认 16px */
  gap?: number;
  /** 整段 className（最外层；bare=false 时即 TenantCard） */
  className?: string;
  /** 段内容 */
  children?: ReactNode;
}

export function TenantSection({
  title,
  actions,
  headingLevel = "section",
  bare = false,
  cardPadding = "default",
  gap = 16,
  className,
  children,
}: TenantSectionProps) {
  const Heading = headingLevel === "panel" ? PanelTitle : SectionTitle;

  /** 段头：左标题 + 右操作（无分割线） */
  const header = (title || actions) ? (
    <div className="flex items-center gap-3">
      {title ? <Heading className="flex-1 min-w-0">{title}</Heading> : <div className="flex-1" />}
      {actions && (
        <div className="flex items-center gap-2 shrink-0">{actions}</div>
      )}
    </div>
  ) : null;

  if (bare) {
    return (
      <section className={cn("flex flex-col", className)} style={{ gap }}>
        {header}
        {children}
      </section>
    );
  }

  return (
    <TenantCard padding={cardPadding} className={className}>
      <div className="flex flex-col" style={{ gap }}>
        {header}
        {children}
      </div>
    </TenantCard>
  );
}

export default TenantSection;
