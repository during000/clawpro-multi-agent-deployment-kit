/**
 * Footer - 永辉版页脚
 */

export default function Footer() {

  return (
    <footer className="yh-footer">
      <div className="yh-footer-inner">
        <div className="yh-footer-brand">
          <img
            src="/landing-assets/yh-features/brand-logo.png"
            alt=""
            width={24}
            height={24}
            className="yh-footer-logo"
          />
          <span>ClawPro 智能体体验平台</span>
        </div>
        <div className="yh-footer-copy">
          Copyright © 2013 - 2026 Tencent Cloud. All Rights Reserved. 腾讯云 版权所有
        </div>
      </div>
    </footer>
  );
}
