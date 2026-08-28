/**
 * AdminModeContext - 管控端模式状态管理
 * 支持「普通（custom）」「OneID 专用（standard）」「统一（unified）」三种模式
 * 默认为普通模式，状态持久化到 localStorage
 *
 * hasOneid 在 standard 与 unified 下均为 true，用于让"统一"模式继承 OneID 视图基础；
 * 普通模式独有的功能在 MemberManagement 中通过 isUnified 叠加。
 */
import { createContext, useContext, useState, ReactNode } from "react";

export type AdminMode = "standard" | "custom" | "unified";

interface AdminModeContextValue {
  mode: AdminMode;
  setMode: (mode: AdminMode) => void;
  isStandard: boolean;
  isCustom: boolean;
  isUnified: boolean;
  hasOneid: boolean;
}

const AdminModeContext = createContext<AdminModeContextValue | null>(null);

const STORAGE_KEY = "openclaw_admin_mode";

export function AdminModeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<AdminMode>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === "standard" || saved === "unified" || saved === "custom") {
      return saved;
    }
    return "custom";
  });

  const setMode = (newMode: AdminMode) => {
    setModeState(newMode);
    localStorage.setItem(STORAGE_KEY, newMode);
  };

  return (
    <AdminModeContext.Provider
      value={{
        mode,
        setMode,
        isStandard: mode === "standard",
        isCustom: mode === "custom",
        isUnified: mode === "unified",
        hasOneid: mode === "standard" || mode === "unified",
      }}
    >
      {children}
    </AdminModeContext.Provider>
  );
}

export function useAdminMode() {
  const ctx = useContext(AdminModeContext);
  if (!ctx) throw new Error("useAdminMode must be used within AdminModeProvider");
  return ctx;
}
