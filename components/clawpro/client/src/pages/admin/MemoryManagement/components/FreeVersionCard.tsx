import React, { useState } from 'react';
import { Switch } from '@/components/ui/switch';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';
import { Zap, Info, AlertTriangle, AlertOctagon } from 'lucide-react';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';

interface FreeVersionCardProps {
  isEnabled: boolean;
  onEnabledChange: (enabled: boolean) => void;
  /** 是否禁用（互斥时禁用） */
  disabled?: boolean;
  /** 禁用时 hover 提示文案 */
  disabledTooltip?: string;
}

export const FreeVersionCard: React.FC<FreeVersionCardProps> = ({
  isEnabled,
  onEnabledChange,
  disabled = false,
  disabledTooltip = '',
}) => {
  const [confirmType, setConfirmType] = useState<'enable' | 'disable' | null>(null);
  const [confirmChecked, setConfirmChecked] = useState(false);

  const handleToggleChange = (checked: boolean) => {
    if (disabled) return;
    setConfirmType(checked ? 'enable' : 'disable');
    setConfirmChecked(false);
  };

  const handleConfirm = () => {
    const type = confirmType!;
    setConfirmType(null);
    setConfirmChecked(false);
    onEnabledChange(type === 'enable');
    toast.success(
      type === 'enable'
        ? '已开启 Memory Free 版，正在为所有实例开启记忆插件'
        : '已关闭 Memory Free 版'
    );
  };

  // Free 版核心特点（带标题 + 描述）
  const freeFeatures = [
    {
      title: '记忆更稳定',
      desc: '自动提取偏好、约束与任务状态，无需手动触发。',
    },
    {
      title: '理解更深刻',
      desc: '四层记忆金字塔逐步提炼，从"记住你说过什么"到"理解你是谁"。',
    },
    {
      title: '检索更精准',
      desc: '记忆分层组织、按场景归类，按需精准召回。',
    },
    {
      title: '跨会话不断',
      desc: '记忆跨聊天通道共享，不随上下文压缩丢失。',
    },
  ];

  return (
    <TooltipProvider>
      <>
        <div
          className={`bg-white rounded-2xl border overflow-hidden transition-all duration-500 ${disabled ? 'opacity-60' : ''}`}
          style={{
            boxShadow: '0 1px 3px rgba(0,0,0,0.06), 0 4px 12px rgba(0,0,0,0.04)',
            borderColor: isEnabled ? 'rgba(0,122,255,0.2)' : 'rgba(229,231,235,1)',
          }}
        >
          <div className="px-8 py-7">
            {/* Header: 图标 + 标题 + 开关 */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div
                  className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0"
                  style={{ background: 'linear-gradient(135deg, #007AFF, #5856D6)' }}
                >
                  <Zap className="w-5 h-5 text-white" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-gray-900">Memory Free 版</h2>
                  <p className="text-xs text-gray-400 mt-0.5">基于实例本地存储，自动提取对话记忆，跨会话精准召回，免费即开即用。</p>
                </div>
              </div>

              {/* 开关区域 */}
              <div className="flex items-center gap-3">
                <div className="flex items-center gap-2">
                  <div
                    className="w-2 h-2 rounded-full transition-colors duration-300"
                    style={{ background: isEnabled ? '#16A34A' : '#d0d0e0' }}
                  />
                  <span className={`text-sm font-medium ${isEnabled ? 'text-gray-900' : 'text-gray-400'}`}>
                    {isEnabled ? '已启用' : '未启用'}
                  </span>
                </div>
                {disabled ? (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="cursor-not-allowed">
                        <Switch checked={isEnabled} disabled />
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="top" className="max-w-xs">
                      <p className="text-sm">{disabledTooltip}</p>
                    </TooltipContent>
                  </Tooltip>
                ) : (
                  <Switch checked={isEnabled} onCheckedChange={handleToggleChange} />
                )}
              </div>
            </div>

            {/* 核心能力介绍 — 两列网格 */}
            <div className="mt-6 grid grid-cols-2 gap-x-8 gap-y-4">
              {freeFeatures.map((feature, i) => (
                <div key={i} className="flex items-start gap-2.5">
                  <div className="w-1.5 h-1.5 rounded-full bg-blue-500 flex-shrink-0 mt-[7px]" />
                  <p className="text-sm text-gray-600 leading-relaxed">
                    <span className="font-semibold text-gray-900">{feature.title}</span>
                    <span className="mx-1.5 text-gray-300">—</span>
                    {feature.desc}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* 确认弹窗 — 开启 */}
        <Dialog open={confirmType === 'enable'} onOpenChange={(o) => { if (!o) setConfirmType(null); }}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>开启 Memory Free 版</DialogTitle>
              <DialogDescription className="sr-only">确认开启记忆功能</DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="flex items-start gap-2.5 bg-blue-50 border border-blue-200 rounded-xl px-4 py-3">
                <Info className="w-4 h-4 text-blue-500 mt-0.5 shrink-0" />
                <div className="text-xs text-blue-800 leading-relaxed space-y-1">
                  <p>• 将为<strong>所有现有实例</strong>自动安装并启用记忆插件。</p>
                  <p>• 之后新创建的实例也将<strong>默认启用</strong>记忆功能。</p>
                  <p>• 用户可在各自实例的设置中自行管理（开启 / 关闭 / 清除数据）。</p>
                </div>
              </div>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={confirmChecked}
                  onChange={(e) => setConfirmChecked(e.target.checked)}
                  className="w-4 h-4 rounded accent-blue-600"
                />
                <span className="text-sm text-gray-600">我已了解上述说明，确认开启</span>
              </label>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setConfirmType(null)}>
                取消
              </Button>
              <Button
                onClick={handleConfirm}
                disabled={!confirmChecked}
                className="text-white disabled:opacity-50"
                style={{ background: 'linear-gradient(135deg, #007AFF, #5856D6)' }}
              >
                确认开启
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* 确认弹窗 — 关闭 */}
        <Dialog open={confirmType === 'disable'} onOpenChange={(o) => { if (!o) setConfirmType(null); }}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>关闭 Memory Free 版</DialogTitle>
              <DialogDescription className="sr-only">确认关闭记忆功能</DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="flex items-start gap-2.5 bg-amber-50 border border-amber-200 rounded-xl px-4 py-3">
                <AlertTriangle className="w-4 h-4 text-amber-500 mt-0.5 shrink-0" />
                <div className="text-xs text-amber-800 leading-relaxed space-y-1">
                  <p>• 新创建的实例将<strong>不再默认启用</strong>记忆功能。</p>
                  <p>• 所有现有实例的记忆插件将被<strong>禁用</strong>（插件保留，但停止工作）。</p>
                  <p>• 已有记忆数据不会删除，重新开启后可恢复。</p>
                </div>
              </div>
              <div className="flex items-start gap-2.5 bg-red-50 border border-red-200 rounded-xl px-4 py-3">
                <AlertOctagon className="w-4 h-4 text-red-500 mt-0.5 shrink-0" />
                <p className="text-xs text-red-800 leading-relaxed">
                  关闭后所有实例将<strong>立即失去记忆能力</strong>，请务必提前通知用户。
                </p>
              </div>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={confirmChecked}
                  onChange={(e) => setConfirmChecked(e.target.checked)}
                  className="w-4 h-4 rounded accent-red-600"
                />
                <span className="text-sm text-gray-600">我已了解上述说明，确认关闭</span>
              </label>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setConfirmType(null)}>
                取消
              </Button>
              <Button
                onClick={handleConfirm}
                disabled={!confirmChecked}
                className="bg-red-600 hover:bg-red-700 text-white disabled:opacity-50"
              >
                确认关闭
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </>
    </TooltipProvider>
  );
};
