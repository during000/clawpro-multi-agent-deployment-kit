/**
 * AllUsersTag - 「全部用户」标签组件
 *
 * 管控端所有"全部用户"展示均使用此组件，确保全局样式一致。
 * 基于 Badge variant="outline"（白底描边）。
 *
 * 用法：
 *   <AllUsersTag />
 */
import { Badge } from "@/components/ui/badge";

function AllUsersTag() {
  return <Badge variant="outline">全部用户</Badge>;
}

export { AllUsersTag };
