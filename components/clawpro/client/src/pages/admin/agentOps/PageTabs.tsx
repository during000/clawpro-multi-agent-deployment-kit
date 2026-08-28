/**
 * @deprecated 请直接使用 `@/components/ui/line-tabs` 中的 `LineTabs`。
 *
 * 本文件已迁移为 `LineTabs` 的兼容代理（API 不变），仅为保持 AgentCommandsPage 等
 * 现有调用点不被破坏。新页面请直接 import `LineTabs`。
 */
import { LineTabs, type LineTabDef } from "@/components/ui/line-tabs";

interface Props<T extends string> {
  tabs: ReadonlyArray<LineTabDef<T>>;
  active: T;
  onChange: (id: T) => void;
  description?: string;
}

export default function PageTabs<T extends string>(props: Props<T>) {
  return <LineTabs {...props} />;
}
