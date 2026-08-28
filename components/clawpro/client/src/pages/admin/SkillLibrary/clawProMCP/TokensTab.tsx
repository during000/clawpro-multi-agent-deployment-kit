/**
 * 下发管理 Tab
 *
 * 展示已下发本 MCP 的 Agent 列表 + Token 管理状态
 */

import React, { useEffect, useState } from 'react';
import {
  ChevronDown,
  ChevronRight,
} from 'lucide-react';
import { StatusTag } from '@/components/ui/status-tag';
import { BodyMedium, MetaText, HelperText } from '@/components/ui/Typography';
import {
  loadDistributedAgents,
  loadUserTokens,
  subscribeChange,
} from './store';
import type {
  DistributedAgent,
  UserToken,
} from './types';

export default function TokensTab() {
  const [tokens, setTokens] = useState<UserToken[]>([]);
  const [agents, setAgents] = useState<DistributedAgent[]>([]);
  const [expandedUsers, setExpandedUsers] = useState<Set<string>>(new Set());

  useEffect(() => {
    const refresh = () => {
      setTokens(loadUserTokens());
      setAgents(loadDistributedAgents());
    };
    refresh();
    return subscribeChange(refresh);
  }, []);

  const toggleExpand = (userId: string) => {
    setExpandedUsers(prev => {
      const next = new Set(prev);
      if (next.has(userId)) {
        next.delete(userId);
      } else {
        next.add(userId);
      }
      return next;
    });
  };

  // 按用户分组 Agent
  const agentsByUser = React.useMemo(() => {
    const map = new Map<string, DistributedAgent[]>();
    agents.forEach(a => {
      const list = map.get(a.ownerUserId) || [];
      list.push(a);
      map.set(a.ownerUserId, list);
    });
    return map;
  }, [agents]);

  return (
    <div className="space-y-4">
      {/* 概览 */}
      <div>
        <BodyMedium>已下发 Agent</BodyMedium>
        <HelperText className="text-gray-500 mt-1">
          本 MCP 已下发到 {agents.length} 个 Agent 实例，涉及 {tokens.length} 个用户
        </HelperText>
      </div>

      {/* 用户 → Agent 嵌套列表 */}
      <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-gray-100 bg-gray-50/50">
              <th className="px-4 py-2 text-left text-xs font-medium text-gray-500" style={{ width: '20%' }}>用户</th>
              <th className="px-4 py-2 text-left text-xs font-medium text-gray-500" style={{ width: '12%' }}>角色</th>
              <th className="px-4 py-2 text-left text-xs font-medium text-gray-500" style={{ width: '15%' }}>Token 掩码</th>
              <th className="px-4 py-2 text-left text-xs font-medium text-gray-500" style={{ width: '10%' }}>Token 状态</th>
              <th className="px-4 py-2 text-center text-xs font-medium text-gray-500" style={{ width: '8%' }}>已下发 Agent</th>
              <th className="px-4 py-2 text-left text-xs font-medium text-gray-500" style={{ width: '15%' }}>最近调用</th>
            </tr>
          </thead>
          <tbody>
            {tokens.map(t => {
              const userAgents = agentsByUser.get(t.userId) || [];
              const isExpanded = expandedUsers.has(t.userId);
              return (
                <React.Fragment key={t.userId}>
                  <tr
                    className="border-b border-gray-50 hover:bg-gray-50/30 cursor-pointer"
                    onClick={() => userAgents.length > 0 && toggleExpand(t.userId)}
                  >
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        {userAgents.length > 0 && (
                          isExpanded ? <ChevronDown className="w-4 h-4 text-gray-400" /> : <ChevronRight className="w-4 h-4 text-gray-400" />
                        )}
                        <BodyMedium>{t.userName}</BodyMedium>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <StatusTag variant={t.role === 'admin' ? 'blue' : 'gray'} mode="soft">
                        {t.role === 'admin' ? '管理员' : '用户端'}
                      </StatusTag>
                    </td>
                    <td className="px-4 py-3">
                      <MetaText className="font-mono text-gray-600">{t.tokenMask}</MetaText>
                    </td>
                    <td className="px-4 py-3">
                      <StatusTag variant={t.status === 'active' ? 'green' : 'red'} mode="fill">
                        {t.status === 'active' ? '启用' : '禁用'}
                      </StatusTag>
                    </td>
                    <td className="px-4 py-3 text-center">
                      <span className="text-sm font-medium text-gray-900">{userAgents.length}</span>
                    </td>
                    <td className="px-4 py-3">
                      <MetaText className="text-gray-500">{t.lastUsedAt || '—'}</MetaText>
                    </td>
                  </tr>
                  {isExpanded && userAgents.map(a => (
                    <tr key={a.agentId} className="border-b border-gray-50 bg-gray-50/20">
                      {/* Col 1 · 用户列位置：Agent 名称，缩进表示层级 */}
                      <td className="px-4 py-2 pl-12">
                        <MetaText className="text-gray-700">└ {a.agentName}</MetaText>
                      </td>
                      {/* Col 2 · 角色列位置：Agent ID */}
                      <td className="px-4 py-2">
                        <MetaText className="text-gray-400">{a.agentId}</MetaText>
                      </td>
                      {/* Col 3 · Token 掩码：注入到本 Agent 的 Token 掩码 */}
                      <td className="px-4 py-2">
                        <MetaText className="font-mono text-gray-400">{a.injectedTokenMask}</MetaText>
                      </td>
                      {/* Col 4 · Token 状态位置：下发来源 tag（Agent 层没有 Token 状态） */}
                      <td className="px-4 py-2">
                        <StatusTag variant="gray" mode="soft">
                          {a.source === 'manual' ? '手动下发' : '资产同步'}
                        </StatusTag>
                      </td>
                      {/* Col 5 · 已下发 Agent 列位置：Agent 层无此值，占位 */}
                      <td className="px-4 py-2 text-center">
                        <span className="text-gray-300">—</span>
                      </td>
                      {/* Col 6 · 最近调用列位置：下发时间 */}
                      <td className="px-4 py-2">
                        <MetaText className="text-gray-400">{a.distributedAt}</MetaText>
                      </td>
                    </tr>
                  ))}
                </React.Fragment>
              );
            })}
            {tokens.length === 0 && (
              <tr>
                <td colSpan={6} className="text-center text-gray-400 py-10">
                  暂无数据
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <HelperText className="text-gray-400">
        来源说明：手动下发 = 管理员在 Agent 工具库点击"下发"；资产同步 = 通过项目资产管理自动同步到分组下 Agent
      </HelperText>
    </div>
  );
}
