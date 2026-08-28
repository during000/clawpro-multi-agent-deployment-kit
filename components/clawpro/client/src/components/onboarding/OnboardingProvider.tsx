/**
 * OnboardingProvider
 * 引导体系全局上下文，注入到 App 根层
 */
import { createContext, useContext, type ReactNode } from "react";
import { useOnboardingStore } from "./useOnboardingStore";
import type { GuideState, GuideActions, GuideFlow } from "./types";

interface OnboardingContextValue {
  state: GuideState;
  actions: GuideActions;
  /** 注册的引导流程 */
  flows: GuideFlow[];
  /** 获取当前活跃流程 */
  getActiveFlow: () => GuideFlow | undefined;
}

const OnboardingContext = createContext<OnboardingContextValue | null>(null);

interface OnboardingProviderProps {
  children: ReactNode;
  /** 引导流程配置 */
  flows?: GuideFlow[];
}

export function OnboardingProvider({ children, flows = [] }: OnboardingProviderProps) {
  const { state, actions } = useOnboardingStore();

  const getActiveFlow = () => {
    if (!state.activeFlow) return undefined;
    return flows.find((f) => f.id === state.activeFlow);
  };

  return (
    <OnboardingContext.Provider value={{ state, actions, flows, getActiveFlow }}>
      {children}
    </OnboardingContext.Provider>
  );
}

export function useOnboarding() {
  const ctx = useContext(OnboardingContext);
  if (!ctx) {
    throw new Error("useOnboarding must be used within OnboardingProvider");
  }
  return ctx;
}
