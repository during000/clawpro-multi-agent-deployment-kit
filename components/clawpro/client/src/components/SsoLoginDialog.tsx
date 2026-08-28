import React, { useCallback, useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { ArrowLeft } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  PASSWORD_RULES_HINT,
  validatePasswordStrength,
} from '@/lib/password-rules';
import { toast, withClose } from '@/components/ui/sonner';

interface SsoLoginDialogProps {
  visible: boolean;
  onClose: () => void;
}

interface SsoImOption {
  type: string;
  label: string;
  iconUrl: string;
}

/** 演示用 SSO 供应商列表 */
const SSO_IM_OPTIONS: SsoImOption[] = [
  { type: 'feishu', label: '飞书', iconUrl: 'https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/lark-v2-logo.png' },
  { type: 'dingtalk', label: '钉钉', iconUrl: 'https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/dd-v2-logo.png' },
  { type: 'aad', label: '微软Entra ID', iconUrl: 'https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/aad-v2-logo.png' },
  { type: 'saml', label: 'SAML2.0', iconUrl: 'https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/saml-v2-logo.png' },
  { type: 'ad', label: 'Windows AD', iconUrl: 'https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/ad-v2-logo.png' },
  { type: 'wework-private', label: '私有化企微', iconUrl: 'https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/wework-logo.png' },
  { type: 'oidc', label: 'OIDC', iconUrl: 'https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/oidc-v2-logo.png' },
  { type: 'jwt', label: 'JWT', iconUrl: 'https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/jwt-v2-logo.png' },
  { type: 'openldap', label: 'OpenLDAP', iconUrl: 'https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/openldap-v2-logo.png' },
  { type: 'wecom', label: '企业微信', iconUrl: 'https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/wework-v2-logo.png' },
  { type: 'cas', label: 'CAS', iconUrl: 'https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/cas-v2-logo.png' },
  { type: 'oauth2', label: 'Oauth2', iconUrl: 'https://toa-web-test-1258344699.cos.ap-guangzhou.myqcloud.com/public/images/oauth2-v2-logo.png' },
];

/** 登录页 Logo + 标题头部 */
function DialogHeaderSection({ siteName }: { siteName: string }) {
  return (
    <DialogHeader className="items-center mb-6">
      <div
        className="w-14 h-14 rounded-2xl flex items-center justify-center shadow-lg mb-4"
        style={{ background: 'linear-gradient(135deg, #007AFF, #5856D6)' }}
      >
        <span className="text-2xl">🦞</span>
      </div>
      <DialogTitle className="text-xl font-bold text-center">
        登录 {siteName}
      </DialogTitle>
      <DialogDescription className="sr-only">
        登录对话框
      </DialogDescription>
    </DialogHeader>
  );
}

/** 底部品牌标识 */
function BrandFooter() {
  return (
    <div className="flex items-center justify-center mt-6">
      <img
        src="/images/eid-new-brand-gray.png"
        alt="eID Digital Identity"
        className="h-4 object-contain"
        style={{ opacity: 0.55 }}
      />
    </div>
  );
}

/** 手机号图标 */
const PhoneIcon = (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="oklch(0.546 0.245 262.881)" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
    <path d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z" />
  </svg>
);

/** SSO 云钥匙图标 */
const SsoCloudIcon = (
  <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 16 16" fill="none">
    <path d="M12.7492 6.45039C14.5439 6.45061 15.9992 7.9056 15.9992 9.70039C15.999 11.495 14.5438 12.9502 12.7492 12.9504C11.9485 12.9504 11.216 12.659 10.6496 12.1789L10.3918 12.3391L9.3703 12.967L9.88788 13.8068C10.1045 14.1594 9.99498 14.6219 9.64276 14.8391C9.29029 15.0559 8.82782 14.946 8.61053 14.5939L8.09296 13.7531L7.29315 14.2463C6.9405 14.4629 6.47796 14.3527 6.26093 14.0002C6.04425 13.6478 6.15412 13.1862 6.50604 12.9689L9.60565 11.0617L9.75507 10.9689C9.5897 10.5793 9.49926 10.1504 9.49921 9.70039C9.49921 7.90552 10.9544 6.45048 12.7492 6.45039ZM8.84882 1.05C11.2202 1.05007 13.27 2.41319 14.4367 4.40742C14.6454 4.76482 14.5244 5.22374 14.1672 5.43282C13.8097 5.64165 13.3509 5.5216 13.1418 5.16426C12.2054 3.56365 10.6148 2.55007 8.84882 2.55C6.79192 2.55009 4.96492 3.93122 4.14764 6.00606C4.13899 6.02803 4.13067 6.05034 4.12225 6.07246C4.01089 6.36496 3.73619 6.54595 3.44159 6.55391L3.25018 6.61153C2.20974 6.99698 1.50018 7.95313 1.50018 9.04219C1.50031 10.4535 2.70191 11.6496 4.25018 11.6496H4.34686C4.76086 11.6497 5.09665 11.9856 5.09686 12.3996C5.09686 12.8138 4.76099 13.1495 4.34686 13.1496H4.25018C1.93262 13.1496 0.000306483 11.3399 0.000183105 9.04219C0.000183105 7.26469 1.16224 5.77297 2.75604 5.19453L2.79901 5.18086L2.87811 5.15645C3.94294 2.75887 6.18963 1.05008 8.84882 1.05ZM12.7492 7.95039C11.7828 7.95048 10.9992 8.73395 10.9992 9.70039C10.9994 10.6667 11.7829 11.4503 12.7492 11.4504C13.7154 11.4502 14.499 10.6666 14.4992 9.70039C14.4992 8.73403 13.7155 7.95061 12.7492 7.95039Z" fill="oklch(0.446 0.03 256.802)" />
  </svg>
);

/** 邮箱图标 */
const EmailIcon = (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="oklch(0.546 0.245 262.881)" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
    <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z" />
    <polyline points="22,6 12,13 2,6" />
  </svg>
);

/** 账号密码图标 - 用户头像轮廓 */
const AccountIcon = (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="oklch(0.546 0.245 262.881)" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
    <circle cx="12" cy="7" r="4" />
  </svg>
);

/** 登录操作按钮配置 */
interface LoginAction {
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
}

/** 其他登录方式 - 分隔线 + 按钮列表 + 品牌标识 */
function OtherLoginMethods({
  title = '其他登录方式',
  actions,
}: {
  title?: string;
  actions: LoginAction[];
}) {
  return (
    <div className="w-full mt-2">
      {/* 分隔线 */}
      <div className="flex items-center gap-3 mt-5">
        <div className="flex-1 h-px" style={{ background: 'oklch(0.91 0.008 240)' }} />
        <span style={{ fontSize: '12px', color: 'oklch(0.65 0.01 240)' }}>{title}</span>
        <div className="flex-1 h-px" style={{ background: 'oklch(0.91 0.008 240)' }} />
      </div>

      {/* 登录按钮 */}
      <div className="flex justify-center gap-[72px] mt-4">
        {actions.map((action) => (
          <button
            key={action.label}
            type="button"
            className="flex flex-col items-center gap-1.5 group cursor-pointer"
            onClick={action.onClick}
          >
            <div
              className="w-10 h-10 rounded-full flex items-center justify-center transition-all group-hover:shadow-md"
              style={{ background: 'oklch(0.96 0.005 240)' }}
            >
              {action.icon}
            </div>
            <span style={{ fontSize: '12px', color: 'oklch(0.52 0.015 240)' }}>
              {action.label}
            </span>
          </button>
        ))}
      </div>

      <BrandFooter />
    </div>
  );
}

/** SSO IM 卡片列表：最多展示 3 个，超过 3 个出现滚动条 */
function SsoImCardList({ options, onSelect }: { options: SsoImOption[]; onSelect: (type: string) => void }) {
  const needScroll = options.length > 3;

  return (
    <div
      className={cn(
        'flex flex-col gap-3',
        needScroll && 'overflow-y-auto pr-1',
      )}
      // 3 个卡片高度（每个约 68px）+ 2 个间距（每个 12px）= 228px
      style={needScroll ? { maxHeight: 228 } : undefined}
    >
      {options.map((opt) => (
        <button
          key={opt.type}
          onClick={() => onSelect(opt.type)}
          className="flex items-center gap-4 w-full px-4 py-3.5 rounded-xl bg-gray-100 transition-all duration-150 text-left group hover:bg-gray-200 cursor-pointer flex-shrink-0"
        >
          <img src={opt.iconUrl} alt={opt.label} className="w-10 h-10 rounded-lg object-contain flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <span className="text-sm font-medium text-gray-900 transition-colors group-hover:text-blue-600">
              {opt.label}
            </span>
          </div>
          <svg className="w-4 h-4 text-gray-400 transition-colors flex-shrink-0 group-hover:text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
          </svg>
        </button>
      ))}
    </div>
  );
}

/** 模拟手机号登录表单 */
function PhoneLoginForm() {
  const [phone, setPhone] = useState('');
  const [code, setCode] = useState('');
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [agreed, setAgreed] = useState(false);

  const handleSendCode = useCallback(() => {
    if (!phone.trim()) {
      toast.error('请输入手机号');
      return;
    }
    toast.success('验证码已发送（Demo 演示）');
    setCodeSent(true);
    setCountdown(60);
    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          clearInterval(timer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  }, [phone]);

  const handleLogin = useCallback(() => {
    if (!phone.trim() || !code.trim()) {
      toast.error('请填写完整信息');
      return;
    }
    if (!agreed) {
      toast.error('请先阅读并同意服务协议');
      return;
    }
    toast.success('登录成功（Demo 演示）');
  }, [phone, code, agreed]);

  return (
    <div className="flex flex-col gap-5 w-full">
      {/* 手机号输入 */}
      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium" style={{ color: '#1d2129' }}>手机号</label>
        <div
          className="flex items-center rounded-lg border border-input transition-[color,box-shadow] overflow-hidden focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50"
        >
          {/* +86 区号选择 */}
          <div
            className="flex items-center gap-1 px-3 py-3 flex-shrink-0 cursor-pointer select-none"
            style={{ borderRight: '1px solid #e5e6eb' }}
            onClick={() => toast.info('Demo: 选择区号')}
          >
            <span className="text-sm" style={{ color: '#1d2129' }}>+86</span>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#86909c" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <polyline points="6 9 12 15 18 9" />
            </svg>
          </div>
          <input
            type="tel"
            placeholder="请输入手机号"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            className="flex-1 px-3 py-3 text-sm outline-none bg-transparent"
            style={{ color: '#1d2129' }}
          />
        </div>
      </div>

      {/* 验证码输入 */}
      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium" style={{ color: '#1d2129' }}>验证码</label>
        <div
          className="flex items-center rounded-lg border border-input transition-[color,box-shadow] overflow-hidden focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50"
        >
          <input
            type="text"
            placeholder="请输入验证码"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            className="flex-1 px-3 py-3 text-sm outline-none bg-transparent"
            style={{ color: '#1d2129' }}
          />
          <button
            type="button"
            disabled={countdown > 0}
            onClick={handleSendCode}
            className="flex-shrink-0 px-4 py-3 text-sm font-medium whitespace-nowrap transition-all disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
            style={{
              borderLeft: '1px solid #e5e6eb',
              color: countdown > 0 ? '#c9cdd4' : '#165dff',
              background: 'transparent',
            }}
          >
            {countdown > 0 ? `${countdown}s` : codeSent ? '重新发送' : '获取验证码'}
          </button>
        </div>
      </div>

      {/* 登录按钮 */}
      <button
        type="button"
        onClick={handleLogin}
        className={PRIMARY_BUTTON_CLASS}
        style={PRIMARY_BUTTON_STYLE}
      >
        登录
      </button>

      {/* 协议勾选 */}
      <div className="flex items-start gap-2 justify-center">
        <button
          type="button"
          onClick={() => setAgreed(!agreed)}
          className="mt-0.5 flex-shrink-0 w-4 h-4 rounded border flex items-center justify-center transition-all cursor-pointer"
          style={{
            borderColor: agreed ? '#165dff' : '#c9cdd4',
            background: agreed ? '#165dff' : 'transparent',
          }}
        >
          {agreed && (
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={3} strokeLinecap="round" strokeLinejoin="round">
              <polyline points="20 6 9 17 4 12" />
            </svg>
          )}
        </button>
        <p className="text-xs" style={{ color: '#86909c' }}>
          我已阅读并同意
          <button type="button" className="hover:underline mx-0.5" style={{ color: '#165dff' }} onClick={() => toast.info('Demo: 查看服务协议')}>
            服务协议
          </button>
          和
          <button type="button" className="hover:underline mx-0.5" style={{ color: '#165dff' }} onClick={() => toast.info('Demo: 查看隐私政策')}>
            隐私政策
          </button>
        </p>
      </div>
    </div>
  );
}

/** 视图类型：sso=SSO选择页, phone=手机号登录页, account=账号密码登录页, email=邮箱登录页, reset-password=首次登录/密码过期重置密码页 */
type ViewMode = 'sso' | 'phone' | 'account' | 'email' | 'reset-password';

/**
 * Demo 用：模拟后台返回的登录状态
 * - 'normal'           : 正常登录，无需重置密码
 * - 'first-login'      : 首次登录，需重置密码
 * - 'password-expired' : 密码已过期，需重置密码
 *
 * 后续接真接口时，把这个常量改成根据 login API 返回字段动态计算即可。
 */
type MockLoginStatus = 'normal' | 'first-login' | 'password-expired';
const MOCK_LOGIN_STATUS: MockLoginStatus = 'first-login';

/** 重置密码视图顶部提示语：根据登录状态切换 */
const RESET_PASSWORD_HINT_MAP: Record<Exclude<MockLoginStatus, 'normal'>, string> = {
  'first-login': '检测到您是首次登录，请先重置密码后再继续',
  'password-expired': '检测到您的密码已过期，请先重置密码后再继续',
};

/** Demo 用：模拟当前账号最近 3 个使用过的密码，新密码不允许命中其中之一 */
const RECENT_PASSWORDS_MOCK = ['Abc@1234', 'Test@5678', 'Hello@2024'];

/** 主提交按钮统一样式（按截图：水平蓝紫渐变 + 圆角 8px + 加宽字间距 + 白字加粗） */
const PRIMARY_BUTTON_CLASS =
  'w-full py-3 rounded-lg text-white text-sm font-semibold transition-all hover:brightness-110 active:scale-[0.99] mt-1';
const PRIMARY_BUTTON_STYLE: React.CSSProperties = {
  background: 'linear-gradient(90deg, #4080ff 0%, #5b6cf5 100%)',
  letterSpacing: '0.5em',
  paddingLeft: '0.5em', // 抵消 letter-spacing 末尾让文字看起来不居中的偏移
};

/** 简单邮箱格式校验 */
const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/** 模拟邮箱登录表单 */
function EmailLoginForm() {
  const [email, setEmail] = useState('');
  const [code, setCode] = useState('');
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [agreed, setAgreed] = useState(false);

  const handleSendCode = useCallback(() => {
    if (!email.trim()) {
      toast.error('请输入邮箱');
      return;
    }
    if (!EMAIL_REGEX.test(email.trim())) {
      toast.error('请输入正确的邮箱');
      return;
    }
    toast.success('验证码已发送（Demo 演示）');
    setCodeSent(true);
    setCountdown(60);
    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          clearInterval(timer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  }, [email]);

  const handleLogin = useCallback(() => {
    if (!email.trim() || !code.trim()) {
      toast.error('请填写完整信息');
      return;
    }
    if (!EMAIL_REGEX.test(email.trim())) {
      toast.error('请输入正确的邮箱');
      return;
    }
    if (!agreed) {
      toast.error('请先阅读并同意服务协议');
      return;
    }
    toast.success('登录成功（Demo 演示）');
  }, [email, code, agreed]);

  return (
    <div className="flex flex-col gap-5 w-full">
      {/* 邮箱输入 */}
      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium" style={{ color: '#1d2129' }}>邮箱</label>
        <div
          className="flex items-center rounded-lg border border-input transition-[color,box-shadow] overflow-hidden focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50"
        >
          <input
            type="email"
            placeholder="请输入邮箱"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="flex-1 px-3 py-3 text-sm outline-none bg-transparent"
            style={{ color: '#1d2129' }}
          />
        </div>
      </div>

      {/* 验证码输入 */}
      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium" style={{ color: '#1d2129' }}>验证码</label>
        <div
          className="flex items-center rounded-lg border border-input transition-[color,box-shadow] overflow-hidden focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50"
        >
          <input
            type="text"
            placeholder="请输入验证码"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            className="flex-1 px-3 py-3 text-sm outline-none bg-transparent"
            style={{ color: '#1d2129' }}
          />
          <button
            type="button"
            disabled={countdown > 0}
            onClick={handleSendCode}
            className="flex-shrink-0 px-4 py-3 text-sm font-medium whitespace-nowrap transition-all disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
            style={{
              borderLeft: '1px solid #e5e6eb',
              color: countdown > 0 ? '#c9cdd4' : '#165dff',
              background: 'transparent',
            }}
          >
            {countdown > 0 ? `${countdown}s` : codeSent ? '重新发送' : '获取验证码'}
          </button>
        </div>
      </div>

      {/* 登录按钮 */}
      <button
        type="button"
        onClick={handleLogin}
        className={PRIMARY_BUTTON_CLASS}
        style={PRIMARY_BUTTON_STYLE}
      >
        登录
      </button>

      {/* 协议勾选 */}
      <div className="flex items-start gap-2 justify-center">
        <button
          type="button"
          onClick={() => setAgreed(!agreed)}
          className="mt-0.5 flex-shrink-0 w-4 h-4 rounded border flex items-center justify-center transition-all cursor-pointer"
          style={{
            borderColor: agreed ? '#165dff' : '#c9cdd4',
            background: agreed ? '#165dff' : 'transparent',
          }}
        >
          {agreed && (
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={3} strokeLinecap="round" strokeLinejoin="round">
              <polyline points="20 6 9 17 4 12" />
            </svg>
          )}
        </button>
        <p className="text-xs" style={{ color: '#86909c' }}>
          我已阅读并同意
          <button type="button" className="hover:underline mx-0.5" style={{ color: '#165dff' }} onClick={() => toast.info('Demo: 查看服务协议')}>
            服务协议
          </button>
          和
          <button type="button" className="hover:underline mx-0.5" style={{ color: '#165dff' }} onClick={() => toast.info('Demo: 查看隐私政策')}>
            隐私政策
          </button>
        </p>
      </div>
    </div>
  );
}

/** 模拟账号密码登录表单 */
function AccountLoginForm({ onRequireResetPassword }: { onRequireResetPassword: () => void }) {
  const [account, setAccount] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [agreed, setAgreed] = useState(false);

  const handleLogin = useCallback(() => {
    if (!account.trim() || !password.trim()) {
      toast.error('请填写完整信息');
      return;
    }
    if (!agreed) {
      toast.error('请先阅读并同意服务协议');
      return;
    }
    // Demo: 后端返回首次登录或密码过期 -> 进入重置密码视图
    if (MOCK_LOGIN_STATUS !== 'normal') {
      toast.info(
        MOCK_LOGIN_STATUS === 'first-login'
          ? '首次登录，请先重置密码'
          : '密码已过期，请先重置密码',
      );
      onRequireResetPassword();
      return;
    }
    toast.success('登录成功（Demo 演示）');
  }, [account, password, agreed, onRequireResetPassword]);

  return (
    <div className="flex flex-col gap-5 w-full">
      {/* 用户名输入 */}
      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium" style={{ color: '#1d2129' }}>用户名</label>
        <div
          className="flex items-center rounded-lg border border-input transition-[color,box-shadow] overflow-hidden focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50"
        >
          <input
            type="text"
            placeholder="请输入用户名"
            value={account}
            onChange={(e) => setAccount(e.target.value)}
            className="flex-1 px-3 py-3 text-sm outline-none bg-transparent"
            style={{ color: '#1d2129' }}
          />
        </div>
      </div>

      {/* 密码输入 */}
      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium" style={{ color: '#1d2129' }}>密码</label>
        <div
          className="flex items-center rounded-lg border border-input transition-[color,box-shadow] overflow-hidden focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50"
        >
          <input
            type={showPassword ? 'text' : 'password'}
            placeholder="请输入密码"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="flex-1 px-3 py-3 text-sm outline-none bg-transparent"
            style={{ color: '#1d2129' }}
          />
          <button
            type="button"
            onClick={() => setShowPassword((v) => !v)}
            className="flex-shrink-0 px-3 py-3 cursor-pointer flex items-center justify-center"
            aria-label={showPassword ? '隐藏密码' : '显示密码'}
            style={{ background: 'transparent' }}
          >
            {showPassword ? (
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#86909c" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                <circle cx="12" cy="12" r="3" />
              </svg>
            ) : (
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#86909c" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
                <path d="M17.94 17.94A10.94 10.94 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A10.94 10.94 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                <line x1="1" y1="1" x2="23" y2="23" />
              </svg>
            )}
          </button>
        </div>
      </div>

      {/* 忘记密码 */}
      <div className="flex justify-end -mt-3">
        <a
          href="/forgot-password"
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs hover:underline"
          style={{ color: '#165dff' }}
        >
          忘记密码？
        </a>
      </div>

      {/* 登录按钮 */}
      <button
        type="button"
        onClick={handleLogin}
        className={PRIMARY_BUTTON_CLASS}
        style={PRIMARY_BUTTON_STYLE}
      >
        登录
      </button>

      {/* 协议勾选 */}
      <div className="flex items-start gap-2 justify-center">
        <button
          type="button"
          onClick={() => setAgreed(!agreed)}
          className="mt-0.5 flex-shrink-0 w-4 h-4 rounded border flex items-center justify-center transition-all cursor-pointer"
          style={{
            borderColor: agreed ? '#165dff' : '#c9cdd4',
            background: agreed ? '#165dff' : 'transparent',
          }}
        >
          {agreed && (
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={3} strokeLinecap="round" strokeLinejoin="round">
              <polyline points="20 6 9 17 4 12" />
            </svg>
          )}
        </button>
        <p className="text-xs" style={{ color: '#86909c' }}>
          我已阅读并同意
          <button type="button" className="hover:underline mx-0.5" style={{ color: '#165dff' }} onClick={() => toast.info('Demo: 查看服务协议')}>
            服务协议
          </button>
          和
          <button type="button" className="hover:underline mx-0.5" style={{ color: '#165dff' }} onClick={() => toast.info('Demo: 查看隐私政策')}>
            隐私政策
          </button>
        </p>
      </div>
    </div>
  );
}

/** View 1: SSO 选择页面 */
function SsoMainView({
  options,
  onPhoneLogin,
  onAccountLogin,
  onEmailLogin,
}: {
  options: SsoImOption[];
  onPhoneLogin: () => void;
  onAccountLogin: () => void;
  onEmailLogin: () => void;
}) {
  const handleSsoSelect = useCallback((type: string) => {
    const id = Date.now();
    toast.success(() => <>{`已选择 ${type} 登录（Demo 演示）`}{withClose(id)}</>, { id });
  }, []);

  return (
    <div className="flex flex-col">
      <SsoImCardList options={options} onSelect={handleSsoSelect} />

      <OtherLoginMethods
        title="其他账号登录"
        actions={[
          { icon: PhoneIcon, label: '手机号', onClick: onPhoneLogin },
          { icon: AccountIcon, label: '账号密码', onClick: onAccountLogin },
          { icon: EmailIcon, label: '邮箱', onClick: onEmailLogin },
        ]}
      />
    </div>
  );
}

/** View 2: 手机号登录页面 */
function PhoneLoginView({
  showSsoOption,
  onSsoClick,
  onAccountClick,
  onEmailClick,
}: {
  showSsoOption: boolean;
  onSsoClick?: () => void;
  onAccountClick?: () => void;
  onEmailClick?: () => void;
}) {
  const ssoAction: LoginAction = {
    icon: SsoCloudIcon,
    label: 'SSO',
    onClick: onSsoClick ?? (() => toast.info('Demo: SSO 登录')),
  };
  const accountAction: LoginAction = {
    icon: AccountIcon,
    label: '账号密码',
    onClick: onAccountClick ?? (() => toast.info('Demo: 账号密码登录')),
  };
  const emailAction: LoginAction = {
    icon: EmailIcon,
    label: '邮箱',
    onClick: onEmailClick ?? (() => toast.info('Demo: 邮箱登录')),
  };

  return (
    <div className="flex flex-col items-center">
      <PhoneLoginForm />

      <OtherLoginMethods
        actions={showSsoOption ? [ssoAction, accountAction, emailAction] : [accountAction, emailAction]}
      />
    </div>
  );
}

/** View 3: 账号密码登录页面 */
function AccountLoginView({
  onSsoClick,
  onPhoneClick,
  onEmailClick,
  onRequireResetPassword,
}: {
  onSsoClick: () => void;
  onPhoneClick: () => void;
  onEmailClick: () => void;
  onRequireResetPassword: () => void;
}) {
  return (
    <div className="flex flex-col items-center">
      <AccountLoginForm onRequireResetPassword={onRequireResetPassword} />

      <OtherLoginMethods
        actions={[
          { icon: SsoCloudIcon, label: 'SSO', onClick: onSsoClick },
          { icon: PhoneIcon, label: '手机号', onClick: onPhoneClick },
          { icon: EmailIcon, label: '邮箱', onClick: onEmailClick },
        ]}
      />
    </div>
  );
}

/** View 4: 邮箱登录页面 */
function EmailLoginView({
  onSsoClick,
  onPhoneClick,
  onAccountClick,
}: {
  onSsoClick: () => void;
  onPhoneClick: () => void;
  onAccountClick: () => void;
}) {
  return (
    <div className="flex flex-col items-center">
      <EmailLoginForm />

      <OtherLoginMethods
        actions={[
          { icon: SsoCloudIcon, label: 'SSO', onClick: onSsoClick },
          { icon: PhoneIcon, label: '手机号', onClick: onPhoneClick },
          { icon: AccountIcon, label: '账号密码', onClick: onAccountClick },
        ]}
      />
    </div>
  );
}

/** View 5: 重置密码视图（首次登录或密码过期） */
function ResetPasswordView({ onDone }: { onDone: () => void }) {
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showNewPassword, setShowNewPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

  /** 行内错误：onBlur 时校验填入；用户继续输入时清掉 */
  const [newPasswordError, setNewPasswordError] = useState<string | null>(null);
  const [confirmPasswordError, setConfirmPasswordError] = useState<string | null>(null);

  /** 顶部提示语：根据后台返回的登录状态切换 */
  const resetHint =
    MOCK_LOGIN_STATUS === 'password-expired'
      ? RESET_PASSWORD_HINT_MAP['password-expired']
      : RESET_PASSWORD_HINT_MAP['first-login'];

  const handleNewPasswordBlur = useCallback(() => {
    if (!newPassword) {
      setNewPasswordError(null);
      return;
    }
    setNewPasswordError(validatePasswordStrength(newPassword));
  }, [newPassword]);

  const handleConfirmPasswordBlur = useCallback(() => {
    if (!confirmPassword) {
      setConfirmPasswordError(null);
      return;
    }
    if (newPassword && confirmPassword !== newPassword) {
      setConfirmPasswordError('两次输入的密码需保持一致');
    } else {
      setConfirmPasswordError(null);
    }
  }, [confirmPassword, newPassword]);

  const handleSubmit = useCallback(() => {
    // 提交时再做一次最终校验：缺项/强度/一致性 → 行内红字阻止提交
    if (!newPassword.trim() || !confirmPassword.trim()) {
      if (!newPassword.trim()) setNewPasswordError('请输入新密码');
      if (!confirmPassword.trim()) setConfirmPasswordError('请再次输入新密码');
      return;
    }
    const strengthError = validatePasswordStrength(newPassword);
    if (strengthError) {
      setNewPasswordError(strengthError);
      return;
    }
    if (newPassword !== confirmPassword) {
      setConfirmPasswordError('两次输入的密码需保持一致');
      return;
    }
    if (RECENT_PASSWORDS_MOCK.includes(newPassword)) {
      // 旧密码命中保留 toast 提示
      toast.error('为了您的账号安全，请不要使用最近 3 个使用过的密码');
      return;
    }
    toast.success('密码重置成功，请使用新密码重新登录');
    onDone();
  }, [newPassword, confirmPassword, onDone]);

  /** 输入框包裹层 className：根据是否有错切换 border 红色 */
  const newPwdWrapperClass = `flex items-center rounded-lg border overflow-hidden transition-[color,box-shadow] ${
    newPasswordError
      ? 'border-[#f53f3f] focus-within:border-[#f53f3f] focus-within:ring-[3px] focus-within:ring-[#f53f3f]/30'
      : 'border-input focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50'
  }`;
  const confirmPwdWrapperClass = `flex items-center rounded-lg border overflow-hidden transition-[color,box-shadow] ${
    confirmPasswordError
      ? 'border-[#f53f3f] focus-within:border-[#f53f3f] focus-within:ring-[3px] focus-within:ring-[#f53f3f]/30'
      : 'border-input focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50'
  }`;

  return (
    <div className="flex flex-col items-center w-full">
      <div className="flex flex-col gap-5 w-full">
        {/* 提示语：根据后端返回的登录状态切换 */}
        <div
          className="rounded-md px-3 py-2 text-xs text-center"
          style={{ background: '#e8f3ff', color: '#165dff' }}
        >
          {resetHint}
        </div>

        {/* 新密码 */}
        <div className="flex flex-col gap-2">
          <label
            className="text-sm font-medium flex items-center gap-1"
            style={{ color: '#1d2129' }}
          >
            <span>新密码</span>
            {/* ⓘ 规范说明 tooltip：使用项目统一的 Tooltip 组件，Portal 到 body，自动避免边界裁切 */}
            <Tooltip>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  aria-label="查看密码规范"
                  className="inline-flex items-center justify-center cursor-help rounded-full focus:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
                  style={{ background: 'transparent' }}
                >
                  <svg
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="#86909c"
                    strokeWidth={1.8}
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    className="transition-colors hover:stroke-[#165dff]"
                  >
                    <circle cx="12" cy="12" r="10" />
                    <line x1="12" y1="8" x2="12" y2="13" />
                    <line x1="12" y1="16" x2="12.01" y2="16" />
                  </svg>
                </button>
              </TooltipTrigger>
              <TooltipContent
                side="top"
                align="center"
                sideOffset={6}
                className="max-w-[240px] whitespace-normal text-xs leading-relaxed"
              >
                {PASSWORD_RULES_HINT}
              </TooltipContent>
            </Tooltip>
          </label>
          <div className={newPwdWrapperClass}>
            <input
              type={showNewPassword ? 'text' : 'password'}
              placeholder="请输入新密码"
              value={newPassword}
              onChange={(e) => {
                setNewPassword(e.target.value);
                if (newPasswordError) setNewPasswordError(null);
              }}
              onBlur={handleNewPasswordBlur}
              className="flex-1 px-3 py-3 text-sm outline-none bg-transparent"
              style={{ color: '#1d2129' }}
              maxLength={16}
            />
            <button
              type="button"
              onClick={() => setShowNewPassword((v) => !v)}
              className="flex-shrink-0 px-3 py-3 cursor-pointer flex items-center justify-center"
              aria-label={showNewPassword ? '隐藏密码' : '显示密码'}
              style={{ background: 'transparent' }}
            >
              {showNewPassword ? (
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#86909c" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
                  <path d="M17.94 17.94A10.94 10.94 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A10.94 10.94 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                  <line x1="1" y1="1" x2="23" y2="23" />
                </svg>
              ) : (
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#86909c" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
                  <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                  <circle cx="12" cy="12" r="3" />
                </svg>
              )}
            </button>
          </div>
          {/* 行内错误：onBlur 后只显示首条不满足的规则 */}
          {newPasswordError && (
            <div className="text-xs leading-tight" style={{ color: '#f53f3f' }}>
              {newPasswordError}
            </div>
          )}
        </div>

        {/* 确认新密码 */}
        <div className="flex flex-col gap-2">
          <label className="text-sm font-medium" style={{ color: '#1d2129' }}>确认新密码</label>
          <div className={confirmPwdWrapperClass}>
            <input
              type={showConfirmPassword ? 'text' : 'password'}
              placeholder="请再次输入新密码"
              value={confirmPassword}
              onChange={(e) => {
                setConfirmPassword(e.target.value);
                if (confirmPasswordError) setConfirmPasswordError(null);
              }}
              onBlur={handleConfirmPasswordBlur}
              className="flex-1 px-3 py-3 text-sm outline-none bg-transparent"
              style={{ color: '#1d2129' }}
              maxLength={16}
            />
            <button
              type="button"
              onClick={() => setShowConfirmPassword((v) => !v)}
              className="flex-shrink-0 px-3 py-3 cursor-pointer flex items-center justify-center"
              aria-label={showConfirmPassword ? '隐藏密码' : '显示密码'}
              style={{ background: 'transparent' }}
            >
              {showConfirmPassword ? (
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#86909c" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
                  <path d="M17.94 17.94A10.94 10.94 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A10.94 10.94 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                  <line x1="1" y1="1" x2="23" y2="23" />
                </svg>
              ) : (
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#86909c" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
                  <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                  <circle cx="12" cy="12" r="3" />
                </svg>
              )}
            </button>
          </div>
          {/* 行内错误：两次密码不一致提示 */}
          {confirmPasswordError && (
            <div className="text-xs leading-tight" style={{ color: '#f53f3f' }}>
              {confirmPasswordError}
            </div>
          )}
        </div>

        {/* 提交按钮 */}
        <button
          type="button"
          onClick={handleSubmit}
          className={PRIMARY_BUTTON_CLASS}
          style={PRIMARY_BUTTON_STYLE}
        >
          确认重置
        </button>
      </div>
    </div>
  );
}

/**
 * SsoLoginDialog - SSO 登录弹窗交互样式 Demo
 *
 * 功能：
 * - 双视图切换（SSO 选择 / 手机号登录）
 * - SSO IM 卡片列表（超过 2 项自动滚动）
 * - 其他登录方式快捷入口
 * - 手机号 + 验证码表单
 * - 纯交互演示，无真实业务逻辑
 */
const SsoLoginDialog: React.FC<SsoLoginDialogProps> = ({ visible, onClose }) => {
  const [view, setView] = useState<ViewMode>('sso');

  // 关闭弹窗时重置视图
  const handleClose = useCallback(() => {
    setView('sso');
    onClose();
  }, [onClose]);

  // 弹窗打开时锁定背景滚动
  React.useEffect(() => {
    if (visible) {
      const originalOverflow = document.body.style.overflow;
      document.body.style.overflow = 'hidden';
      return () => {
        document.body.style.overflow = originalOverflow;
      };
    }
  }, [visible]);

  return (
    <Dialog open={visible} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent
        className="sm:max-w-[420px] p-0 gap-0 overflow-hidden bg-white"
        onInteractOutside={(e) => e.preventDefault()}
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        {/* 返回按钮 - 在非 SSO 主视图时显示 */}
        {view !== 'sso' && (
          <button
            type="button"
            className="absolute top-4 left-4 rounded-xs opacity-70 transition-opacity hover:opacity-100 focus:outline-none z-10"
            onClick={() => setView(view === 'reset-password' ? 'account' : 'sso')}
          >
            <ArrowLeft className="size-4" />
          </button>
        )}

        <div className="flex flex-col max-h-[85vh] overflow-hidden">
          <div className="px-8 pt-8 pb-2 flex-shrink-0">
            <DialogHeaderSection siteName="OpenClaw Enterprise" />
          </div>

          <div className="px-8 pb-8 overflow-y-auto flex-1 min-h-0">
            {view === 'sso' && (
              <SsoMainView
                options={SSO_IM_OPTIONS}
                onPhoneLogin={() => setView('phone')}
                onAccountLogin={() => setView('account')}
                onEmailLogin={() => setView('email')}
              />
            )}
            {view === 'phone' && (
              <PhoneLoginView
                showSsoOption
                onSsoClick={() => setView('sso')}
                onAccountClick={() => setView('account')}
                onEmailClick={() => setView('email')}
              />
            )}
            {view === 'account' && (
              <AccountLoginView
                onSsoClick={() => setView('sso')}
                onPhoneClick={() => setView('phone')}
                onEmailClick={() => setView('email')}
                onRequireResetPassword={() => setView('reset-password')}
              />
            )}
            {view === 'email' && (
              <EmailLoginView
                onSsoClick={() => setView('sso')}
                onPhoneClick={() => setView('phone')}
                onAccountClick={() => setView('account')}
              />
            )}
            {view === 'reset-password' && (
              <ResetPasswordView onDone={() => setView('account')} />
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default SsoLoginDialog;
