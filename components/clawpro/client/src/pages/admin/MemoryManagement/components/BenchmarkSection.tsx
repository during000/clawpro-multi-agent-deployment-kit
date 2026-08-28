import React from 'react';
import { FreeVersionCard } from './FreeVersionCard';

interface BenchmarkSectionProps {
  /** 是否有实例开启了 Memory，用于判断默认折叠状态 */
  hasEnabledInstances: boolean;
}

export const BenchmarkSection: React.FC<BenchmarkSectionProps> = () => {
  return (
    <div
      className="bg-white rounded-xl border border-gray-100 overflow-hidden"
      style={{ boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}
    >
      {/* 标题栏 — 常驻展开，不可折叠 */}
      <div className="px-5 py-4 flex items-center gap-2">
        <span className="text-lg">📈</span>
        <h3 className="font-semibold text-gray-900">记忆效果对比</h3>
        <span className="text-xs text-gray-400">基于 PersonaMem 数据集评测</span>
      </div>

      {/* 内容区 — 常驻展示 */}
      <div className="px-5 pb-5">
        <FreeVersionCard isEnabled={false} onEnabledChange={() => {}} />
      </div>
    </div>
  );
};
