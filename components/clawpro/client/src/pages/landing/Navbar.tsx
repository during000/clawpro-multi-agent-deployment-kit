/**
 * Navbar - 永辉版顶部导航栏
 * 1920px 容器 + 64px 高度 + 8px 32px 内边距 + space-between + 1px 描边 + 半透明白底 + 模糊
 * Logo 由全局 brandLogo store 提供，与 Hero 中央图标、Footer logo 同步更新
 */
import { useLocation } from "wouter";

export default function Navbar() {
  const [, navigate] = useLocation();

  return (
    <nav className="yh-navbar">
      <div className="yh-navbar-inner">
        {/* 左侧 Logo + 标题 */}
        <div className="yh-navbar-brand">
          <img
            src="/landing-assets/yh-features/brand-logo.png"
            alt="Logo"
            width={28}
            height={28}
            className="yh-navbar-logo"
          />
          <span className="yh-navbar-title">ClawPro 智能体体验平台</span>
        </div>

        {/* 右侧按钮组 */}
        <div className="yh-navbar-actions">
          <button
            className="yh-btn yh-btn-outline"
            onClick={() => navigate("/my-openclaw")}
          >
            进入我的 Agent
          </button>
          <button
            className="yh-btn yh-btn-primary"
            onClick={() => navigate("/admin/basic-info")}
          >
            管理后台
          </button>
        </div>
      </div>
    </nav>
  );
}
