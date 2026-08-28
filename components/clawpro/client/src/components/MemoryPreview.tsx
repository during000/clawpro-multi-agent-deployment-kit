import { useState, useEffect } from "react";
import { Pagination } from "@/components/ui/pagination";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { Zap, Database, Layers, User, FileText, ChevronRight, ChevronDown, ChevronLeft, ChevronUp, MessageSquare, ArrowUpDown, Shield, Crown, Loader2, Lock, Calendar, Clock, X } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { DatePicker } from "@/components/ui/date-picker";
import { StatusTag } from "@/components/ui/status-tag";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { TenantCard } from "@/components/ui/Surface";
import memoryStableIcon from "@/assets/icons/agent-page/memory-stable.svg";
import deeperUnderstandingIcon from "@/assets/icons/agent-page/deeper-understanding.svg";
import preciseRetrievalIcon from "@/assets/icons/agent-page/precise-retrieval.svg";
import crossSessionIcon from "@/assets/icons/agent-page/cross-session.svg";

// Mock 数据 - Persona
const mockPersona = `# 用户画像

## 基本信息
- 名称：张三
- 角色：产品经理
- 偏好语言：简洁直接

## 工作习惯
- 喜欢在早晨处理重要任务
- 偏好使用 Markdown 格式记录
- 注重数据驱动决策

## 沟通风格
- 喜欢结构化的信息呈现
- 倾向于先看结论再看过程`;

// Mock 数据 - Scene Blocks（每个都是 md 文件，带有内容）
const mockSceneBlocks = [
  { 
    id: 's1', 
    name: '项目管理.md', 
    content: `# 项目管理场景

## 当前项目
- 正在负责 Memory Pro 产品的功能迭代
- 与后端团队协作开发 API 接口

## 项目习惯
- 使用 TAPD 进行任务管理
- 每周一进行 Sprint Planning`
  },
  { 
    id: 's2', 
    name: '技术调研.md', 
    content: `# 技术调研场景

## 关注技术栈
- React + TypeScript 前端开发
- 向量数据库与 RAG 技术

## 调研偏好
- 优先查看官方文档
- 喜欢对比多种方案的优缺点`
  },
  { 
    id: 's3', 
    name: '会议纪要.md', 
    content: `# 会议纪要场景

## 记录习惯
- 使用 Markdown 格式记录
- 重点标注行动项和负责人

## 常见会议
- 周会、技术评审、需求评审`
  },
  { 
    id: 's4', 
    name: '客户沟通.md', 
    content: `# 客户沟通场景

## 沟通渠道
- 企业微信群进行日常沟通
- 腾讯会议进行需求对齐

## 注意事项
- 记录客户的核心诉求
- 及时同步进展给客户`
  },
  { 
    id: 's5', 
    name: '产品设计.md', 
    content: `# 产品设计场景

## 设计工具
- Figma 进行原型设计
- 墨刀进行快速草图

## 设计原则
- 以用户为中心
- 保持界面简洁清晰`
  },
  { 
    id: 's6', 
    name: '数据分析.md', 
    content: `# 数据分析场景

## 分析工具
- SQL 查询数据库
- Excel 进行数据处理

## 分析维度
- 用户行为分析
- 功能使用率统计`
  },
  { 
    id: 's7', 
    name: '文档编写.md', 
    content: `# 文档编写场景

## 文档类型
- PRD 产品需求文档
- 技术方案文档

## 编写规范
- 结构清晰，分层明确
- 配图辅助说明`
  },
];

// Mock 数据 - Records
const mockRecords = [
  { id: 'r001', type: 'fact', tag: '工作', content: '用户是产品经理，负责 B 端产品线', confidence: 0.95 },
  { id: 'r002', type: 'preference', tag: '沟通', content: '偏好简洁直接的沟通方式，不喜欢冗长的解释', confidence: 0.88 },
  { id: 'r003', type: 'event', tag: '项目', content: '2026-03-28 完成了 Q1 产品规划评审', confidence: 0.92 },
  { id: 'r004', type: 'fact', tag: '技能', content: '熟悉 SQL 和基础数据分析', confidence: 0.85 },
  { id: 'r005', type: 'preference', tag: '工具', content: '常用 Figma 进行原型设计', confidence: 0.90 },
  { id: 'r006', type: 'fact', tag: '团队', content: '所在团队有 8 人，包含前后端和设计师', confidence: 0.87 },
  { id: 'r007', type: 'event', tag: '会议', content: '2026-04-01 参加了技术方案评审会议', confidence: 0.91 },
  { id: 'r008', type: 'preference', tag: '时间', content: '喜欢在上午处理复杂任务，下午处理沟通事务', confidence: 0.82 },
  { id: 'r009', type: 'fact', tag: '产品', content: '负责的产品月活用户约 10 万', confidence: 0.78 },
  { id: 'r010', type: 'event', tag: '发布', content: '2026-03-15 完成了 v2.3 版本的发布', confidence: 0.94 },
  { id: 'r011', type: 'preference', tag: '文档', content: '偏好使用 Markdown 格式编写文档', confidence: 0.89 },
  { id: 'r012', type: 'fact', tag: '技术', content: '对 AI 和大模型技术有浓厚兴趣', confidence: 0.86 },
];

// Mock 数据 - Conversations（平铺的消息列表，只有 sessionId，没有 title）
const mockConversations = [
  { id: 'c001', sessionId: 'session-001', role: 'user', content: '我想讨论一下 Memory Pro 的新功能需求，主要是关于记忆检索的优化。', time: '2026-04-03 10:30' },
  { id: 'c002', sessionId: 'session-001', role: 'assistant', content: '好的，关于记忆检索优化，您希望从哪些方面入手？比如检索速度、准确性还是用户体验？', time: '2026-04-03 10:31' },
  { id: 'c003', sessionId: 'session-001', role: 'user', content: '主要是准确性，目前语义匹配的效果还不够理想，有时候检索不到相关的记忆。', time: '2026-04-03 10:35' },
  { id: 'c004', sessionId: 'session-001', role: 'assistant', content: '理解了。针对语义匹配准确性的问题，我建议可以从以下几个方面优化：\n1. 优化 Embedding 模型的选择\n2. 调整向量相似度阈值\n3. 增加混合检索策略（关键词+语义）', time: '2026-04-03 10:36' },
  { id: 'c005', sessionId: 'session-002', role: 'user', content: '今天我们评审一下向量数据库的选型方案。', time: '2026-04-02 15:20' },
  { id: 'c006', sessionId: 'session-002', role: 'assistant', content: '好的，目前主流的向量数据库有 Milvus、Pinecone、Weaviate 等，您希望我从哪些维度进行对比分析？', time: '2026-04-02 15:21' },
  { id: 'c007', sessionId: 'session-002', role: 'user', content: '主要看性能、成本和易用性。', time: '2026-04-02 15:25' },
  { id: 'c008', sessionId: 'session-002', role: 'assistant', content: '从这三个维度来看：\n\n**性能**：Milvus 在大规模数据下表现优秀，支持多种索引类型\n\n**成本**：Milvus 开源免费，Pinecone 按用量付费\n\n**易用性**：Pinecone 托管服务更易上手，Milvus 需要自行运维', time: '2026-04-02 15:26' },
  { id: 'c009', sessionId: 'session-003', role: 'user', content: '最近收到了一些用户反馈，想和你一起分析一下。', time: '2026-04-01 09:45' },
  { id: 'c010', sessionId: 'session-003', role: 'assistant', content: '好的，请分享用户反馈的内容，我来帮您分析并提出改进建议。', time: '2026-04-01 09:46' },
  { id: 'c011', sessionId: 'session-003', role: 'user', content: '用户反映记忆管理界面不够直观，找不到想要的功能。', time: '2026-04-01 09:50' },
  { id: 'c012', sessionId: 'session-003', role: 'assistant', content: '这是一个 UX 问题。建议：\n1. 添加功能引导提示\n2. 优化导航结构，将常用功能前置\n3. 增加搜索功能，方便快速定位', time: '2026-04-01 09:51' },
  { id: 'c013', sessionId: 'session-004', role: 'user', content: '我们来讨论一下记忆服务的 API 接口设计。', time: '2026-03-31 14:20' },
  { id: 'c014', sessionId: 'session-004', role: 'assistant', content: '好的，关于记忆服务 API，我建议包含以下核心接口：\n1. 记忆写入 POST /memory\n2. 记忆检索 GET /memory/search\n3. 记忆更新 PUT /memory/{id}\n4. 记忆删除 DELETE /memory/{id}', time: '2026-03-31 14:21' },
  { id: 'c015', sessionId: 'session-005', role: 'user', content: '最近系统响应有点慢，想讨论下优化方案。', time: '2026-03-30 11:00' },
  { id: 'c016', sessionId: 'session-005', role: 'assistant', content: '了解，我们可以从以下几个方面入手：\n1. 数据库查询优化\n2. 缓存策略优化\n3. 前端渲染性能优化\n\n您想先从哪个方面开始？', time: '2026-03-30 11:01' },
  { id: 'c017', sessionId: 'session-006', role: 'user', content: '下个版本的发布计划是什么？', time: '2026-03-29 16:30' },
  { id: 'c018', sessionId: 'session-006', role: 'assistant', content: '根据之前的讨论，v2.4 版本计划包含：\n1. 记忆检索优化\n2. 新增批量导入功能\n3. UI 体验改进\n\n预计发布时间为 4 月中旬。', time: '2026-03-29 16:31' },
  { id: 'c019', sessionId: 'session-007', role: 'user', content: '帮我分析一下市面上的竞品。', time: '2026-03-28 10:15' },
  { id: 'c020', sessionId: 'session-007', role: 'assistant', content: '目前主要竞品包括：\n1. Mem0 - 开源记忆框架\n2. Zep - 长期记忆存储\n3. MemGPT - 自主记忆管理\n\n各有优缺点，您想深入了解哪个？', time: '2026-03-28 10:16' },
];

// Memory 状态类型：pro / free / none / upgrading（升级中）
type MemoryStatus = 'pro' | 'free' | 'none' | 'upgrading';

interface MemoryPreviewProps {
  // 当前实例的 Memory 状态
  memoryStatus?: MemoryStatus;
  // 是否在原子记忆中显示置信度列（默认 true）
  showConfidence?: boolean;
  // Pro 版额度是否可用
  proQuotaAvailable?: boolean;
  // Memory 状态变更回调
  onStatusChange?: (newStatus: MemoryStatus) => Promise<void>;
  // 是否正在加载数据（首次进入时加载）
  isLoading?: boolean;
  // [本期新增] 是否处于 Pro 免费体验期
  isFreeTrial?: boolean;
  // [本期新增] 免费体验预计结束时间（展示用字符串，如 "2026-06-30"）
  freeTrialEndDate?: string;
  // 是否显示组件内标题区（外层 Tab 已有标题描述时可关闭）
  showHeader?: boolean;
}

// 记忆类型标签组件
function TypeBadge({ type }: { type: string }) {
  const config: Record<string, { color: "blue" | "green" | "purple" | "red"; label: string }> = {
    fact: { color: 'blue', label: '事实' },
    preference: { color: 'green', label: '偏好' },
    event: { color: 'purple', label: '事件' },
  };
  const c = config[type] || config.fact;
  return <Badge color={c.color}>{c.label}</Badge>;
}

// Pro 版左侧导航项类型
type NavItem = 'persona' | 'scenes' | 'records' | 'conversations';

export function MemoryPreview({ 
  memoryStatus = 'none',
  showConfidence = true,
  proQuotaAvailable: _proQuotaAvailable = true,
  onStatusChange: _onStatusChange,
  isLoading = false,
  isFreeTrial = false,
  freeTrialEndDate,
  showHeader = true,
}: MemoryPreviewProps) {
  const [activeNav, setActiveNav] = useState<NavItem>('persona');
  const [recordFilter, setRecordFilter] = useState<'all' | 'fact' | 'preference' | 'event'>('all');
  const [sortByConfidence, setSortByConfidence] = useState<'none' | 'asc' | 'desc'>('none');
  const [expandedScene, setExpandedScene] = useState<string | null>(mockSceneBlocks[0]?.id ?? null);
  
  // 对话记录状态
  const [convSessionFilter, setConvSessionFilter] = useState<string>('all');
  const [convSortOrder, setConvSortOrder] = useState<'asc' | 'desc'>('desc');
  const [expandedConvId, setExpandedConvId] = useState<string | null>(null); // 当前展开的对话 ID
  
  // 时间筛选状态（默认选中近7天）
  const [convTimeFilter, setConvTimeFilter] = useState<'7days' | '30days' | 'custom'>('7days');
  const [convCustomStartDate, setConvCustomStartDate] = useState<string>('');
  const [convCustomEndDate, setConvCustomEndDate] = useState<string>('');
  const [showDatePicker, setShowDatePicker] = useState(false);
  
  // 每页显示条数（固定为 10）
  const pageSize = 10;

  // 免费体验期提示条（仅 Pro 且 isFreeTrial=true 时展示）
  const renderFreeTrialBanner = () => {
    if (!isFreeTrial) return null;
    return (
      <div className="mb-4 px-4 py-2.5 rounded-[4px] bg-amber-50 border border-amber-100 flex items-center gap-2 text-xs text-amber-700">
        <Clock className="w-3.5 h-3.5 flex-shrink-0" />
        <span className="leading-relaxed">
          当前处于 Pro 免费体验期
          {freeTrialEndDate && <> · 预计结束时间 <span className="font-medium">{freeTrialEndDate}</span></>}
          。免费期结束后不会自动扣费，如需延续请在控制台确认转为付费。
        </span>
      </div>
    );
  };

  // 分页状态
  const [scenesPage, setScenesPage] = useState(1);
  const [recordsPage, setRecordsPage] = useState(1);
  const [conversationsPage, setConversationsPage] = useState(1);

  // 过滤和排序记录
  const filteredRecords = mockRecords
    .filter(r => {
      if (recordFilter !== 'all' && r.type !== recordFilter) return false;
      return true;
    })
    .sort((a, b) => {
      if (sortByConfidence === 'asc') return a.confidence - b.confidence;
      if (sortByConfidence === 'desc') return b.confidence - a.confidence;
      return 0;
    });

  // 切换场景块展开状态（只允许展开一个）
  const toggleSceneExpand = (id: string) => {
    setExpandedScene(prev => prev === id ? null : id);
  };

  // 获取所有 sessionId 列表
  const sessionIds = Array.from(new Set(mockConversations.map(c => c.sessionId))).sort();

  // 过滤和排序对话记录
  const filteredConversations = mockConversations
    .filter(c => {
      // Session 筛选
      if (convSessionFilter !== 'all' && c.sessionId !== convSessionFilter) return false;
      
      // 时间筛选
      const convTime = new Date(c.time).getTime();
      const now = new Date().getTime();
      const dayMs = 24 * 60 * 60 * 1000;
      
      if (convTimeFilter === '7days') {
        const sevenDaysAgo = now - 7 * dayMs;
        if (convTime < sevenDaysAgo) return false;
      } else if (convTimeFilter === '30days') {
        const thirtyDaysAgo = now - 30 * dayMs;
        if (convTime < thirtyDaysAgo) return false;
      } else if (convTimeFilter === 'custom' && convCustomStartDate && convCustomEndDate) {
        const startTime = new Date(convCustomStartDate).getTime();
        const endTime = new Date(convCustomEndDate).getTime() + dayMs - 1; // 包含结束日期当天
        if (convTime < startTime || convTime > endTime) return false;
      }
      
      return true;
    })
    .sort((a, b) => {
      const timeA = new Date(a.time).getTime();
      const timeB = new Date(b.time).getTime();
      return convSortOrder === 'desc' ? timeB - timeA : timeA - timeB;
    });

  // 分页数据计算
  const paginatedScenes = mockSceneBlocks.slice((scenesPage - 1) * pageSize, scenesPage * pageSize);
  const totalScenesPages = Math.ceil(mockSceneBlocks.length / pageSize);
  
  const paginatedRecords = filteredRecords.slice((recordsPage - 1) * pageSize, recordsPage * pageSize);
  const totalRecordsPages = Math.ceil(filteredRecords.length / pageSize);
  
  const paginatedConversations = filteredConversations.slice((conversationsPage - 1) * pageSize, conversationsPage * pageSize);
  const totalConversationsPages = Math.ceil(filteredConversations.length / pageSize);

  // 记忆管理自定义图标
  const MemoryPersonaIcon = ({ active }: { active?: boolean }) => (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-4 h-4">
      <path d="M14.6487 13.125C13.7918 11.621 12.4538 10.4492 10.85 9.79811C11.6476 9.19994 12.2367 8.366 12.5339 7.4144C12.8312 6.46281 12.8215 5.44181 12.5062 4.49603C12.191 3.55024 11.5861 2.72763 10.7774 2.14471C9.96861 1.56179 8.99694 1.24811 8 1.24811C7.00306 1.24811 6.03138 1.56179 5.22262 2.14471C4.41386 2.72763 3.80901 3.55024 3.49375 4.49603C3.17849 5.44181 3.1688 6.46281 3.46606 7.4144C3.76331 8.366 4.35244 9.19994 5.15 9.79811C3.5462 10.4492 2.20815 11.621 1.35125 13.125C1.29815 13.2104 1.26276 13.3055 1.24718 13.4049C1.23161 13.5042 1.23616 13.6056 1.26057 13.7032C1.28499 13.8007 1.32876 13.8923 1.38929 13.9726C1.44982 14.0529 1.52588 14.1202 1.61293 14.1705C1.69999 14.2208 1.79627 14.253 1.89605 14.2654C1.99583 14.2777 2.09707 14.2699 2.19376 14.2423C2.29045 14.2148 2.38061 14.168 2.45888 14.1049C2.53715 14.0418 2.60193 13.9636 2.64937 13.875C3.78187 11.9175 5.78187 10.75 8 10.75C10.2181 10.75 12.2181 11.9181 13.3506 13.875C13.4534 14.0403 13.6165 14.1592 13.8054 14.2065C13.9943 14.2537 14.1941 14.2257 14.3627 14.1283C14.5313 14.0309 14.6554 13.8718 14.7088 13.6845C14.7621 13.4973 14.7406 13.2966 14.6487 13.125ZM4.75 5.99998C4.75 5.35719 4.94061 4.72884 5.29772 4.19438C5.65484 3.65992 6.16242 3.24336 6.75628 2.99737C7.35014 2.75139 8.0036 2.68703 8.63404 2.81243C9.26448 2.93783 9.84357 3.24736 10.2981 3.70189C10.7526 4.15641 11.0621 4.7355 11.1875 5.36594C11.313 5.99638 11.2486 6.64984 11.0026 7.2437C10.7566 7.83756 10.3401 8.34514 9.8056 8.70226C9.27114 9.05937 8.64279 9.24998 8 9.24998C7.13835 9.24899 6.31227 8.90626 5.703 8.29698C5.09372 7.68771 4.75099 6.86163 4.75 5.99998Z" fill={active ? "url(#paint0_linear_memory_persona)" : "#A3A3A3"}/>
      {active && (
        <defs>
          <linearGradient id="paint0_linear_memory_persona" x1="13.6254" y1="14.4195" x2="11.468" y2="8.93355" gradientUnits="userSpaceOnUse">
            <stop stopColor="#0080FF"/>
            <stop offset="1" stopColor="#202020"/>
          </linearGradient>
        </defs>
      )}
    </svg>
  );

  const MemorySceneIcon = ({ active }: { active?: boolean }) => (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-4 h-4">
      <g clipPath="url(#clip0_memory_scene)">
        <path d="M15.5369 3.66687C15.1425 2.98749 14.0994 2.35062 11.4019 3.05999C10.5016 2.43871 9.44844 2.07555 8.35654 2.00983C7.26464 1.9441 6.17556 2.17832 5.20721 2.68711C4.23887 3.19591 3.42815 3.9599 2.86283 4.89637C2.29751 5.83284 1.99914 6.90612 2 7.99999C2 8.14999 2.00542 8.29874 2.01625 8.44624C0.0350031 10.4287 0.0687533 11.65 0.465003 12.3331C0.837503 12.9756 1.58125 13.25 2.5425 13.25C3.15438 13.25 3.855 13.1387 4.60188 12.9431C5.50239 13.5635 6.55542 13.9258 7.64701 13.9909C8.73861 14.056 9.82722 13.8213 10.795 13.3122C11.7629 12.8032 12.5731 12.0392 13.138 11.1028C13.7029 10.1665 14.001 9.09353 14 7.99999C14 7.84937 13.9944 7.70062 13.9831 7.55249C14.8775 6.65374 15.4744 5.78937 15.6706 5.05249C15.8469 4.40124 15.695 3.93749 15.5369 3.66687ZM8 3.49999C9.01925 3.50137 10.0079 3.8482 10.8047 4.48387C11.6014 5.11953 12.1592 6.0065 12.3869 6.99999C11.5 7.79562 10.3125 8.66187 8.87 9.49124C7.51125 10.2719 6.14813 10.8887 4.92688 11.2819C4.26922 10.6649 3.81209 9.86456 3.61477 8.98465C3.41745 8.10474 3.48905 7.18585 3.82027 6.34712C4.15149 5.50839 4.72706 4.78852 5.47234 4.28085C6.21763 3.77318 7.09824 3.50114 8 3.49999ZM1.76188 11.5806C1.72375 11.5137 1.75125 11.0669 2.42875 10.2237C2.63919 10.7491 2.92306 11.2419 3.27188 11.6875C2.195 11.8506 1.80313 11.6506 1.76188 11.5806ZM8 12.5C7.52086 12.5001 7.04481 12.4233 6.59 12.2725C7.63136 11.8486 8.64248 11.3539 9.61625 10.7919C10.5864 10.2386 11.5171 9.61901 12.4019 8.93749C12.1859 9.94499 11.6312 10.8481 10.8302 11.4963C10.0293 12.1445 9.0304 12.4987 8 12.5ZM14.2231 4.66562C14.1431 4.96437 13.9225 5.34499 13.5738 5.77812C13.3633 5.25209 13.0792 4.75863 12.73 4.31249C13.7194 4.16437 14.1781 4.31249 14.2394 4.41937C14.25 4.43749 14.2631 4.51749 14.2231 4.66562Z" fill={active ? "url(#paint0_linear_memory_scene)" : "#A3A3A3"}/>
      </g>
      <defs>
        <clipPath id="clip0_memory_scene">
          <rect width="16" height="16" fill="white"/>
        </clipPath>
        {active && (
          <linearGradient id="paint0_linear_memory_scene" x1="13.6254" y1="14.4195" x2="11.468" y2="8.93355" gradientUnits="userSpaceOnUse">
            <stop stopColor="#0080FF"/>
            <stop offset="1" stopColor="#202020"/>
          </linearGradient>
        )}
      </defs>
    </svg>
  );

  const MemoryAtomIcon = ({ active }: { active?: boolean }) => (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-4 h-4">
      <path d="M13.75 2H4.75C4.41848 2 4.10054 2.1317 3.86612 2.36612C3.6317 2.60054 3.5 2.91848 3.5 3.25V4.5H2.25C1.91848 4.5 1.60054 4.6317 1.36612 4.86612C1.1317 5.10054 1 5.41848 1 5.75V12.75C1 13.0815 1.1317 13.3995 1.36612 13.6339C1.60054 13.8683 1.91848 14 2.25 14H11.25C11.5815 14 11.8995 13.8683 12.1339 13.6339C12.3683 13.3995 12.5 13.0815 12.5 12.75V11.5H13.75C14.0815 11.5 14.3995 11.3683 14.6339 11.1339C14.8683 10.8995 15 10.5815 15 10.25V3.25C15 2.91848 14.8683 2.60054 14.6339 2.36612C14.3995 2.1317 14.0815 2 13.75 2ZM11 6V7H2.5V6H11ZM11 12.5H2.5V8.5H11V12.5ZM13.5 10H12.5V5.75C12.5 5.41848 12.3683 5.10054 12.1339 4.86612C11.8995 4.6317 11.5815 4.5 11.25 4.5H5V3.5H13.5V10Z" fill={active ? "url(#paint0_linear_memory_atom)" : "#A3A3A3"}/>
      {active && (
        <defs>
          <linearGradient id="paint0_linear_memory_atom" x1="13.6254" y1="14.4195" x2="11.468" y2="8.93355" gradientUnits="userSpaceOnUse">
            <stop stopColor="#0080FF"/>
            <stop offset="1" stopColor="#202020"/>
          </linearGradient>
        </defs>
      )}
    </svg>
  );

  const MemoryDialogueIcon = ({ active }: { active?: boolean }) => (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-4 h-4">
      <path d="M5.50001 8C5.50001 7.80222 5.55866 7.60888 5.66854 7.44443C5.77842 7.27998 5.9346 7.15181 6.11733 7.07612C6.30005 7.00043 6.50112 6.98063 6.6951 7.01921C6.88908 7.0578 7.06727 7.15304 7.20712 7.29289C7.34697 7.43275 7.44221 7.61093 7.4808 7.80491C7.51938 7.99889 7.49958 8.19996 7.42389 8.38268C7.3482 8.56541 7.22003 8.72159 7.05558 8.83147C6.89113 8.94135 6.69779 9 6.50001 9C6.23479 9 5.98044 8.89464 5.7929 8.70711C5.60537 8.51957 5.50001 8.26522 5.50001 8ZM9.50001 9C9.69779 9 9.89113 8.94135 10.0556 8.83147C10.22 8.72159 10.3482 8.56541 10.4239 8.38268C10.4996 8.19996 10.5194 7.99889 10.4808 7.80491C10.4422 7.61093 10.347 7.43275 10.2071 7.29289C10.0673 7.15304 9.88908 7.0578 9.6951 7.01921C9.50112 6.98063 9.30005 7.00043 9.11733 7.07612C8.9346 7.15181 8.77842 7.27998 8.66854 7.44443C8.55866 7.60888 8.50001 7.80222 8.50001 8C8.50001 8.26522 8.60537 8.51957 8.7929 8.70711C8.98044 8.89464 9.23479 9 9.50001 9ZM14.75 4V12C14.75 12.3315 14.6183 12.6495 14.3839 12.8839C14.1495 13.1183 13.8315 13.25 13.5 13.25H5.27939L3.31251 14.9481L3.30501 14.955C3.08091 15.1449 2.79687 15.2494 2.50314 15.25C2.31972 15.2495 2.13862 15.209 1.97251 15.1313C1.75612 15.032 1.57289 14.8726 1.44476 14.672C1.31663 14.4713 1.24901 14.238 1.25001 14V4C1.25001 3.66848 1.38171 3.35054 1.61613 3.11612C1.85055 2.8817 2.16849 2.75 2.50001 2.75H13.5C13.8315 2.75 14.1495 2.8817 14.3839 3.11612C14.6183 3.35054 14.75 3.66848 14.75 4ZM13.25 4.25H2.75001V13.4519L4.51001 11.9319C4.64605 11.8141 4.82008 11.7495 5.00001 11.75H13.25V4.25Z" fill={active ? "url(#paint0_linear_memory_dialogue)" : "#A3A3A3"}/>
      {active && (
        <defs>
          <linearGradient id="paint0_linear_memory_dialogue" x1="13.6254" y1="14.4195" x2="11.468" y2="8.93355" gradientUnits="userSpaceOnUse">
            <stop stopColor="#0080FF"/>
            <stop offset="1" stopColor="#202020"/>
          </linearGradient>
        </defs>
      )}
    </svg>
  );

  // Pro 版左侧导航 - 左侧流程线 + 顶部箭头
  const ProNavigation = () => {
    const navItems = [
      { key: 'persona', icon: MemoryPersonaIcon, label: '个性化记忆' },
      { key: 'scenes', icon: MemoryAtomIcon, label: '场景记忆' },
      { key: 'records', icon: MemorySceneIcon, label: '原子记忆' },
      { key: 'conversations', icon: MemoryDialogueIcon, label: '对话记录' },
    ];

    return (
      <div className="w-40 flex-shrink-0 pr-4">
        <div className="flex flex-col">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = activeNav === item.key;
            
            return (
              <button
                key={item.key}
                onClick={() => setActiveNav(item.key as typeof activeNav)}
                className={`w-full flex items-center gap-2 px-3 py-2 rounded-[4px] text-sm transition-colors mb-0.5 ${
                  isActive 
                    ? 'bg-blue-50 text-[#0A0A0A] font-medium' 
                    : 'text-gray-600 hover:bg-gray-50'
                }`}
              >
                <Icon active={isActive} />
                {item.label}
              </button>
            );
          })}
        </div>
      </div>
    );
  };

  // Persona 面板 - 用户画像，样式与场景记忆卡片一致
  const PersonaPanel = () => (
    <div className="h-full flex flex-col">
      <div className="bg-white border border-gray-200 rounded-[12px] overflow-hidden flex-1 overflow-y-auto">
        {/* 白色标题栏 */}
        <div className="px-4 py-3 border-b border-gray-200">
          <span className="text-sm font-medium text-[#0A0A0A]">用户画像</span>
        </div>
        {/* 内容区 */}
        <div className="px-4 py-3 bg-[#FAFAFA]">
          <div className="space-y-2">
            {mockPersona.split('\n').map((line, i) => {
              if (!line.trim()) return null;
              if (line.startsWith('# ')) return null;
              if (line.startsWith('## ')) {
                return <h4 key={i} className="text-sm font-medium text-[#0A0A0A] mt-3">{line.replace('## ', '')}</h4>;
              }
              if (line.startsWith('- ')) {
                return (
                  <div key={i} className="flex items-start gap-2 pl-2">
                    <span className="w-1.5 h-1.5 rounded-full bg-[#A3A3A3] mt-1.5 shrink-0" />
                    <span className="text-sm text-[#525252] leading-relaxed">{line.replace('- ', '')}</span>
                  </div>
                );
              }
              return <p key={i} className="text-sm text-[#525252] leading-relaxed">{line}</p>;
            })}
          </div>
        </div>
      </div>
    </div>
  );

  // Scene Blocks 面板 - 每个都是 md 文件，点击后折叠展开（只允许展开一个）
  const ScenesPanel = () => (
    <div className="h-full flex flex-col">
      <div className="space-y-2 flex-1 overflow-auto">
        {paginatedScenes.map(scene => {
          const isExpanded = expandedScene === scene.id;
          return (
            <div key={scene.id} className="bg-white border border-gray-200 rounded-[12px] overflow-hidden">
              <button
                onClick={() => toggleSceneExpand(scene.id)}
                className="w-full flex items-center gap-3 p-4 hover:bg-gray-50 transition-colors"
              >
                <div className="w-5 h-5 flex items-center justify-center flex-shrink-0">
                  {isExpanded 
                    ? <ChevronDown className="w-4 h-4 text-[#737373]" />
                    : <ChevronRight className="w-4 h-4 text-[#737373]" />
                  }
                </div>
                <div className="flex-1 min-w-0 text-left">
                  <div className="font-medium text-gray-900 text-sm">{scene.name.replace(/\.md$/, '')}</div>
                </div>
              </button>
              {isExpanded && (
                <div className="border-t border-gray-200 px-4 py-3 max-h-[200px] overflow-auto bg-[#FAFAFA]">
                  <div className="space-y-2">
                    {(() => {
                      // 去掉内容区第一行的一级标题（与卡片头部的 scene.name 重复）
                      const lines = scene.content.split('\n');
                      if (lines[0]?.startsWith('# ')) {
                        lines.shift();
                        while (lines.length > 0 && !lines[0].trim()) {
                          lines.shift();
                        }
                      }
                      return lines;
                    })().map((line, i) => {
                      if (!line.trim()) return null;
                      if (line.startsWith('# ')) {
                        return <h4 key={i} className="text-sm font-semibold text-[#0A0A0A]">{line.replace('# ', '')}</h4>;
                      }
                      if (line.startsWith('## ')) {
                        return <h5 key={i} className="text-sm font-medium text-[#0A0A0A] mt-2">{line.replace('## ', '')}</h5>;
                      }
                      if (line.startsWith('- ')) {
                        return (
                          <div key={i} className="flex items-start gap-2 pl-2">
                            <span className="w-1.5 h-1.5 rounded-full bg-[#A3A3A3] mt-1.5 shrink-0" />
                            <span className="text-sm text-[#525252] leading-relaxed">{line.replace('- ', '')}</span>
                          </div>
                        );
                      }
                      return <p key={i} className="text-sm text-[#525252] leading-relaxed">{line}</p>;
                    })}
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
      <Pagination total={mockSceneBlocks.length} current={scenesPage} pageSize={pageSize} showTotal={(total) => `共 ${total} 条记录`} simple className="w-full justify-between mt-4 pt-4 border-t border-gray-200" onChange={(p) => setScenesPage(p)} />
    </div>
  );

  // Records 面板
  const RecordsPanel = () => (
    <div className="h-full flex flex-col">
      {/* 过滤器（§8.1 按钮规范） */}
      <div className="flex items-center justify-between mb-4 flex-wrap gap-3">
        <div className="flex gap-2">
          {(['all', 'fact', 'preference', 'event'] as const).map(filter => (
            <Button
              key={filter}
              variant="tenant-plain"
              size="sm"
              data-state={recordFilter === filter ? "active" : undefined}
              onClick={() => { setRecordFilter(filter); setRecordsPage(1); }}
            >
              {filter === 'all' ? '全部' : filter === 'fact' ? '事实' : filter === 'preference' ? '偏好' : '事件'}
            </Button>
          ))}
        </div>
      </div>

      {/* 记录表格（§8.4 表格规范） */}
      <div className="bg-white border border-gray-200 rounded-[12px] overflow-hidden flex-1 flex flex-col">
        <Table autoFixedColumns={false}>
          <TableHeader>
            <TableRow>
              <TableHead>类型</TableHead>
              <TableHead>标签</TableHead>
              <TableHead>内容</TableHead>
              {showConfidence && (
                <TableHead>
                  <button
                    onClick={() => {
                      setSortByConfidence(prev => {
                        if (prev === 'none') return 'desc';
                        if (prev === 'desc') return 'asc';
                        return 'none';
                      });
                    }}
                    className="inline-flex items-center gap-1 hover:text-[#1447E6] transition-colors"
                  >
                    置信度
                    <ArrowUpDown className="w-3 h-3" />
                  </button>
                </TableHead>
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {paginatedRecords.map(record => (
              <TableRow key={record.id}>
                <TableCell><TypeBadge type={record.type} /></TableCell>
                <TableCell>
                  <Badge variant="secondary">{record.tag}</Badge>
                </TableCell>
                <TableCell className="text-gray-700 max-w-[280px] truncate">{record.content}</TableCell>
                {showConfidence && (
                  <TableCell>
                    <span className={`font-semibold ${
                      record.confidence >= 0.9 ? 'text-[#16A34A]' :
                      record.confidence >= 0.8 ? 'text-[#F59E0B]' : 'text-[var(--text-muted)]'
                    }`}>
                      {record.confidence.toFixed(2)}
                    </span>
                  </TableCell>
                )}
              </TableRow>
            ))}
          </TableBody>
        </Table>
        {paginatedRecords.length === 0 && (
          <div className="text-center py-12 text-sm text-[var(--text-weak)]">暂无匹配的记忆记录</div>
        )}
      </div>
      <Pagination total={filteredRecords.length} current={recordsPage} pageSize={pageSize} showTotal={(total) => `共 ${total} 条记录`} simple className="w-full justify-between mt-4 pt-4 border-t border-gray-200" onChange={(p) => setRecordsPage(p)} />
    </div>
  );

  // Conversations 面板 - 平铺展示消息列表，支持按 sessionId 筛选
  const ConversationsPanel = () => {
    // 计算自定义日期范围是否有效（最多30天）
    const isCustomDateValid = () => {
      if (!convCustomStartDate || !convCustomEndDate) return false;
      const start = new Date(convCustomStartDate).getTime();
      const end = new Date(convCustomEndDate).getTime();
      const dayMs = 24 * 60 * 60 * 1000;
      const diffDays = (end - start) / dayMs;
      return diffDays >= 0 && diffDays <= 30;
    };

    // 处理快速筛选按钮点击
    const handleQuickFilter = (type: '7days' | '30days') => {
      setConvTimeFilter(type);
      setShowDatePicker(false);
      setConversationsPage(1);
    };

    // 应用自定义日期范围
    const applyCustomDateRange = () => {
      if (isCustomDateValid()) {
        setConvTimeFilter('custom');
        setShowDatePicker(false);
        setConversationsPage(1);
      }
    };

    return (
      <div className="h-full flex flex-col">
        {/* 筛选器 */}
        <div className="flex items-center gap-3 mb-4 flex-wrap">
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-500">会话:</span>
            <Select value={convSessionFilter} onValueChange={(v) => { setConvSessionFilter(v); setConversationsPage(1); }}>
              <SelectTrigger className="h-8 w-[160px] text-xs rounded-[4px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部</SelectItem>
                {sessionIds.map(id => (
                  <SelectItem key={id} value={id}>{id}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* 时间筛选 */}
          <div className="flex items-center gap-2 relative">
            <div className="inline-flex items-center gap-1 p-1 rounded-[4px]" style={{ background: "#F5F5F5" }}>
              {(['7days', '30days'] as const).map(type => (
                <button
                  key={type}
                  onClick={() => handleQuickFilter(type)}
                  className={`px-3 py-1.5 text-xs rounded-[3px] transition-all duration-150 ${
                    convTimeFilter === type
                      ? 'bg-white text-[#0A0A0A] font-medium'
                      : 'text-[#737373] hover:text-[#0A0A0A] font-normal'
                  }`}
                  style={convTimeFilter === type ? { boxShadow: "0 1px 2px rgba(0,0,0,0.06)" } : undefined}
                >
                  {type === '7days' ? '近7天' : '近30天'}
                </button>
              ))}
            </div>

            {/* 自定义日期按钮 */}
            <button
              onClick={() => {
                if (showDatePicker) {
                  setShowDatePicker(false);
                } else {
                  setShowDatePicker(true);
                  const today = new Date();
                  const sevenDaysAgo = new Date(today.getTime() - 7 * 24 * 60 * 60 * 1000);
                  if (!convCustomStartDate || !convCustomEndDate) {
                    setConvCustomEndDate(today.toISOString().split('T')[0]);
                    setConvCustomStartDate(sevenDaysAgo.toISOString().split('T')[0]);
                  }
                }
              }}
              className={`w-8 h-8 flex items-center justify-center rounded-[4px] border transition-colors ${
                convTimeFilter === 'custom' || showDatePicker
                  ? 'bg-[#EFF6FF] border-[#1447E6] text-[#1447E6]'
                  : 'bg-white border-gray-200 text-[#737373] hover:text-[#0A0A0A] hover:border-[#1447E6]'
              }`}
              title="自定义日期范围"
            >
              <Calendar className="w-4 h-4" />
            </button>

            {/* 显示当前自定义筛选范围 */}
            {convTimeFilter === 'custom' && !showDatePicker && convCustomStartDate && convCustomEndDate && (
              <div className="flex items-center gap-1.5 px-2 py-1 bg-[#EFF6FF] text-[#1447E6] rounded-[4px] text-xs">
                <span>{convCustomStartDate} ~ {convCustomEndDate}</span>
                <button
                  onClick={() => {
                    setConvTimeFilter('7days');
                    setConvCustomStartDate('');
                    setConvCustomEndDate('');
                  }}
                  className="p-0.5 hover:bg-[#1447E6]/10 rounded transition-colors"
                >
                  <X className="w-3 h-3" />
                </button>
              </div>
            )}

            {/* 自定义日期选择器弹窗 */}
            {showDatePicker && (
              <div
                className="absolute right-0 top-full mt-2 p-4 bg-white border border-gray-200 rounded-[4px] z-10"
                style={{ boxShadow: "0 1px 3px rgba(0,0,0,0.06), 0 4px 12px rgba(0,0,0,0.04)" }}
              >
                <div className="flex flex-col gap-3">
                  <div className="flex items-center gap-2">
                    <DatePicker
                      value={convCustomStartDate}
                      onChange={(v) => setConvCustomStartDate(v)}
                      placeholder="开始日期"
                      className="h-8 text-xs w-[132px]"
                    />
                    <span className="text-xs text-[#737373]">至</span>
                    <DatePicker
                      value={convCustomEndDate}
                      onChange={(v) => setConvCustomEndDate(v)}
                      placeholder="结束日期"
                      className="h-8 text-xs w-[132px]"
                    />
                  </div>
                  {convCustomStartDate && convCustomEndDate && !isCustomDateValid() && (
                    <span className="text-xs text-[#d42a1e]">日期范围最多30天</span>
                  )}
                  <div className="flex items-center justify-end gap-2">
                    <button
                      onClick={() => setShowDatePicker(false)}
                      className="px-3 py-1.5 text-xs text-[#737373] hover:text-[#0A0A0A] rounded-[4px] border border-gray-200 hover:bg-[#F5F5F5] transition-colors"
                    >
                      取消
                    </button>
                    <button
                      onClick={applyCustomDateRange}
                      disabled={!isCustomDateValid()}
                      className={`px-3 py-1.5 text-xs rounded-[4px] transition-colors ${
                        isCustomDateValid()
                          ? 'bg-[#0A0A0A] text-white hover:opacity-90'
                          : 'bg-[#F5F5F5] text-[#A3A3A3] cursor-not-allowed'
                      }`}
                    >
                      确定
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* 消息表格（§8.4 表格规范） */}
        <div className="bg-white border border-gray-200 rounded-[12px] overflow-hidden flex-1 flex flex-col">
          <Table autoFixedColumns={false}>
            <TableHeader>
              <TableRow>
                <TableHead>会话 ID</TableHead>
                <TableHead>角色</TableHead>
                <TableHead>内容</TableHead>
                <TableHead>
                  <button
                    onClick={() => setConvSortOrder(prev => prev === 'desc' ? 'asc' : 'desc')}
                    className="inline-flex items-center gap-1 hover:text-[#1447E6] transition-colors"
                  >
                    时间
                    <ArrowUpDown className="w-3 h-3" />
                  </button>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginatedConversations.map(conv => {
                const isExpanded = expandedConvId === conv.id;
                const isLongContent = conv.content.length > 80;
                return (
                  <TableRow key={conv.id}>
                    <TableCell>
                      <span className="text-[var(--text-muted)] font-mono">{conv.sessionId}</span>
                    </TableCell>
                    <TableCell>
                      <Badge color={conv.role === 'user' ? 'blue' : 'green'}>
                        {conv.role}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-gray-700 max-w-[320px]">
                      <div className={isExpanded ? 'whitespace-pre-wrap' : 'line-clamp-2'}>
                        {conv.content}
                      </div>
                      {isLongContent && (
                        <button
                          onClick={() => setExpandedConvId(isExpanded ? null : conv.id)}
                          className="text-[#1447E6] hover:opacity-80 mt-1 inline-flex items-center gap-0.5"
                        >
                          {isExpanded ? (
                            <>收起 <ChevronUp className="w-3 h-3" /></>
                          ) : (
                            <>展开 <ChevronDown className="w-3 h-3" /></>
                          )}
                        </button>
                      )}
                    </TableCell>
                    <TableCell className="text-[var(--text-muted)] whitespace-nowrap">{conv.time}</TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
          {paginatedConversations.length === 0 && (
            <div className="text-center py-12 text-sm text-[var(--text-weak)]">暂无匹配的对话记录</div>
          )}
        </div>
        <Pagination total={filteredConversations.length} current={conversationsPage} pageSize={pageSize} showTotal={(total) => `共 ${total} 条记录`} simple className="w-full justify-between mt-4 pt-4 border-t border-gray-200" onChange={(p) => setConversationsPage(p)} />
      </div>
    );
  };

  // Pro 版内容区域
  const ProContent = () => (
    <div className="flex-1 pl-4 overflow-hidden flex flex-col">
      {activeNav === 'persona' && <PersonaPanel />}
      {activeNav === 'scenes' && <ScenesPanel />}
      {activeNav === 'records' && <RecordsPanel />}
      {activeNav === 'conversations' && <ConversationsPanel />}
    </div>
  );

  // ══════════════════════════════════════════════════════════════════════════════
  // 数据面板加载状态 - 简单的 loading 占位
  // ══════════════════════════════════════════════════════════════════════════════
  const DataLoadingPlaceholder = () => (
    <div className="flex-1 flex flex-col items-center justify-center p-8">
      <Loader2 className="w-8 h-8 text-blue-500 animate-spin mb-4" />
      <p className="text-sm text-gray-500">正在加载记忆数据，请稍候...</p>
      <p className="text-xs text-gray-400 mt-1">首次加载可能需要一些时间</p>
    </div>
  );

  // ══════════════════════════════════════════════════════════════════════════════
  // 升级中状态 - 简单的加载提示
  // ══════════════════════════════════════════════════════════════════════════════
  if (memoryStatus === 'upgrading') {
    return (
      <div className="w-full h-full flex flex-col">
        {/* 升级中头部 */}
        <div className="mb-6">
          <div className="flex items-center gap-3">
            <div 
              className="w-10 h-10 rounded-[4px] flex items-center justify-center flex-shrink-0 relative"
              style={{ background: '#355EF1' }}
            >
              <Crown className="w-5 h-5 text-white" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h2 className="text-xl font-semibold text-gray-900">Memory Pro 服务</h2>
                <span className="px-2 py-0.5 bg-purple-100 text-purple-600 text-xs font-medium rounded-full flex items-center gap-1">
                  <Loader2 className="w-3 h-3 animate-spin" />
                  开通中
                </span>
              </div>
              <p className="text-sm text-gray-500 mt-1">
                正在开通服务，请稍候...
              </p>
            </div>
          </div>
        </div>

        {/* 居中的加载提示 */}
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center">
            <Loader2 className="w-10 h-10 text-purple-500 animate-spin mx-auto mb-4" />
            <p className="text-gray-600 text-sm">正在迁移数据并配置服务...</p>
            <p className="text-gray-400 text-xs mt-2">这可能需要几分钟时间</p>
          </div>
        </div>
      </div>
    );
  }

  // ══════════════════════════════════════════════════════════════════════════════
  // 未开通状态 - 显示宣传话术和状态
  // ══════════════════════════════════════════════════════════════════════════════
  if (memoryStatus === 'none') {
    const featureItems = [
      {
        icon: memoryStableIcon,
        title: '自动沉淀偏好',
        description: '从对话中提取偏好、约束与任务状态，减少重复说明',
      },
      {
        icon: deeperUnderstandingIcon,
        title: '形成用户画像',
        description: '按基本信息、工作习惯和沟通风格组织长期记忆',
      },
      {
        icon: preciseRetrievalIcon,
        title: '按场景召回',
        description: '结合当前任务语境检索相关记忆，让回复更贴近需求',
      },
      {
        icon: crossSessionIcon,
        title: '跨会话延续',
        description: '不同聊天通道共享记忆，不因上下文压缩而丢失',
      },
    ];

    return (
      <div className="w-full">
        {showHeader && (
          <div className="mb-6 flex items-center gap-3">
            <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-[4px] border border-[#DCE6FF] bg-[#F4F7FF]">
              <Database className="h-5 w-5 text-[#355EF1]" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="text-xl font-semibold text-[var(--text-title)]">Memory Pro 服务</h2>
                <span className="inline-flex h-6 items-center rounded-full bg-[#F5F5F5] px-2 text-xs font-medium text-[var(--text-muted)]">
                  已关闭
                </span>
              </div>
              <p className="mt-1 text-sm leading-[1.6] text-[var(--text-secondary)]">
                开启后，Agent 可在后续对话中延续偏好、习惯和历史上下文。
              </p>
            </div>
          </div>
        )}

        <TenantCard padding="none" className="px-6 py-10 sm:px-10">
          <Empty className="border-0 bg-transparent p-0 md:p-0">
            <EmptyHeader>
              <EmptyMedia variant="hint" />
              <EmptyTitle>
                为 Agent 开启长期记忆能力
              </EmptyTitle>
              <EmptyDescription className="max-w-none">
                <span className="inline-block whitespace-nowrap">Memory 服务会帮助 Agent 记住稳定偏好、</span>
                <span className="inline-block whitespace-nowrap">工作方式和历史对话线索，</span>
                <span className="inline-block whitespace-nowrap">当前实例暂未开通。</span>
              </EmptyDescription>
            </EmptyHeader>
          </Empty>

          <div className="mx-auto mt-8 grid w-full max-w-[966px] grid-cols-1 gap-4 md:grid-cols-2">
            {featureItems.map(({ icon, title, description }) => (
              <TenantCard key={title} padding="none" className="flex min-h-[92px] flex-row items-center gap-4 px-5 py-4">
                <img src={icon} alt="" aria-hidden="true" className="flex-shrink-0" />
                <div className="min-w-0 text-left">
                  <h4 className="mb-1 text-sm font-medium text-[var(--text-title)]">{title}</h4>
                  <p className="text-xs leading-[1.6] text-[var(--text-muted)]">{description}</p>
                </div>
              </TenantCard>
            ))}
          </div>

          <div className="mt-7 flex items-center justify-center gap-2 text-sm text-[var(--text-secondary)]">
            <Shield className="h-4 w-4 text-[var(--text-muted)]" />
            <span>请联系管理员在管控端开通 Memory 服务</span>
          </div>
        </TenantCard>
      </div>
    );
  }

  // ══════════════════════════════════════════════════════════════════════════════
  // 已开通 Free 版界面
  // ══════════════════════════════════════════════════════════════════════════════
  if (memoryStatus === 'free') {
    return (
      <div className="w-full h-full flex flex-col">
        {/* Free 版头部标题 */}
        <div className="mb-6">
          <div className="flex items-center gap-3">
            <div 
              className="w-10 h-10 rounded-[4px] flex items-center justify-center flex-shrink-0"
              style={{ background: '#355EF1' }}
            >
              <Zap className="w-5 h-5 text-white" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h2 className="text-xl font-semibold text-gray-900">Memory Free 服务</h2>
                <StatusTag mode="fill" variant="blue">已开启</StatusTag>
              </div>
              <p className="text-sm text-gray-500 mt-1">
                让 AI 智能体真正理解你、记住你，长期保持一致的工作习惯与决策偏好。
              </p>
            </div>
          </div>
        </div>

        {/* 记忆数据预览 - 加载中或已加载 */}
        {isLoading ? (
          <DataLoadingPlaceholder />
        ) : (
          <div className="flex items-start flex-1 min-h-0">
            <ProNavigation />
            <ProContent />
          </div>
        )}
      </div>
    );
  }

  // ══════════════════════════════════════════════════════════════════════════════
  // 已开通 Pro 版界面
  // ══════════════════════════════════════════════════════════════════════════════
  return (
    <div className="w-full h-full flex flex-col">
      {/* [本期新增] 免费体验期提示条 */}
      {renderFreeTrialBanner()}

      <div className="flex items-start flex-1 min-h-0">
        <ProNavigation />
        <ProContent />
      </div>
    </div>
  );
}

export default MemoryPreview;
