/**
 * 能力配置 Tab — 一页分块布局
 *
 * 53 个工具按 9 个模块分组：管控端 8 组、用户端 1 组。
 * 顶部按角色分段（管控端工具 / 用户端工具），下方各自展示模块折叠列表。
 *
 * 勾选语义：仅控制"平台是否允许该工具被 MCP 曝光/调用"，
 * 不直接影响已下发 Agent 侧配置——Agent 侧最终能力由后台管理。
 *
 * 默认所有模块折叠，用户按需展开。
 */

import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { ChevronDown, ChevronRight, Search } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { StatusTag } from '@/components/ui/status-tag';
import { BodyMedium, MetaText, HelperText } from '@/components/ui/Typography';
import { CAPABILITIES, MODULES } from './mockData';
import type { CapabilityRisk, CapabilityRoleScope, CapabilityToggles } from './types';
import { loadCapabilityToggles, saveCapabilityToggles } from './store';

const RISK_LABEL: Record<CapabilityRisk, { label: string; variant: 'green' | 'orange' | 'red' }> = {
  low: { label: '低', variant: 'green' },
  medium: { label: '中', variant: 'orange' },
  high: { label: '高', variant: 'red' },
};

const SECTION_META: Record<CapabilityRoleScope, { title: string; hint: string; colLabel: string }> = {
  admin: {
    title: '管控端工具',
    hint: '仅管理员可用，走 /admin/* 接口',
    colLabel: '管控端可用',
  },
  member: {
    title: '用户端工具',
    hint: '管理员和用户端角色均可用，走 /openclaw/* 接口',
    colLabel: '用户端可用',
  },
};

/** 第一期已上线的能力 */
const LIVE_CAPABILITIES = CAPABILITIES.filter(c => c.phase === 1);

interface SectionProps {
  role: CapabilityRoleScope;
  searchQuery: string;
  dirtyToggles: CapabilityToggles;
  expandedModules: Set<string>;
  onToggleModule: (moduleId: string) => void;
  onToggleCapability: (capId: string, role: CapabilityRoleScope, value: boolean) => void;
  onModuleToggleAll: (moduleId: string, role: CapabilityRoleScope, value: boolean) => void;
}

function Section({
  role,
  searchQuery,
  dirtyToggles,
  expandedModules,
  onToggleModule,
  onToggleCapability,
  onModuleToggleAll,
}: SectionProps) {
  const meta = SECTION_META[role];
  const roleModules = useMemo(() => MODULES.filter(m => m.roleScope === role), [role]);
  const roleCapabilities = useMemo(
    () => LIVE_CAPABILITIES.filter(c => c.roleScope === role),
    [role],
  );

  const filteredCapabilities = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return roleCapabilities;
    return roleCapabilities.filter(cap =>
      cap.label.toLowerCase().includes(q) ||
      cap.id.toLowerCase().includes(q) ||
      cap.description.toLowerCase().includes(q),
    );
  }, [searchQuery, roleCapabilities]);

  // 搜索时自动展示所有匹配到的模块；未搜索时展示所有模块
  const visibleModules = useMemo(() => {
    if (searchQuery) {
      const matched = new Set(filteredCapabilities.map(c => c.module));
      return roleModules.filter(m => matched.has(m.id));
    }
    return roleModules;
  }, [searchQuery, filteredCapabilities, roleModules]);

  const getModuleStats = (moduleId: string) => {
    const caps = roleCapabilities.filter(c => c.module === moduleId);
    const enabled = caps.filter(c => dirtyToggles[c.id]?.[role]).length;
    return { total: caps.length, enabled };
  };

  if (visibleModules.length === 0) return null;

  const totalCount = roleCapabilities.length;

  return (
    <div className="space-y-3">
      {/* 分段标题 */}
      <div className="flex items-baseline gap-2 pt-2">
        <BodyMedium className="text-gray-900 font-semibold">{meta.title}</BodyMedium>
        <span className="text-xs text-gray-400">（{totalCount}）</span>
        <span className="text-xs text-gray-400">· {meta.hint}</span>
      </div>

      {/* 模块列表 */}
      <div className="space-y-3">
        {visibleModules.map(mod => {
          const isExpanded = !!searchQuery || expandedModules.has(mod.id);
          const stats = getModuleStats(mod.id);
          const moduleCaps = filteredCapabilities.filter(c => c.module === mod.id);
          const allEnabled = stats.enabled === stats.total;
          const noneEnabled = stats.enabled === 0;

          return (
            <div key={mod.id} className="bg-white border border-gray-200 rounded-lg overflow-hidden">
              {/* 模块头部 */}
              <div
                className="flex items-center justify-between px-4 py-3 cursor-pointer hover:bg-gray-50 transition-colors"
                onClick={() => !searchQuery && onToggleModule(mod.id)}
              >
                <div className="flex items-center gap-3">
                  {!searchQuery && (
                    isExpanded
                      ? <ChevronDown className="w-4 h-4 text-gray-400" />
                      : <ChevronRight className="w-4 h-4 text-gray-400" />
                  )}
                  <div>
                    <div className="flex items-center gap-2">
                      <BodyMedium>{mod.label}</BodyMedium>
                      <span className="text-xs text-gray-400">
                        （{stats.enabled}/{stats.total}）
                      </span>
                    </div>
                    <HelperText className="text-gray-500 mt-0.5">{mod.description}</HelperText>
                  </div>
                </div>
                <div className="flex items-center gap-2" onClick={e => e.stopPropagation()}>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-xs h-7"
                    disabled={allEnabled}
                    onClick={() => onModuleToggleAll(mod.id, role, true)}
                  >
                    全部开启
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-xs h-7"
                    disabled={noneEnabled}
                    onClick={() => onModuleToggleAll(mod.id, role, false)}
                  >
                    全部关闭
                  </Button>
                </div>
              </div>

              {/* 工具列表 */}
              {isExpanded && moduleCaps.length > 0 && (
                <div className="border-t border-gray-100">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b border-gray-100 bg-gray-50/50">
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-500" style={{ width: '18%' }}>工具</th>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-500" style={{ width: '30%' }}>说明</th>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-500" style={{ width: '17%' }}>接口</th>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-500" style={{ width: '10%' }}>风险</th>
                        <th className="px-4 py-2 text-center text-xs font-medium text-gray-500" style={{ width: '25%' }}>{meta.colLabel}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {moduleCaps.map(cap => {
                        const t = dirtyToggles[cap.id] ?? { admin: false, member: false };
                        return (
                          <tr key={cap.id} className="border-b border-gray-50 last:border-0 hover:bg-gray-50/30">
                            <td className="px-4 py-3">
                              <MetaText className="font-mono text-gray-900 text-sm">{cap.id}</MetaText>
                            </td>
                            <td className="px-4 py-3">
                              <div className="space-y-0.5">
                                <BodyMedium>{cap.label}</BodyMedium>
                                <HelperText className="text-gray-500">{cap.description}</HelperText>
                              </div>
                            </td>
                            <td className="px-4 py-3">
                              <MetaText className="font-mono text-gray-400 text-xs">{cap.backendApi}</MetaText>
                            </td>
                            <td className="px-4 py-3">
                              <StatusTag variant={RISK_LABEL[cap.risk].variant} mode="fill">
                                {RISK_LABEL[cap.risk].label}
                              </StatusTag>
                            </td>
                            <td className="px-4 py-3 text-center">
                              <Checkbox
                                checked={t[role]}
                                onCheckedChange={v => onToggleCapability(cap.id, role, v === true)}
                              />
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}

              {isExpanded && moduleCaps.length === 0 && (
                <div className="border-t border-gray-100 px-4 py-6 text-center text-gray-400 text-sm">
                  无匹配工具
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default function CapabilitiesTab() {
  const [toggles, setToggles] = useState<CapabilityToggles>(() => loadCapabilityToggles());
  const [dirtyToggles, setDirtyToggles] = useState<CapabilityToggles>(toggles);
  // 默认全部折叠，用户按需展开
  const [expandedModules, setExpandedModules] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    setDirtyToggles(toggles);
  }, [toggles]);

  const isDirty = useMemo(() => {
    return LIVE_CAPABILITIES.some(cap => {
      const a = dirtyToggles[cap.id];
      const b = toggles[cap.id];
      return !b || !a || a.admin !== b.admin || a.member !== b.member;
    });
  }, [dirtyToggles, toggles]);

  const handleToggle = (capId: string, role: CapabilityRoleScope, value: boolean) => {
    setDirtyToggles(prev => ({
      ...prev,
      [capId]: {
        ...(prev[capId] ?? { admin: false, member: false }),
        [role]: value,
      },
    }));
  };

  const handleModuleToggleAll = (moduleId: string, role: CapabilityRoleScope, value: boolean) => {
    const moduleCaps = LIVE_CAPABILITIES.filter(c => c.module === moduleId);
    setDirtyToggles(prev => {
      const next = { ...prev };
      moduleCaps.forEach(cap => {
        next[cap.id] = {
          ...(next[cap.id] ?? { admin: false, member: false }),
          [role]: value,
        };
      });
      return next;
    });
  };

  const handleSave = () => {
    saveCapabilityToggles(dirtyToggles);
    setToggles(dirtyToggles);
    toast.success('能力配置已保存');
  };

  const handleReset = () => {
    setDirtyToggles(toggles);
  };

  const toggleModule = (moduleId: string) => {
    setExpandedModules(prev => {
      const next = new Set(prev);
      if (next.has(moduleId)) {
        next.delete(moduleId);
      } else {
        next.add(moduleId);
      }
      return next;
    });
  };

  return (
    <div className="space-y-4">
      {/* 工具栏 */}
      <div className="flex items-center justify-between gap-4">
        <div className="relative w-72">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input
            placeholder="搜索工具名称、ID 或描述..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={handleReset} disabled={!isDirty}>
            取消修改
          </Button>
          <Button onClick={handleSave} disabled={!isDirty}>
            保存配置
          </Button>
        </div>
      </div>

      {/* 管控端工具 分段 */}
      <Section
        role="admin"
        searchQuery={searchQuery}
        dirtyToggles={dirtyToggles}
        expandedModules={expandedModules}
        onToggleModule={toggleModule}
        onToggleCapability={handleToggle}
        onModuleToggleAll={handleModuleToggleAll}
      />

      {/* 用户端工具 分段 */}
      <Section
        role="member"
        searchQuery={searchQuery}
        dirtyToggles={dirtyToggles}
        expandedModules={expandedModules}
        onToggleModule={toggleModule}
        onToggleCapability={handleToggle}
        onModuleToggleAll={handleModuleToggleAll}
      />
    </div>
  );
}
