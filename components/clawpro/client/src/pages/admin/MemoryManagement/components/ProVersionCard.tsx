import React, { useState } from 'react';
import { Gem, Sparkles, Loader2, AlertCircle, RotateCcw, Search, ShieldCheck, LayoutGrid } from 'lucide-react';
import { ProActivationDialog } from './ProActivationDialog';
import { Button } from '@/components/ui/button';

// Pro 版独有能力
const PRO_FEATURES: { icon: React.ElementType; title: string; color: string; tag?: string }[] = [
  { icon: Search, title: '混合双路检索', color: '#2563EB' },
  { icon: ShieldCheck, title: '企业级安全保障', color: '#14B8A6' },
  { icon: LayoutGrid, title: '全局资源管控', color: '#7C3AED' },
];

// 服务状态类型
type ServiceStatus = 'inactive' | 'activating' | 'active' | 'error';

interface ProVersionCardProps {
  onActivated?: (purchaseSpaces: number) => void;
  /** 当前服务状态 */
  serviceStatus?: ServiceStatus;
  /** 错误信息 */
  errorMessage?: string;
  /** 重试回调 */
  onRetry?: () => void;
}

export const ProVersionCard: React.FC<ProVersionCardProps> = ({ 
  onActivated,
  serviceStatus = 'inactive',
  errorMessage = '',
  onRetry,
}) => {
  const [dialogOpen, setDialogOpen] = useState(false);

  const handleActivationSuccess = (config: { autoEnableForNewInstances: boolean }) => {
    onActivated?.(FIXED_MEMORY_SPACES);
  };
  const FIXED_MEMORY_SPACES = 500;

  const handleOpenClick = () => {
    setDialogOpen(true);
  };

  return (
    <>
      <div className="bg-white h-full flex flex-col">
        <div className="px-6 py-6 flex-1 flex flex-col">
          {/* 头部：图标 + 标题 + 状态标签（跟在标题后面） */}
          <div className="flex items-center gap-3 mb-4">
            <div
              className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0"
              style={{ background: 'linear-gradient(135deg, #7C3AED, #A855F7)' }}
            >
              <Gem className="w-5 h-5 text-white" />
            </div>
            <h2 className="text-lg font-bold text-gray-900">Memory Pro 版</h2>
            
            {/* 状态标签 - 跟在标题后面 */}
            {serviceStatus === 'activating' && (
              <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-700 border border-blue-200">
                <Loader2 className="w-3 h-3 animate-spin" />
                初始化中
              </span>
            )}
            {serviceStatus === 'error' && (
              <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-red-100 text-red-700 border border-red-200">
                <AlertCircle className="w-3 h-3" />
                初始化失败
              </span>
            )}
          </div>
          
          {/* 根据状态显示不同的描述文案 */}
          {serviceStatus === 'activating' ? (
            <div className="mb-4">
              <h3 className="text-base font-bold text-slate-700 mb-1.5">🚀 Memory Pro 正在初始化</h3>
              <p className="text-sm text-gray-500 leading-relaxed">
                正在为您创建向量数据库并配置记忆空间，预计需要 <strong className="text-gray-700">1-2 分钟</strong>，初始化完成后将自动进入管理页面。
              </p>
            </div>
          ) : serviceStatus === 'error' ? (
            <div className="mb-4">
              <h3 className="text-base font-bold text-slate-700 mb-1.5">⚠️ 初始化遇到问题</h3>
              <p className="text-sm text-red-600/80 leading-relaxed">
                {errorMessage || '向量数据库创建失败，请稍后重试。如问题持续，请联系技术支持。'}
              </p>
            </div>
          ) : (
            <>
              <p className="text-sm text-gray-500 mb-4 leading-relaxed">
                基于腾讯云向量数据库的企业级记忆服务，实现语义级记忆检索与企业级数据管理。
              </p>

              {/* Pro 版独有能力 - 特性列表 */}
              <div className="flex-1">
                <div className="space-y-2.5">
                  {PRO_FEATURES.map((f) => {
                    const Icon = f.icon;
                    return (
                      <div key={f.title} className="flex items-center gap-2 text-sm text-gray-600">
                        <div
                          className="w-5 h-5 rounded flex-shrink-0 flex items-center justify-center"
                          style={{ background: `${f.color}15` }}
                        >
                          <Icon className="w-3 h-3" style={{ color: f.color }} />
                        </div>
                        <span>{f.title}</span>
                        {f.tag && (
                          <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-amber-50 text-amber-600 font-medium border border-amber-100">
                            {f.tag}
                          </span>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            </>
          )}
          
          {/* 操作区 - 与 Free 版对齐，放在底部 */}
          <div className="mt-auto pt-4 border-t border-gray-100">
            {serviceStatus === 'activating' ? (
              <div className="flex items-center gap-2 px-4 py-2.5 rounded-lg bg-blue-50 border border-blue-100 w-fit">
                <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
                <span className="text-sm text-blue-700 font-medium">请稍候...</span>
              </div>
            ) : serviceStatus === 'error' ? (
              <Button
                onClick={onRetry}
                variant="outline"
                className="inline-flex items-center gap-2 px-5 py-2.5 rounded-md text-sm font-semibold border-red-200 text-red-600 hover:bg-red-50 hover:border-red-300"
              >
                <RotateCcw className="w-4 h-4" />
                重试
              </Button>
            ) : (
              <div>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-2xl font-bold text-gray-900">免费体验中</span>
                  </div>
                  <button
                    onClick={handleOpenClick}
                    className="inline-flex items-center gap-2 px-6 py-2.5 rounded-lg text-sm font-semibold text-white transition-all hover:shadow-lg hover:shadow-purple-500/25 hover:scale-[1.02] active:scale-[0.98]"
                    style={{ background: 'linear-gradient(135deg, #7C3AED, #A855F7)' }}
                  >
                    <Sparkles className="w-4 h-4" />
                    立即开通
                  </button>
                </div>
                {/* 引导说明 */}
                <p className="text-xs text-gray-400 mt-3 leading-relaxed">
                  开通后，用户可在 Agent 设置页面自行选择启用 Memory Pro 版
                </p>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Pro 版开通对话框 */}
      <ProActivationDialog 
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onConfirm={handleActivationSuccess}
      />
    </>
  );
};
