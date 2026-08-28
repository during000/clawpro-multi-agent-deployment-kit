/**
 * SsoLoginDemo - SSO 登录弹窗交互样式演示页
 *
 * 在首页（LandingPage）的基础上展示 SSO 登录弹窗，
 * 模拟未登录用户访问时的场景。点击相关按钮可触发登录弹窗。
 */
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { SITE_CONFIG } from '@/lib/mockData';
import {
  Bot, Zap, Shield, Cloud, Settings, Users, ArrowRight,
  MessageSquare, Brain, Puzzle, Clock,
} from 'lucide-react';
import SsoLoginDialog from '@/components/SsoLoginDialog';

const CONCEPT_POINTS = [
  {
    icon: Bot,
    title: '什么是 Agent？',
    desc: 'Agent 是一个开源的 AI Agent 框架，让你能够快速创建、部署和管理专属的 AI 智能助理，连接各类大模型与即时通讯工具。',
    gradient: 'from-blue-500 to-blue-600',
  },
  {
    icon: Brain,
    title: '大模型驱动',
    desc: '支持接入腾讯云 DeepSeek、混元、Coding Plan 等主流大模型，也支持自定义模型接入，灵活适配企业需求。',
    gradient: 'from-purple-500 to-purple-600',
  },
  {
    icon: MessageSquare,
    title: '多通道覆盖',
    desc: '支持企业微信、飞书、钉钉、QQ 等主流即时通讯工具，让 AI 助理在用户最常用的工作平台上随时待命。',
    gradient: 'from-green-500 to-green-600',
  },
  {
    icon: Puzzle,
    title: '技能扩展',
    desc: '通过 ClawHub 技能市场，为你的 Agent 安装各类技能插件，包括搜索、文档处理、代码生成等，持续扩展能力边界。',
    gradient: 'from-orange-500 to-orange-600',
  },
];

const FEATURE_POINTS = [
  {
    icon: Shield,
    title: '企业级安全管控',
    desc: '完善的成员权限管理、Tokens 配额控制，确保 AI 使用在可控范围内，保护企业数据安全。',
    gradient: 'from-blue-500 to-indigo-600',
  },
  {
    icon: Users,
    title: '多成员协同',
    desc: '支持企业内多名成员各自创建和管理专属 Agent，统一在企业账号体系下管理，互不干扰。',
    gradient: 'from-purple-500 to-pink-600',
  },
  {
    icon: Settings,
    title: '集中化配置管理',
    desc: '管理员可统一配置可用模型、通道和帮助文档，用户无需关心底层配置，专注于使用 AI 提升工作效率。',
    gradient: 'from-green-500 to-teal-600',
  },
  {
    icon: Cloud,
    title: '云端部署，24小时随时可用',
    desc: '部署在腾讯云服务器上，7×24 小时稳定运行，随时随地通过 IM 工具与你的 AI 助理对话。',
    gradient: 'from-cyan-500 to-blue-600',
  },
  {
    icon: Zap,
    title: '一键配置，小白也能快速上手',
    desc: '极简的创建流程，只需输入名称即可创建 Agent，再按步骤配置通道，几分钟内即可拥有专属 AI 助理。',
    gradient: 'from-yellow-500 to-orange-600',
  },
  {
    icon: Clock,
    title: '实时监控与审计',
    desc: '全面的运营监控面板，实时掌握 Agent 运行状态和 Tokens 消耗情况，操作记录全程可追溯。',
    gradient: 'from-red-500 to-rose-600',
  },
];

export default function SsoLoginDemo() {
  const [dialogVisible, setDialogVisible] = useState(false);

  return (
    <div className="min-h-screen" style={{ background: '#FAFBFF' }}>
      {/* Top Bar - 未登录版本 */}
      <header className="fixed top-0 left-0 right-0 z-50 h-16 bg-white/90 backdrop-blur-md border-b border-gray-100">
        <div className="max-w-7xl mx-auto px-6 h-full flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div
              className="w-8 h-8 rounded-lg flex items-center justify-center text-lg"
              style={{ background: '#FFFFFF' }}
            >
              🦞
            </div>
            <span className="font-semibold text-gray-900 text-sm">{SITE_CONFIG.name}</span>
          </div>
          <button
            className="text-sm text-gray-600 hover:text-blue-600 transition-colors cursor-pointer"
            onClick={() => setDialogVisible(true)}
          >
            进入我的Agent
          </button>
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative min-h-screen flex flex-col items-center justify-center pt-16 overflow-hidden">
        {/* Background orbs */}
        <div className="absolute top-20 right-10 w-96 h-96 orb-blue opacity-60 pointer-events-none" />
        <div className="absolute bottom-20 left-10 w-80 h-80 orb-purple opacity-50 pointer-events-none" />
        <div
          className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] rounded-full pointer-events-none"
          style={{ background: 'radial-gradient(circle, rgba(0,122,255,0.04) 0%, transparent 70%)' }}
        />

        <div className="relative z-10 text-center max-w-3xl mx-auto px-6 page-enter">
          {/* Mascot */}
          <div className="flex justify-center mb-8">
            <div className="relative">
              <div
                className="w-40 h-40 rounded-3xl overflow-hidden flex items-center justify-center relative"
                style={{ background: '#FFFFFF' }}
              >
                <div className="text-8xl select-none" style={{ filter: 'drop-shadow(0 4px 12px rgba(0,122,255,0.2))' }}>🦞</div>
              </div>
              {/* Floating badge */}
              <div
                className="absolute -top-2 -right-2 px-2 py-1 rounded-full text-xs font-semibold text-white"
                style={{ background: 'linear-gradient(135deg, #007AFF, #5856D6)' }}
              >
                Enterprise
              </div>
            </div>
          </div>

          {/* Title */}
          <h1 className="text-5xl font-bold text-gray-900 mb-4 leading-tight">
            {SITE_CONFIG.name}
          </h1>
          <p className="text-xl text-gray-500 mb-8 leading-relaxed">
            快速创建属于你的 24 小时 AI 私人助理<br />
            <span className="text-gray-400 text-lg">对话即可完成各种工作任务，随时随地提升工作效率</span>
          </p>

          {/* CTA */}
          <div className="flex items-center justify-center gap-4">
            <Button
              size="lg"
              className="px-8 py-3 text-base font-semibold rounded-xl text-white btn-primary-glow"
              style={{ background: 'linear-gradient(135deg, #007AFF, #5856D6)' }}
              onClick={() => setDialogVisible(true)}
            >
              立刻创建
              <ArrowRight className="ml-2 w-5 h-5" />
            </Button>
          </div>

          {/* Quick steps */}
          <div className="mt-12 flex items-center justify-center gap-8 text-sm text-gray-400">
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded-full bg-blue-100 text-blue-600 flex items-center justify-center text-xs font-bold">1</div>
              <span>创建 Agent</span>
            </div>
            <div className="w-8 h-px bg-gray-200" />
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded-full bg-blue-100 text-blue-600 flex items-center justify-center text-xs font-bold">2</div>
              <span>配置模型和通道（企微/飞书/钉钉等）</span>
            </div>
            <div className="w-8 h-px bg-gray-200" />
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded-full bg-green-100 text-green-600 flex items-center justify-center text-xs font-bold">✓</div>
              <span>开始使用</span>
            </div>
          </div>
        </div>

        {/* Scroll indicator */}
        <div className="absolute bottom-8 left-1/2 -translate-x-1/2 flex flex-col items-center gap-2 text-gray-300">
          <span className="text-xs">向下滚动了解更多</span>
          <div className="w-5 h-8 border-2 border-gray-200 rounded-full flex items-start justify-center pt-1.5">
            <div className="w-1 h-2 bg-gray-300 rounded-full animate-bounce" />
          </div>
        </div>
      </section>

      {/* Concept Section */}
      <section className="py-24 px-6 relative overflow-hidden">
        <div className="absolute top-0 right-0 w-64 h-64 orb-blue opacity-30 pointer-events-none" />
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-16">
            <span className="inline-block px-3 py-1 rounded-full text-xs font-semibold text-blue-600 bg-blue-50 mb-4">
              概念介绍
            </span>
            <h2 className="text-3xl font-bold text-gray-900 mb-4">什么是 Agent？</h2>
            <p className="text-gray-500 max-w-2xl mx-auto">
              Agent 是一个强大的 AI Agent 平台，让你轻松构建和管理专属的智能助理
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {CONCEPT_POINTS.map((point) => {
              const Icon = point.icon;
              return (
                <div
                  key={point.title}
                  className="bg-white rounded-2xl p-6 border border-gray-100 hover:-translate-y-0.5 transition-all duration-200"
                  style={{ boxShadow: '0 1px 3px rgba(0,0,0,0.06), 0 4px 12px rgba(0,0,0,0.04)' }}
                >
                  <div className={`w-10 h-10 rounded-xl bg-gradient-to-br ${point.gradient} flex items-center justify-center mb-4`}>
                    <Icon className="w-5 h-5 text-white" />
                  </div>
                  <h3 className="text-base font-semibold text-gray-900 mb-2">{point.title}</h3>
                  <p className="text-sm text-gray-500 leading-relaxed">{point.desc}</p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section
        className="py-24 px-6 relative overflow-hidden"
        style={{ background: 'linear-gradient(180deg, #FAFBFF 0%, #F0F4FF 100%)' }}
      >
        <div className="absolute bottom-0 left-0 w-72 h-72 orb-purple opacity-30 pointer-events-none" />
        <div className="max-w-6xl mx-auto">
          <div className="text-center mb-16">
            <span className="inline-block px-3 py-1 rounded-full text-xs font-semibold text-purple-600 bg-purple-50 mb-4">
              功能与特色
            </span>
            <h2 className="text-3xl font-bold text-gray-900 mb-4">企业版 OpenClaw 的功能与特色</h2>
            <p className="text-gray-500 max-w-2xl mx-auto">
              专为企业场景设计，提供完善的管控能力和极致的使用体验
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {FEATURE_POINTS.map((feature) => {
              const Icon = feature.icon;
              return (
                <div
                  key={feature.title}
                  className="bg-white rounded-2xl p-6 border border-gray-100 hover:-translate-y-0.5 transition-all duration-200"
                  style={{ boxShadow: '0 1px 3px rgba(0,0,0,0.06), 0 4px 12px rgba(0,0,0,0.04)' }}
                >
                  <div className={`w-10 h-10 rounded-xl bg-gradient-to-br ${feature.gradient} flex items-center justify-center mb-4`}>
                    <Icon className="w-5 h-5 text-white" />
                  </div>
                  <h3 className="text-base font-semibold text-gray-900 mb-2">{feature.title}</h3>
                  <p className="text-sm text-gray-500 leading-relaxed">{feature.desc}</p>
                </div>
              );
            })}
          </div>

          {/* Bottom CTA */}
          <div className="text-center mt-16">
            <Button
              size="lg"
              className="px-10 py-3 text-base font-semibold rounded-xl text-white btn-primary-glow"
              style={{ background: 'linear-gradient(135deg, #007AFF, #5856D6)' }}
              onClick={() => setDialogVisible(true)}
            >
              立刻开始使用
              <ArrowRight className="ml-2 w-5 h-5" />
            </Button>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-8 px-6 border-t border-gray-100 bg-white">
        <div className="max-w-6xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div
              className="w-6 h-6 rounded-md flex items-center justify-center text-sm"
              style={{ background: 'linear-gradient(135deg, #007AFF, #5856D6)' }}
            >
              🦞
            </div>
            <span className="text-sm text-gray-500">{SITE_CONFIG.name}</span>
          </div>
          <p className="text-xs text-gray-400">© 2026 企业版 OpenClaw. All rights reserved.</p>
        </div>
      </footer>

      {/* SSO 登录弹窗 - 点击按钮触发 */}
      <SsoLoginDialog
        visible={dialogVisible}
        onClose={() => setDialogVisible(false)}
      />
    </div>
  );
}
