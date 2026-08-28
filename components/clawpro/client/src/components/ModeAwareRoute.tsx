/**
 * ModeAwareRoute - 模式感知路由包装组件
 * 根据当前管控端模式（standard / custom / unified）渲染不同的页面组件
 * - standard：渲染 standard
 * - custom / unified：渲染 custom
 *   （unified "统一"模式在该路由场景下与普通模式保持一致）
 */
import { useAdminMode } from "@/contexts/AdminModeContext";
import { ReactNode } from "react";

interface ModeAwareRouteProps {
  standard: ReactNode;
  custom: ReactNode;
}

export default function ModeAwareRoute({ standard, custom }: ModeAwareRouteProps) {
  const { isStandard } = useAdminMode();
  return <>{isStandard ? standard : custom}</>;
}
