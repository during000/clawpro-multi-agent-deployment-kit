/**
 * useAdminDisabled - 管控端停服禁用态 hook
 * 
 * 在管控端页面中使用，快速获取当前是否禁用 + 通用禁用样式 props
 */
import { useServiceStatus } from "@/contexts/ServiceStatusContext";

export function useAdminDisabled() {
  const { isAdminDisabled } = useServiceStatus();

  /** 通用禁用态 props，直接展开到 Button 上 */
  const disabledProps = isAdminDisabled
    ? {
        disabled: true,
        className: "opacity-40 cursor-not-allowed",
        title: "管控台已到期，请续费后操作",
      }
    : {};

  return {
    isAdminDisabled,
    disabledProps,
    disabledTip: "管控台已到期，请续费后操作",
  };
}
