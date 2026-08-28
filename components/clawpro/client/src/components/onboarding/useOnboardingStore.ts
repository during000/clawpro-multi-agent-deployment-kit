/**
 * Onboarding Guide Store
 * 使用 React useState + Context 实现（避免额外引入 zustand 依赖）
 */
import { useCallback, useMemo, useState } from "react";
import type { GuideState, GuideActions } from "./types";

const STORAGE_KEY = "openclaw-onboarding-state";

function loadState(): GuideState {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw);
  } catch {}
  return getDefaultState();
}

function getDefaultState(): GuideState {
  return {
    isSimulating: false,
    completedFlows: [],
    skippedFlows: [],
    activeFlow: null,
    activeStepIndex: 0,
    showDevPanel: false,
  };
}

function saveState(state: GuideState) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {}
}

export function useOnboardingStore() {
  const [state, setState] = useState<GuideState>(loadState);

  const updateState = useCallback((updater: (prev: GuideState) => GuideState) => {
    setState((prev) => {
      const next = updater(prev);
      saveState(next);
      return next;
    });
  }, []);

  const actions: GuideActions = useMemo(
    () => ({
      startSimulation: () =>
        updateState((s) => ({
          ...getDefaultState(),
          isSimulating: true,
          showDevPanel: s.showDevPanel,
        })),

      stopSimulation: () =>
        updateState((s) => ({
          ...s,
          isSimulating: false,
          activeFlow: null,
          activeStepIndex: 0,
        })),

      startFlow: (flowId: string) =>
        updateState((s) => ({
          ...s,
          activeFlow: flowId,
          activeStepIndex: 0,
        })),

      completeFlow: () =>
        updateState((s) => ({
          ...s,
          completedFlows: s.activeFlow
            ? [...s.completedFlows, s.activeFlow]
            : s.completedFlows,
          activeFlow: null,
          activeStepIndex: 0,
        })),

      skipFlow: () =>
        updateState((s) => ({
          ...s,
          skippedFlows: s.activeFlow
            ? [...s.skippedFlows, s.activeFlow]
            : s.skippedFlows,
          activeFlow: null,
          activeStepIndex: 0,
        })),

      nextStep: () =>
        updateState((s) => ({
          ...s,
          activeStepIndex: s.activeStepIndex + 1,
        })),

      prevStep: () =>
        updateState((s) => ({
          ...s,
          activeStepIndex: Math.max(0, s.activeStepIndex - 1),
        })),

      goToStep: (index: number) =>
        updateState((s) => ({
          ...s,
          activeStepIndex: index,
        })),

      resetAll: () =>
        updateState((s) => ({
          ...getDefaultState(),
          isSimulating: s.isSimulating,
          showDevPanel: s.showDevPanel,
        })),

      toggleDevPanel: () =>
        updateState((s) => ({
          ...s,
          showDevPanel: !s.showDevPanel,
        })),
    }),
    [updateState],
  );

  return { state, actions };
}
