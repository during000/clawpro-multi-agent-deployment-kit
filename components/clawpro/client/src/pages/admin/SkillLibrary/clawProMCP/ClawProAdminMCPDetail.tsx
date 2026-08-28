/**
 * ClawPro 平台 MCP 详情页（4-Tab）
 *
 * 与普通 MCPDetail 区别：
 *   - 顶部带"官方/管控"徽标 + 盾形图标
 *   - 没有删除按钮（不允许删除）
 *   - 4 Tab：概览 / 能力配置 / 下发管理 / 调用日志
 */

import { useState } from 'react';
import { ArrowLeft, ShieldCheck, Send } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { StatusTag } from '@/components/ui/status-tag';
import {
  CLAWPRO_PLATFORM_MCP_NAME,
  CLAWPRO_PLATFORM_MCP_TAGLINE,
  CLAWPRO_PLATFORM_MCP_VERSION,
} from './constants';
import OverviewTab from './OverviewTab';
import CapabilitiesTab from './CapabilitiesTab';
import TokensTab from './TokensTab';
import CallLogsTab from './CallLogsTab';

interface Props {
  onBack: () => void;
  /** 上层透传"打开下发对话框"的回调；用于详情页顶部的"下发到 Agent"按钮 */
  onRequestDistribute?: () => void;
}

const TAB_TRIGGER_CLS =
  'rounded-lg px-4 py-1.5 text-sm text-gray-600 bg-white hover:bg-gray-50 border border-transparent data-[state=active]:bg-white data-[state=active]:text-blue-600 data-[state=active]:border-blue-200 transition-colors';

export default function ClawProAdminMCPDetail({ onBack, onRequestDistribute }: Props) {
  const [activeTab, setActiveTab] = useState('overview');

  return (
    <div className="space-y-4">
      {/* 页头 */}
      <div className="bg-white border border-gray-200 rounded-lg p-4">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3 min-w-0">
            <Button variant="ghost" size="sm" onClick={onBack} className="-ml-2">
              <ArrowLeft className="w-4 h-4 mr-1" />
              返回
            </Button>
            <div className="w-10 h-10 rounded-md bg-blue-50 text-blue-600 flex items-center justify-center flex-shrink-0">
              <ShieldCheck className="w-5 h-5" />
            </div>
            <div className="min-w-0 space-y-1">
              <div className="flex items-center gap-2 flex-wrap">
                <h1 className="text-base font-semibold text-gray-900 truncate">
                  {CLAWPRO_PLATFORM_MCP_NAME}
                </h1>
                <StatusTag variant="orange" mode="soft">管控</StatusTag>
                <span className="inline-block px-2 py-0.5 bg-gray-100 text-gray-600 text-xs rounded-full">
                  v{CLAWPRO_PLATFORM_MCP_VERSION}
                </span>
              </div>
              <p className="text-sm text-gray-600">{CLAWPRO_PLATFORM_MCP_TAGLINE}</p>
            </div>
          </div>
          <div className="flex-shrink-0">
            {onRequestDistribute && (
              <Button onClick={onRequestDistribute}>
                <Send className="w-4 h-4 mr-1.5" />
                下发到 Agent
              </Button>
            )}
          </div>
        </div>
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="w-full justify-start bg-white p-0 h-auto gap-2 border-b-0">
          <TabsTrigger value="overview" className={TAB_TRIGGER_CLS}>
            概览
          </TabsTrigger>
          <TabsTrigger value="capabilities" className={TAB_TRIGGER_CLS}>
            能力配置
          </TabsTrigger>
          <TabsTrigger value="tokens" className={TAB_TRIGGER_CLS}>
            下发管理
          </TabsTrigger>
          <TabsTrigger value="logs" className={TAB_TRIGGER_CLS}>
            调用日志
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-4 p-0">
          <OverviewTab />
        </TabsContent>
        <TabsContent value="capabilities" className="mt-4 p-0">
          <CapabilitiesTab />
        </TabsContent>
        <TabsContent value="tokens" className="mt-4 p-0">
          <TokensTab />
        </TabsContent>
        <TabsContent value="logs" className="mt-4 p-0">
          <CallLogsTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
