/**
 * 概览 Tab
 *
 * 布局：
 *   顶部：4 个统计卡片横排
 *   下方左：服务信息（地址/协议/版本/状态）
 *   下方右：使用须知
 */

import { useEffect, useState } from 'react';
import { Wrench, Send, KeyRound, TrendingUp, AlertCircle } from 'lucide-react';
import { BodyMedium, MetaText, HelperText } from '@/components/ui/Typography';
import { StatusTag } from '@/components/ui/status-tag';
import {
  CLAWPRO_PLATFORM_MCP_NAME,
  CLAWPRO_PLATFORM_MCP_SERVICE_URL,
  CLAWPRO_PLATFORM_MCP_TRANSPORT,
  CLAWPRO_PLATFORM_MCP_VERSION,
} from './constants';
import { CAPABILITIES, MODULES } from './mockData';
import {
  loadCapabilityToggles,
  loadDistributedAgents,
  loadUserTokens,
  subscribeChange,
} from './store';

function StatCard({ icon, label, value, hint }: { icon: React.ReactNode; label: string; value: string | number; hint?: string }) {
  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4">
      <div className="flex items-center gap-2 text-gray-500 mb-2">
        {icon}
        <MetaText>{label}</MetaText>
      </div>
      <div className="text-2xl font-semibold text-gray-900">{value}</div>
      {hint && <HelperText className="mt-1">{hint}</HelperText>}
    </div>
  );
}

export default function OverviewTab() {
  const [tokenCount, setTokenCount] = useState(0);
  const [agentCount, setAgentCount] = useState(0);
  const [enabledToolCount, setEnabledToolCount] = useState(0);
  const [totalToolCount, setTotalToolCount] = useState(0);

  useEffect(() => {
    const refresh = () => {
      setTokenCount(loadUserTokens().filter(t => t.status === 'active').length);
      setAgentCount(loadDistributedAgents().length);
      const toggles = loadCapabilityToggles();
      const liveCaps = CAPABILITIES.filter(c => c.phase === 1);
      setTotalToolCount(liveCaps.length);
      // 已启用 = 该工具在其 roleScope 对应列上被开启
      setEnabledToolCount(
        liveCaps.filter(c => toggles[c.id]?.[c.roleScope]).length,
      );
    };
    refresh();
    return subscribeChange(refresh);
  }, []);

  const callStats = {
    total24h: 156,
    successRate: '94.2%',
    topTool: 'list_skills',
    topToolCount: 48,
  };

  return (
    <div className="space-y-4">
      {/* 顶部：统计卡片横排 */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <StatCard
          icon={<Wrench className="w-3.5 h-3.5" />}
          label="已启用工具"
          value={`${enabledToolCount} / ${totalToolCount}`}
          hint={`共 ${MODULES.length} 个模块`}
        />
        <StatCard
          icon={<Send className="w-3.5 h-3.5" />}
          label="已下发 Agent"
          value={agentCount}
        />
        <StatCard
          icon={<KeyRound className="w-3.5 h-3.5" />}
          label="已颁发 Token"
          value={tokenCount}
        />
        <StatCard
          icon={<TrendingUp className="w-3.5 h-3.5" />}
          label="24h 调用次数"
          value={callStats.total24h}
          hint={`成功率 ${callStats.successRate}`}
        />
      </div>

      {/* 下方：服务信息 + 使用须知 */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* 左栏：服务信息 */}
        <div className="lg:col-span-2">
          <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
            <BodyMedium>服务信息</BodyMedium>
            <MetaText className="text-gray-600 leading-6">
              {CLAWPRO_PLATFORM_MCP_NAME} 是平台内置的 MCP 服务，让 Agent 能在对话中操作 ClawPro：
              管理员管技能/规范/MCP 库、看 Agent 实例/用量/审计、管用户；用户端管自己的实例（查询/开关机/重启等）。
            </MetaText>

            <div className="grid grid-cols-2 gap-4 pt-3 border-t border-gray-100">
              <div>
                <HelperText className="mb-1">服务地址</HelperText>
                <MetaText className="font-mono text-gray-900 break-all">
                  {CLAWPRO_PLATFORM_MCP_SERVICE_URL}
                </MetaText>
              </div>
              <div>
                <HelperText className="mb-1">传输协议</HelperText>
                <MetaText className="text-gray-900">{CLAWPRO_PLATFORM_MCP_TRANSPORT}</MetaText>
              </div>
              <div>
                <HelperText className="mb-1">当前版本</HelperText>
                <MetaText className="text-gray-900">v{CLAWPRO_PLATFORM_MCP_VERSION}</MetaText>
              </div>
              <div>
                <HelperText className="mb-1">健康状态</HelperText>
                <StatusTag variant="green" mode="fill">● 运行中</StatusTag>
              </div>
            </div>
          </div>
        </div>

        {/* 右栏：使用须知 */}
        <div className="bg-amber-50 border border-amber-200 rounded-lg p-4 flex items-start gap-2">
          <AlertCircle className="w-4 h-4 text-amber-700 mt-0.5 flex-shrink-0" />
          <div>
            <MetaText className="text-amber-800 font-medium">使用须知</MetaText>
            <HelperText className="text-amber-700 mt-1 leading-5">
              本 MCP 操作的是 ClawPro 平台本身，请在「能力配置」中谨慎开启高风险工具；
              管控端工具走 <code className="font-mono text-[11px]">/admin/*</code>，用户端工具走 <code className="font-mono text-[11px]">/openclaw/*</code>。
              <strong>管理员权限是超集</strong>：可使用全部 53 个工具；用户端角色仅可使用 12 个用户端工具。
              所有调用都会记录在「调用日志」中以便追溯。
            </HelperText>
          </div>
        </div>
      </div>
    </div>
  );
}
