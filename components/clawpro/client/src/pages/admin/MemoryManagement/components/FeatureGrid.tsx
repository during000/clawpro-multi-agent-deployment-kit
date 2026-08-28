import React from 'react';
import { Search, ShieldCheck, LayoutGrid } from 'lucide-react';

const FEATURES: { icon: React.ElementType; title: string; desc: string; color: string; tag?: string }[] = [
  { icon: Search, title: '混合双路检索', desc: '融合"关键字 + 向量语义"双路召回，精准捕获深层关联，让 Agent 的回答更精准', color: '#2563EB' },
  { icon: ShieldCheck, title: '企业级安全保障', desc: '提供完善的数据备份与强加密机制，匹配企业级数据隐私与合规要求', color: '#14B8A6' },
  { icon: LayoutGrid, title: '全局资源管控', desc: '一站式可视化看板，统一管控所有实例的记忆资源，运维更省心', color: '#7C3AED' },
];

export const FeatureGrid: React.FC = () => {
  return (
    <div>
      <div className="flex items-center gap-2 mb-3">
        <h3 className="text-xs font-semibold text-gray-500">Pro 版独有能力</h3>
        <span className="px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-violet-50 text-violet-600 border border-violet-100">
          升级解锁
        </span>
      </div>
      <div className="grid grid-cols-2 gap-2">
        {FEATURES.map((f) => {
          const Icon = f.icon;
          return (
            <div
              key={f.title}
              className="p-3 rounded-lg border border-gray-100 bg-white/60 hover:border-gray-200 hover:shadow-sm transition-all flex items-start gap-2.5"
            >
              <div
                className="w-8 h-8 rounded-lg flex-shrink-0 flex items-center justify-center"
                style={{ background: `${f.color}10` }}
              >
                <Icon className="w-4 h-4" style={{ color: f.color }} />
              </div>
              <div className="min-w-0">
                <div className="flex items-center gap-1.5 mb-0.5">
                  <h4 className="text-xs font-semibold text-gray-900">{f.title}</h4>
                  {f.tag && (
                    <span className="text-[9px] px-1 py-0.5 rounded-full bg-amber-50 text-amber-600 font-medium border border-amber-100">
                      {f.tag}
                    </span>
                  )}
                </div>
                <p className="text-[11px] text-gray-400 leading-relaxed line-clamp-2">{f.desc}</p>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
