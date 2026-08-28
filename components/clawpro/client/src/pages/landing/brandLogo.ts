/**
 * brandLogo - 全站品牌 Logo 共享存储
 *
 * 三处使用同一个 Logo 来源，确保用户上传一次即可同步更新：
 *   1. 顶部导航栏（Navbar）
 *   2. Banner 中央图标卡片（Hero）
 *   3. 底部页脚（Footer）
 *
 * 数据持久化在 localStorage（key: "yh-brand-logo"）。
 * 通过自定义事件 "yh-brand-logo-change" + storage 跨标签同步通知所有订阅者。
 */

const STORAGE_KEY = "yh-brand-logo";
const EVENT_NAME = "yh-brand-logo-change";

/** 默认 Logo（永辉品牌 logo.png） */
export const DEFAULT_BRAND_LOGO = "/landing-assets/yh-features/logo.png?v=2";

/** 读取当前 Logo
 *  注意：早期版本曾把不带 ?v=N 后缀的同路径写进 localStorage；
 *  这里如果检测到旧默认值，自动覆盖为最新版本，避免用户看到老 logo。 */
export function getBrandLogo(): string {
  if (typeof window === "undefined") return DEFAULT_BRAND_LOGO;
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) return DEFAULT_BRAND_LOGO;
    // 兼容历史版本：旧路径不带 ?v= 时，按最新默认值返回
    if (stored === "/landing-assets/yh-features/logo.png") {
      return DEFAULT_BRAND_LOGO;
    }
    return stored;
  } catch {
    return DEFAULT_BRAND_LOGO;
  }
}

/** 写入新 Logo（通常是 dataURL 或资源 URL），并通知所有订阅者 */
export function setBrandLogo(src: string): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, src);
  } catch {
    /* 容量超限或隐私模式时静默失败，仍然广播事件让本页渲染更新 */
  }
  window.dispatchEvent(new CustomEvent(EVENT_NAME, { detail: src }));
}

/** 重置为默认 Logo */
export function resetBrandLogo(): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    /* noop */
  }
  window.dispatchEvent(
    new CustomEvent(EVENT_NAME, { detail: DEFAULT_BRAND_LOGO })
  );
}

/**
 * 订阅 Logo 变化（同窗口 + 跨标签）。返回取消订阅函数。
 */
export function subscribeBrandLogo(cb: (src: string) => void): () => void {
  if (typeof window === "undefined") return () => {};

  const onCustom = (e: Event) => {
    const ce = e as CustomEvent<string>;
    cb(ce.detail || DEFAULT_BRAND_LOGO);
  };
  const onStorage = (e: StorageEvent) => {
    if (e.key === STORAGE_KEY) cb(e.newValue || DEFAULT_BRAND_LOGO);
  };

  window.addEventListener(EVENT_NAME, onCustom);
  window.addEventListener("storage", onStorage);
  return () => {
    window.removeEventListener(EVENT_NAME, onCustom);
    window.removeEventListener("storage", onStorage);
  };
}
