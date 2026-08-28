/**
 * Onboarding Guide System - Type Definitions
 * 新手引导体系类型定义
 */

/** 引导组件类型 - 对应图示原型的 7 种组件 */
export type GuideComponentType =
  | "global-modal"        // 影响面极大的更新：全局弹窗
  | "module-float"        // 模块级更新：非阻断性浮窗
  | "nav-bubble"          // 新功能预览：导航功能预览介绍气泡
  | "point-bubble"        // 功能级更新：点对点的指引气泡
  | "update-bar"          // 全局更新：轻量更新概览提示条
  | "changelog-drawer"    // 更新记录：侧边抽屉
  | "highlight-bubble";   // 单页面结构/UI元素变化：高亮+气泡

/** 端类型 */
export type GuideEndpoint = "admin" | "tenant";

/** 版本更新场景层级 */
export type SceneLayer =
  | "structure"   // 结构层：页面结构级别的变化
  | "element"     // 元素层：UI 元素级别的变化
  | "logic"       // 逻辑层：不可见逻辑级别的变化
  | "system"      // 系统层：账号/安全级别的变化
  | "cross-end";  // 跨端层：双端联动级别的变化

/** 场景编号 */
export type SceneCode =
  | "1.1" | "1.2" | "1.3" | "1.4" | "1.5"   // 结构层
  | "2.1" | "2.2" | "2.3" | "2.4" | "2.5" | "2.6"  // 元素层
  | "3.1" | "3.2" | "3.3"                    // 逻辑层
  | "4.1" | "4.2" | "4.3"                    // 系统层
  | "5.1" | "5.2";                           // 跨端层

/** 引导步骤定义 */
export interface GuideStep {
  id: string;
  /** 步骤标题 */
  title: string;
  /** 步骤描述 */
  description: string;
  /** 目标元素选择器（用于 Spotlight / Bubble 定位） */
  targetSelector?: string;
  /** 引导组件类型 */
  componentType: GuideComponentType;
  /** 气泡/浮窗位置 */
  placement?: "top" | "bottom" | "left" | "right" | "center";
  /** 可选图片/截图 */
  image?: string;
  /** 自定义操作按钮 */
  actions?: GuideAction[];
}

/** 引导操作按钮 */
export interface GuideAction {
  label: string;
  type: "primary" | "secondary" | "link";
  onClick?: () => void;
  href?: string;
}

/** 引导流程定义 */
export interface GuideFlow {
  id: string;
  /** 流程名称 */
  name: string;
  /** 版本号 */
  version: string;
  /** 适用端 */
  endpoint: GuideEndpoint;
  /** 场景层级 */
  layer: SceneLayer;
  /** 场景编号 */
  sceneCode: SceneCode;
  /** 步骤列表 */
  steps: GuideStep[];
  /** 触发条件 */
  trigger: GuideTrigger;
  /** 优先级（数字越小越高） */
  priority: number;
}

/** 引导触发条件 */
export interface GuideTrigger {
  /** 触发类型 */
  type: "page-enter" | "first-visit" | "login-after-update" | "manual";
  /** 触发路径（正则匹配） */
  pathPattern?: string;
  /** 生效起始时间 */
  validFrom?: string;
  /** 失效时间 */
  validUntil?: string;
}

/** 引导状态 */
export interface GuideState {
  /** 当前是否处于模拟新手模式 */
  isSimulating: boolean;
  /** 已完成的流程 ID 列表 */
  completedFlows: string[];
  /** 已跳过的流程 ID 列表 */
  skippedFlows: string[];
  /** 当前正在播放的流程 */
  activeFlow: string | null;
  /** 当前步骤索引 */
  activeStepIndex: number;
  /** 是否显示开发者面板 */
  showDevPanel: boolean;
}

/** Store Actions */
export interface GuideActions {
  /** 开启模拟新手模式 */
  startSimulation: () => void;
  /** 关闭模拟新手模式 */
  stopSimulation: () => void;
  /** 开始一个引导流程 */
  startFlow: (flowId: string) => void;
  /** 完成当前流程 */
  completeFlow: () => void;
  /** 跳过当前流程 */
  skipFlow: () => void;
  /** 下一步 */
  nextStep: () => void;
  /** 上一步 */
  prevStep: () => void;
  /** 跳转到指定步骤 */
  goToStep: (index: number) => void;
  /** 重置所有进度 */
  resetAll: () => void;
  /** 切换开发者面板 */
  toggleDevPanel: () => void;
}
