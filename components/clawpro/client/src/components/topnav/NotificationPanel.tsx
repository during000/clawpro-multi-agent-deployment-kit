/**
 * NotificationPanel - 顶部导航的消息通知（右侧抽屉，非模态）
 *
 * 组件结构严格遵循 shadcn/ui Sheet 规范（https://ui.shadcn.com/docs/components/sheet）：
 *   Sheet
 *   ├── SheetTrigger（铃铛按钮）
 *   └── SheetContent
 *       ├── SheetHeader
 *       │   ├── SheetTitle + 全局操作（全部已读 / 全部删除）
 *       │   └── SheetDescription (sr-only)
 *       └── （主体内容：TenantSegment + 列表）
 *
 * 项目设计系统对齐：
 *   - §5.1 L3 浮层（Sheet 内部已使用 var(--shadow-overlay)）
 *   - §8.6 Tab 切换（TenantSegment 用户端胶囊分段）
 *   - §8.1 按钮（Button claw-* / ghost 变体）
 *   - §1.4 五档文字色阶
 */
import React, { useEffect, useState } from "react";
import { Link } from "wouter";
import { Check, Copy, Trash2, X } from "lucide-react";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import {
  TenantSegment,
  TenantSegmentList,
  TenantSegmentItem,
  TenantSegmentContent,
} from "@/components/ui/segment";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { StatusTag } from "@/components/ui/status-tag";
import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyDescription,
} from "@/components/ui/empty";
import { MetaText, MiniBodyText, PanelTitle } from "@/components/ui/Typography";
import { SurfaceCard } from "@/components/ui/Surface";
import NavIconButton from "./NavIconButton";
import { BellIcon } from "./NavIcons";

export type NotificationCategory = "success" | "failure" | "notice";

export interface Notification {
  id: string;
  message: string;
  timestamp: string;
  category: NotificationCategory;
  read: boolean;
  actionHref?: string;
  actionLabel?: string;
}

const CATEGORY_CONFIG: Record<
  NotificationCategory,
  { label: string; variant: "green" | "red" | "blue" }
> = {
  success: { label: "操作成功", variant: "green" },
  failure: { label: "操作报错", variant: "red" },
  notice: { label: "通知公告", variant: "blue" },
};

export interface NotificationPanelProps {
  /** 初始通知列表（组件内部维护本地态：标记已读/删除均为本地，刷新即恢复） */
  notifications: Notification[];
  /** 是否管理员（保留对外参数，便于上层根据角色决定推送哪些通知） */
  isAdmin?: boolean;
}

type TabKey = "all" | NotificationCategory;

export default function NotificationPanel({
  notifications: initialNotifications,
}: NotificationPanelProps) {
  const [notifications, setNotifications] = useState<Notification[]>(initialNotifications);
  const [showPanel, setShowPanel] = useState(false);
  const [activeTab, setActiveTab] = useState<TabKey>("all");
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [hoveredId, setHoveredId] = useState<string | null>(null);

  useEffect(() => {
    setNotifications(initialNotifications);
  }, [initialNotifications]);

  const hasUnread = notifications.some((n) => !n.read);
  const unreadCount = notifications.filter((n) => !n.read).length;

  const handleMarkRead = (id: string) =>
    setNotifications((prev) => prev.map((n) => (n.id === id ? { ...n, read: true } : n)));

  const handleMarkAllRead = () =>
    setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));

  const handleDelete = (id: string) =>
    setNotifications((prev) => prev.filter((n) => n.id !== id));

  const handleClearAll = () => setNotifications([]);

  const handleCopy = (notif: Notification) => {
    navigator.clipboard.writeText(notif.message).then(() => {
      setCopiedId(notif.id);
      handleMarkRead(notif.id);
      setTimeout(() => setCopiedId(null), 1000);
    });
  };

  const tabs: { key: TabKey; label: string }[] = [
    { key: "all", label: "全部" },
    { key: "notice", label: "通知公告" },
    { key: "failure", label: "操作报错" },
    { key: "success", label: "操作成功" },
  ];


  const listFor = (key: TabKey) =>
    key === "all" ? notifications : notifications.filter((n) => n.category === key);

  // ─── 单条通知渲染（被各 Tab 复用） ──────────────────────────────────────
  const renderList = (list: Notification[]) => {
    if (list.length === 0) {
      return (
        <Empty className="border-0 py-16">
          <EmptyHeader>
            <EmptyMedia />
            <EmptyDescription>暂无消息</EmptyDescription>
          </EmptyHeader>
        </Empty>
      );
    }

    return (
      <div className="flex flex-col gap-2 p-3">
        {list.map((notif) => {
          const catCfg = CATEGORY_CONFIG[notif.category];
          const isCopied = copiedId === notif.id;
          const isClickable = !!notif.actionHref;

          // 卡片本体（可点击查看的卡片才显示 cursor-pointer）
          const cardBody = (
            <SurfaceCard
              className={`bg-[var(--card)] px-3 py-2.5 border border-[var(--border)] transition-colors hover:bg-[var(--accent)] ${isClickable ? "cursor-pointer" : ""}`}
              onMouseEnter={() => setHoveredId(notif.id)}
              onMouseLeave={() => setHoveredId(null)}
            >
              <div>
                <MiniBodyText
                  as="p"
                  tone={notif.read ? "weak" : "secondary"}
                  className="leading-relaxed line-clamp-2 transition-colors"
                  title={notif.message}
                >
                  {notif.message}
                </MiniBodyText>
              </div>
              <div className="flex items-center justify-between mt-1.5">
                <div className="flex items-center gap-1.5">
                  <StatusTag mode="fill" variant={catCfg.variant}>
                    {catCfg.label}
                  </StatusTag>
                  <MetaText tone="weak">{notif.timestamp}</MetaText>
                </div>
                <div className="flex items-center gap-2">
                  <TooltipProvider>
                  {!notif.read && (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => {
                            e.stopPropagation();
                            e.preventDefault();
                            handleMarkRead(notif.id);
                          }}
                          className={`h-6 w-6 p-0 text-[var(--text-muted)] hover:text-[var(--text-emphasis)] hover:bg-[var(--accent)] [&_svg]:size-3.5 transition-opacity ${hoveredId === notif.id ? "opacity-100" : "opacity-0 pointer-events-none"}`}
                          aria-label="标为已读"
                        >
                          <Check strokeWidth={3} />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>已读</TooltipContent>
                    </Tooltip>
                  )}
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          e.preventDefault();
                          handleDelete(notif.id);
                        }}
                        className={`h-6 w-6 p-0 text-[var(--text-muted)] hover:text-[var(--text-danger)] hover:bg-[var(--accent)] [&_svg]:size-3.5 transition-opacity ${hoveredId === notif.id ? "opacity-100" : "opacity-0 pointer-events-none"}`}
                        aria-label="删除"
                      >
                        <Trash2 />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>删除</TooltipContent>
                  </Tooltip>
                  {notif.category === "failure" && (
                    <Tooltip open={isCopied || undefined}>
                      <TooltipTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => {
                            e.stopPropagation();
                            e.preventDefault();
                            handleCopy(notif);
                          }}
                          className={`h-6 w-6 p-0 text-[var(--text-muted)] hover:text-[var(--text-emphasis)] hover:bg-[var(--accent)] [&_svg]:size-3.5 transition-opacity ${hoveredId === notif.id || isCopied ? "opacity-100" : "opacity-0 pointer-events-none"}`}
                          aria-label="复制"
                        >
                          <Copy />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{isCopied ? "复制成功" : "复制"}</TooltipContent>
                    </Tooltip>
                  )}
                  </TooltipProvider>
                </div>
              </div>
            </SurfaceCard>
          );

          if (!isClickable) {
            return <React.Fragment key={notif.id}>{cardBody}</React.Fragment>;
          }

          // 可跳转的卡片：整张卡片可点击跳转
          return (
            <Link
              key={notif.id}
              href={notif.actionHref!}
              onClick={() => {
                handleMarkRead(notif.id);
                setShowPanel(false);
              }}
              className="block"
            >
              {cardBody}
            </Link>
          );
        })}
      </div>
    );
  };

  return (
    <Sheet open={showPanel} onOpenChange={setShowPanel} modal={false}>
      <Tooltip>
        <TooltipTrigger asChild>
          <SheetTrigger asChild>
            <NavIconButton
              icon={
                <span className="inline-flex items-start gap-[2px]">
                  <BellIcon />
                  {hasUnread && (
                    <span className="-mt-1 inline-flex h-[14px] min-w-[16px] items-center justify-center rounded-full bg-[var(--accent)] px-1 text-[10px] font-medium leading-4 text-[var(--text-emphasis)]">
                      {unreadCount > 99 ? "99+" : unreadCount}
                    </span>
                  )}
                </span>
              }
              title="消息通知"
            />
          </SheetTrigger>
        </TooltipTrigger>
        <TooltipContent side="bottom" sideOffset={6}>
          消息通知
        </TooltipContent>
      </Tooltip>

      <SheetContent
        side="right"
        showOverlay={false}
        className="!w-[420px] !max-w-none !top-[64px] !bottom-0 !h-[calc(100vh-64px)] p-0 flex flex-col gap-0 border-t [&>[data-slot=sheet-close]]:hidden"
      >
        {/* ───── shadcn 规范：SheetHeader > SheetTitle + SheetDescription ───── */}
        <SheetHeader className="px-5 pt-5 pb-4 border-b border-[var(--border)] gap-0 space-y-0">
          <div className="flex items-center justify-between">
            <SheetTitle asChild>
              <PanelTitle>消息通知</PanelTitle>
            </SheetTitle>
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleMarkAllRead}
                disabled={!hasUnread}
                className="h-7 px-2 gap-1 text-xs text-[var(--text-emphasis)] hover:text-[var(--text-brand)] hover:bg-[var(--accent)]"
              >
                <Check className="w-3.5 h-3.5" strokeWidth={3} />
                全部已读
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleClearAll}
                disabled={notifications.length === 0}
                className="h-7 px-2 gap-1 text-xs text-[var(--text-emphasis)] hover:text-[var(--text-danger)] hover:bg-[var(--accent)]"
              >
                <Trash2 className="w-3.5 h-3.5" />
                全部删除
              </Button>
              <SheetClose asChild>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="关闭"
                  className="h-7 w-7 text-[var(--text-muted)] hover:text-[var(--text-emphasis)]"
                >
                  <X className="w-4 h-4" />
                </Button>
              </SheetClose>
            </div>
          </div>
          <SheetDescription className="sr-only">
            查看与管理您的系统通知，包括操作成功、操作报错、通知公告三大类。
          </SheetDescription>
        </SheetHeader>

        {/* ───── 主体内容：TenantSegment（用户端胶囊分段）+ 滚动列表 ───── */}
        <TenantSegment
          value={activeTab}
          onValueChange={(v) => setActiveTab(v as TabKey)}
          className="flex flex-col flex-1 min-h-0 gap-0"
        >
          <div className="px-3 pt-3 pb-1">
            <TenantSegmentList className="w-full">
              {tabs.map((tab) => {
                const count = tab.key === "all"
                  ? notifications.filter((n) => !n.read).length
                  : notifications.filter((n) => n.category === tab.key && !n.read).length;
                return (
                  <TenantSegmentItem
                    key={tab.key}
                    value={tab.key}
                    className="flex-1 text-xs"
                  >
                    {tab.label}
                    {count > 0 && (
                      <span className="text-[var(--text-muted)]">{count}</span>
                    )}
                  </TenantSegmentItem>
                );
              })}
            </TenantSegmentList>
          </div>

          <div
            className="overflow-y-auto flex-1 min-h-0"
            style={{ scrollbarWidth: "thin", scrollbarColor: "#d1d5db #f3f4f6" }}
          >
            {tabs.map((tab) => (
              <TenantSegmentContent key={tab.key} value={tab.key} className="m-0">
                {renderList(listFor(tab.key))}
              </TenantSegmentContent>
            ))}
          </div>
        </TenantSegment>
      </SheetContent>
    </Sheet>
  );
}
