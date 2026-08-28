import { useState, useEffect, useMemo, useCallback, lazy, Suspense } from 'react';
import TenantLayout from '@/components/TenantLayout';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Tabs, TabsContent } from '@/components/ui/tabs';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { TenantCard } from '@/components/ui/Surface';
import { TenantSegmentGroup, TenantSegmentOption, TenantSegmentList, TenantSegmentItem } from '@/components/ui/segment';
import { LineTabs } from '@/components/ui/line-tabs';
import { CodeText, CardTitle, MetaText, CompactText } from '@/components/ui/Typography';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table';
import {
  GuideNewTag,
  isNewTagExpired,
  trackOnboarding,
} from '@/components/onboarding';
import {
  TENANT_SKILL_SQUARE_PUBLIC_ANALYTICS,
  TENANT_SKILL_SQUARE_PUBLIC_VISITED_EVENT,
  TENANT_SKILL_SQUARE_PUBLIC_UPDATE,
  isTenantSkillSquarePublicNewTagActive,
  markTenantSkillSquarePublicVisited,
} from '@/lib/tenantSkillSquarePublicOnboarding';
import {
  Empty,
  EmptyHeader,
  EmptyTitle,
  EmptyDescription,
  EmptyMedia,
} from '@/components/ui/empty';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

// 格式化下载量：>=1000 显示 x.xk，否则原样
function formatDownloadCount(count: number): string {
  if (count >= 1000) {
    return `${(count / 1000).toFixed(1)}k`;
  }
  return String(count);
}

// 获取名称首字符：英文返回大写字母，中文统一返回 'A'
function getSkillInitial(name: string): string | null {
  if (!name) return null;
  const firstChar = name.charAt(0);
  // 英文字母
  if (/[a-zA-Z]/.test(firstChar)) {
    return firstChar.toUpperCase();
  }
  // 中文标题统一用 A 替代
  return 'A';
}

// 为不同字母分配渐变色，色系对齐 Agent 头像风格（蓝紫、靛蓝、青色系柔和渐变）
const LETTER_COLORS: Record<string, { bg: string; text: string }> = {
  A: { bg: "#E8F4FD", text: "#1A73E8" },
  B: { bg: "#F3E8FD", text: "#8B5CF6" },
  C: { bg: "#E8FDF0", text: "#16A34A" },
  D: { bg: "#FDF2E8", text: "#EA580C" },
  E: { bg: "#FDE8F0", text: "#DC2626" },
  F: { bg: "#FDE8F0", text: "#DC2626" },
  G: { bg: "#E8FDF0", text: "#16A34A" },
  H: { bg: "#E8F4FD", text: "#1A73E8" },
  I: { bg: "#F3E8FD", text: "#8B5CF6" },
  J: { bg: "#FDF2E8", text: "#EA580C" },
  K: { bg: "#E8FDF0", text: "#16A34A" },
  L: { bg: "#E8F4FD", text: "#1A73E8" },
  M: { bg: "#F3E8FD", text: "#8B5CF6" },
  N: { bg: "#FDE8F0", text: "#DC2626" },
  O: { bg: "#FDF2E8", text: "#EA580C" },
  P: { bg: "#E8FDF0", text: "#16A34A" },
  Q: { bg: "#E8F4FD", text: "#1A73E8" },
  R: { bg: "#F3E8FD", text: "#8B5CF6" },
  S: { bg: "#E8F4FD", text: "#1A73E8" },
  T: { bg: "#F3E8FD", text: "#8B5CF6" },
  U: { bg: "#E8FDF0", text: "#16A34A" },
  V: { bg: "#FDF2E8", text: "#EA580C" },
  W: { bg: "#FDE8F0", text: "#DC2626" },
  X: { bg: "#E8F4FD", text: "#1A73E8" },
  Y: { bg: "#F3E8FD", text: "#8B5CF6" },
  Z: { bg: "#E8FDF0", text: "#16A34A" },
};
function getLetterColor(letter: string): { bg: string; text: string } {
  return LETTER_COLORS[letter.toUpperCase()] || { bg: "#E8F4FD", text: "#1A73E8" };
}

import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  ArrowLeft,
  ChevronDown,
  ChevronRight,
  Folder,
  FolderOpen,
  FileText,
  Code,
  Eye,
  Loader,
  CheckCircle,
  XCircle,
  Circle,
  Info,
  Trash2,
} from 'lucide-react';

// ─── 自定义图标（设计稿：技能广场） ─────────────────────────────────────────

const Search = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M14.5306 13.4693L11.5625 10.4999C12.4524 9.34021 12.8679 7.88541 12.7247 6.43063C12.5814 4.97585 11.8902 3.63002 10.7912 2.66616C9.69212 1.7023 8.2676 1.19257 6.80657 1.24039C5.34554 1.2882 3.95739 1.88998 2.92373 2.92364C1.89007 3.9573 1.2883 5.34544 1.24048 6.80648C1.19266 8.26751 1.70239 9.69203 2.66625 10.7911C3.63011 11.8901 4.97594 12.5814 6.43072 12.7246C7.8855 12.8678 9.3403 12.4524 10.5 11.5624L13.4706 14.5337C13.5404 14.6034 13.6232 14.6588 13.7144 14.6965C13.8055 14.7343 13.9032 14.7537 14.0019 14.7537C14.1005 14.7537 14.1982 14.7343 14.2894 14.6965C14.3805 14.6588 14.4634 14.6034 14.5331 14.5337C14.6029 14.4639 14.6582 14.3811 14.696 14.2899C14.7337 14.1988 14.7532 14.1011 14.7532 14.0024C14.7532 13.9038 14.7337 13.8061 14.696 13.7149C14.6582 13.6238 14.6029 13.5409 14.5331 13.4712L14.5306 13.4693ZM2.75001 6.99991C2.75001 6.15934 2.99926 5.33765 3.46626 4.63874C3.93326 3.93983 4.59702 3.3951 5.3736 3.07343C6.15019 2.75175 7.00472 2.66759 7.82914 2.83158C8.65356 2.99556 9.41084 3.40034 10.0052 3.99471C10.5996 4.58908 11.0044 5.34636 11.1683 6.17078C11.3323 6.9952 11.2482 7.84973 10.9265 8.62632C10.6048 9.40291 10.0601 10.0667 9.36118 10.5337C8.66227 11.0007 7.84058 11.2499 7.00001 11.2499C5.87319 11.2488 4.79286 10.8006 3.99608 10.0038C3.1993 9.20706 2.75116 8.12673 2.75001 6.99991Z" fill="currentColor"/>
  </svg>
);

const RefreshCw = ({ className }: { className?: string }) => (
  <svg width="18" height="18" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M14.25 3.00001V6.00001C14.25 6.19893 14.171 6.38969 14.0303 6.53034C13.8897 6.671 13.6989 6.75001 13.5 6.75001H10.5C10.3011 6.75001 10.1103 6.671 9.96967 6.53034C9.82902 6.38969 9.75 6.19893 9.75 6.00001C9.75 5.8011 9.82902 5.61034 9.96967 5.46968C10.1103 5.32903 10.3011 5.25001 10.5 5.25001H11.6875L11.2 4.76251C10.3174 3.87553 9.11879 3.37514 7.8675 3.37126H7.84062C6.6005 3.36855 5.40917 3.8542 4.52437 4.72314C4.38215 4.86221 4.19051 4.93909 3.9916 4.93686C3.7927 4.93464 3.60282 4.85349 3.46375 4.71126C3.32468 4.56904 3.2478 4.3774 3.25002 4.17849C3.25225 3.97959 3.3334 3.78971 3.47563 3.65064C4.64076 2.50655 6.20957 1.8673 7.8425 1.87126H7.875C9.52234 1.87576 11.1005 2.53421 12.2625 3.70189L12.75 4.18751V3.00001C12.75 2.8011 12.829 2.61034 12.9697 2.46968C13.1103 2.32903 13.3011 2.25001 13.5 2.25001C13.6989 2.25001 13.8897 2.32903 14.0303 2.46968C14.171 2.61034 14.25 2.8011 14.25 3.00001ZM11.4756 11.2769C10.5904 12.1463 9.39827 12.632 8.1575 12.6288H8.13062C6.87933 12.6249 5.68074 12.1245 4.79812 11.2375L4.3125 10.75H5.5C5.69891 10.75 5.88968 10.671 6.03033 10.5303C6.17098 10.3897 6.25 10.1989 6.25 10C6.25 9.8011 6.17098 9.61034 6.03033 9.46969C5.88968 9.32903 5.69891 9.25001 5.5 9.25001H2.5C2.30109 9.25001 2.11032 9.32903 1.96967 9.46969C1.82902 9.61034 1.75 9.8011 1.75 10V13C1.75 13.1989 1.82902 13.3897 1.96967 13.5303C2.11032 13.671 2.30109 13.75 2.5 13.75C2.69891 13.75 2.88968 13.671 3.03033 13.5303C3.17098 13.3897 3.25 13.1989 3.25 13V11.8125L3.7375 12.3C4.89983 13.467 6.47793 14.1248 8.125 14.1288H8.16C9.79293 14.1327 11.3617 13.4935 12.5269 12.3494C12.5973 12.2805 12.6535 12.1985 12.6922 12.1079C12.7309 12.0173 12.7514 11.92 12.7525 11.8215C12.7536 11.723 12.7353 11.6253 12.6986 11.5339C12.6619 11.4425 12.6076 11.3592 12.5387 11.2888C12.4699 11.2183 12.3878 11.1622 12.2973 11.1235C12.2067 11.0848 12.1094 11.0643 12.0109 11.0632C11.9124 11.0621 11.8147 11.0804 11.7233 11.117C11.6318 11.1537 11.5485 11.208 11.4781 11.2769H11.4756Z" fill="currentColor"/>
  </svg>
);

const LayoutGrid = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M6.5 2.5H3.5C3.16848 2.5 2.85054 2.6317 2.61612 2.86612C2.3817 3.10054 2.25 3.41848 2.25 3.75V6.5C2.25 6.83152 2.3817 7.14946 2.61612 7.38388C2.85054 7.6183 3.16848 7.75 3.5 7.75H6.5C6.83152 7.75 7.14946 7.6183 7.38388 7.38388C7.6183 7.14946 7.75 6.83152 7.75 6.5V3.75C7.75 3.41848 7.6183 3.10054 7.38388 2.86612C7.14946 2.6317 6.83152 2.5 6.5 2.5ZM6.25 6.25H3.75V4H6.25V6.25ZM12.5 2.5H9.75C9.41848 2.5 9.10054 2.6317 8.86612 2.86612C8.6317 3.10054 8.5 3.41848 8.5 3.75V6.5C8.5 6.83152 8.6317 7.14946 8.86612 7.38388C9.10054 7.6183 9.41848 7.75 9.75 7.75H12.5C12.8315 7.75 13.1495 7.6183 13.3839 7.38388C13.6183 7.14946 13.75 6.83152 13.75 6.5V3.75C13.75 3.41848 13.6183 3.10054 13.3839 2.86612C13.1495 2.6317 12.8315 2.5 12.5 2.5ZM12.25 6.25H10V4H12.25V6.25ZM6.5 8.5H3.5C3.16848 8.5 2.85054 8.6317 2.61612 8.86612C2.3817 9.10054 2.25 9.41848 2.25 9.75V12.5C2.25 12.8315 2.3817 13.1495 2.61612 13.3839C2.85054 13.6183 3.16848 13.75 3.5 13.75H6.5C6.83152 13.75 7.14946 13.6183 7.38388 13.3839C7.6183 13.1495 7.75 12.8315 7.75 12.5V9.75C7.75 9.41848 7.6183 9.10054 7.38388 8.86612C7.14946 8.6317 6.83152 8.5 6.5 8.5ZM6.25 12.25H3.75V10H6.25V12.25ZM12.5 8.5H9.75C9.41848 8.5 9.10054 8.6317 8.86612 8.86612C8.6317 9.10054 8.5 9.41848 8.5 9.75V12.5C8.5 12.8315 8.6317 13.1495 8.86612 13.3839C9.10054 13.6183 9.41848 13.75 9.75 13.75H12.5C12.8315 13.75 13.1495 13.6183 13.3839 13.3839C13.6183 13.1495 13.75 12.8315 13.75 12.5V9.75C13.75 9.41848 13.6183 9.10054 13.3839 8.86612C13.1495 8.6317 12.8315 8.5 12.5 8.5ZM12.25 12.25H10V10H12.25V12.25Z" fill="currentColor"/>
  </svg>
);

const List = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M4.75 4C4.75 3.80109 4.82902 3.61032 4.96967 3.46967C5.11032 3.32902 5.30109 3.25 5.5 3.25H13.5C13.6989 3.25 13.8897 3.32902 14.0303 3.46967C14.171 3.61032 14.25 3.80109 14.25 4C14.25 4.19891 14.171 4.38968 14.0303 4.53033C13.8897 4.67098 13.6989 4.75 13.5 4.75H5.5C5.30109 4.75 5.11032 4.67098 4.96967 4.53033C4.82902 4.38968 4.75 4.19891 4.75 4ZM13.5 7.25H5.5C5.30109 7.25 5.11032 7.32902 4.96967 7.46967C4.82902 7.61032 4.75 7.80109 4.75 8C4.75 8.19891 4.82902 8.38968 4.96967 8.53033C5.11032 8.67098 5.30109 8.75 5.5 8.75H13.5C13.6989 8.75 13.8897 8.67098 14.0303 8.53033C14.171 8.38968 14.25 8.19891 14.25 8C14.25 7.80109 14.171 7.61032 14.0303 7.46967C13.8897 7.32902 13.6989 7.25 13.5 7.25ZM13.5 11.25H5.5C5.30109 11.25 5.11032 11.329 4.96967 11.4697C4.82902 11.6103 4.75 11.8011 4.75 12C4.75 12.1989 4.82902 12.3897 4.96967 12.5303C5.11032 12.671 5.30109 12.75 5.5 12.75H13.5C13.6989 12.75 13.8897 12.671 14.0303 12.5303C14.171 12.3897 14.25 12.1989 14.25 12C14.25 11.8011 14.171 11.6103 14.0303 11.4697C13.8897 11.329 13.6989 11.25 13.5 11.25ZM2.75 7C2.55222 7 2.35888 7.05865 2.19443 7.16853C2.02998 7.27841 1.90181 7.43459 1.82612 7.61732C1.75043 7.80004 1.73063 8.00111 1.76922 8.19509C1.8078 8.38907 1.90304 8.56725 2.04289 8.70711C2.18275 8.84696 2.36093 8.9422 2.55491 8.98079C2.74889 9.01937 2.94996 8.99957 3.13268 8.92388C3.31541 8.84819 3.47159 8.72002 3.58147 8.55557C3.69135 8.39112 3.75 8.19778 3.75 8C3.75 7.73478 3.64464 7.48043 3.45711 7.29289C3.26957 7.10536 3.01522 7 2.75 7ZM2.75 3C2.55222 3 2.35888 3.05865 2.19443 3.16853C2.02998 3.27841 1.90181 3.43459 1.82612 3.61732C1.75043 3.80004 1.73063 4.00111 1.76922 4.19509C1.8078 4.38907 1.90304 4.56725 2.04289 4.70711C2.18275 4.84696 2.36093 4.9422 2.55491 4.98079C2.74889 5.01937 2.94996 4.99957 3.13268 4.92388C3.31541 4.84819 3.47159 4.72002 3.58147 4.55557C3.69135 4.39112 3.75 4.19778 3.75 4C3.75 3.73478 3.64464 3.48043 3.45711 3.29289C3.26957 3.10536 3.01522 3 2.75 3ZM2.75 11C2.55222 11 2.35888 11.0586 2.19443 11.1685C2.02998 11.2784 1.90181 11.4346 1.82612 11.6173C1.75043 11.8 1.73063 12.0011 1.76922 12.1951C1.8078 12.3891 1.90304 12.5673 2.04289 12.7071C2.18275 12.847 2.36093 12.9422 2.55491 12.9808C2.74889 13.0194 2.94996 12.9996 3.13268 12.9239C3.31541 12.8482 3.47159 12.72 3.58147 12.5556C3.69135 12.3911 3.75 12.1978 3.75 12C3.75 11.7348 3.64464 11.4804 3.45711 11.2929C3.26957 11.1054 3.01522 11 2.75 11Z" fill="currentColor"/>
  </svg>
);

const Download = ({ className }: { className?: string }) => (
  <svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M10.6875 6.75V9.75C10.6875 9.89918 10.6282 10.0423 10.5227 10.1477C10.4173 10.2532 10.2742 10.3125 10.125 10.3125H1.875C1.72582 10.3125 1.58274 10.2532 1.47725 10.1477C1.37176 10.0423 1.3125 9.89918 1.3125 9.75V6.75C1.3125 6.60082 1.37176 6.45774 1.47725 6.35225C1.58274 6.24676 1.72582 6.1875 1.875 6.1875C2.02418 6.1875 2.16726 6.24676 2.27275 6.35225C2.37824 6.45774 2.4375 6.60082 2.4375 6.75V9.1875H9.5625V6.75C9.5625 6.60082 9.62176 6.45774 9.72725 6.35225C9.83274 6.24676 9.97582 6.1875 10.125 6.1875C10.2742 6.1875 10.4173 6.24676 10.5227 6.35225C10.6282 6.45774 10.6875 6.60082 10.6875 6.75ZM5.60203 7.14797C5.65429 7.20041 5.71639 7.24202 5.78476 7.27041C5.85313 7.2988 5.92644 7.31341 6.00047 7.31341C6.0745 7.31341 6.14781 7.2988 6.21618 7.27041C6.28455 7.24202 6.34665 7.20041 6.39891 7.14797L8.27391 5.27297C8.37958 5.1673 8.43894 5.02397 8.43894 4.87453C8.43894 4.72509 8.37958 4.58177 8.27391 4.47609C8.16823 4.37042 8.02491 4.31106 7.87547 4.31106C7.72603 4.31106 7.5827 4.37042 7.47703 4.47609L6.5625 5.39062V1.5C6.5625 1.35082 6.50324 1.20774 6.39775 1.10225C6.29226 0.996763 6.14918 0.9375 6 0.9375C5.85082 0.9375 5.70774 0.996763 5.60225 1.10225C5.49676 1.20774 5.4375 1.35082 5.4375 1.5V5.39062L4.52297 4.47703C4.47065 4.42471 4.40853 4.3832 4.34016 4.35489C4.2718 4.32657 4.19853 4.31199 4.12453 4.31199C3.97509 4.31199 3.83177 4.37136 3.72609 4.47703C3.67377 4.52935 3.63226 4.59147 3.60395 4.65984C3.57563 4.7282 3.56106 4.80147 3.56106 4.87547C3.56106 5.02491 3.62042 5.16823 3.72609 5.27391L5.60203 7.14797Z" fill="currentColor"/>
  </svg>
);

const Plus = ({ className }: { className?: string }) => (
  <svg width="18" height="18" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M8 4.5V11.5M4.5 8H11.5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"/>
  </svg>
);
import { toast } from 'sonner';

// 复用管控端数据和组件
import { MOCK_SKILLS, DEFAULT_CATEGORIES, MOCK_OPENCLAW_INSTANCES } from '../admin/SkillLibrary/mockData';
import { type Skill, type DistributionStatus, DISTRIBUTION_STATUS_MAP } from '../admin/SkillLibrary/types';
import {
  getDistributionRecords,
  addDistributionRecord,
  updateDistributionRecord,
  createDistributionRecordId,
  initMockDistributionRecords,
  type CachedDistributionRecord,
  type RecordType,
} from '../admin/SkillLibrary/distributionCache';
import { downloadSkillAsZip } from '../admin/SkillLibrary/downloadUtils';
import BatchDistributeDialog from '../admin/SkillLibrary/BatchDistributeDialog';
import BatchDeleteDialog from '../admin/SkillLibrary/BatchDeleteDialog';
import MDXRenderer from '@/components/MDXRenderer';
import {
  PUBLIC_SKILLS,
  PUBLIC_SKILL_CATEGORIES,
  type PublicSkill,
  type PublicSkillFile,
} from '../admin/SkillLibrary/publicSkillMockData';

// ── 用户端发布 / 我的申请 / 申请下架（复用管控端发布弹窗 + tenant 私有辅件） ──
import SkillUploadDialog from '../admin/SkillLibrary/SkillUploadDialog';
import MyRequestsSheet from './SkillSquare/MyRequestsSheet';
import OffshelfSkillDialog from './SkillSquare/OffshelfSkillDialog';
import {
  addPublishRequest,
  hasPendingOffshelfRequest,
  useMyRequestCount,
} from './SkillSquare/myRequestsStore';
import { ClipboardList, Upload } from 'lucide-react';

// 懒加载 react-syntax-highlighter
const SyntaxHighlighter = lazy(() =>
  import('react-syntax-highlighter').then(mod => ({ default: mod.Light as any }))
) as unknown as React.ComponentType<any>;
const loadedLanguages = new Set<string>();
const registerLanguage = async (lang: string) => {
  if (loadedLanguages.has(lang)) return;
  loadedLanguages.add(lang);
  try {
    const mod = await import('react-syntax-highlighter');
    const Light = mod.Light as any;
    const langModules: Record<string, () => Promise<any>> = {
      xml: () => import('react-syntax-highlighter/dist/esm/languages/hljs/xml'),
      json: () => import('react-syntax-highlighter/dist/esm/languages/hljs/json'),
      yaml: () => import('react-syntax-highlighter/dist/esm/languages/hljs/yaml'),
      python: () => import('react-syntax-highlighter/dist/esm/languages/hljs/python'),
      javascript: () => import('react-syntax-highlighter/dist/esm/languages/hljs/javascript'),
      typescript: () => import('react-syntax-highlighter/dist/esm/languages/hljs/typescript'),
      bash: () => import('react-syntax-highlighter/dist/esm/languages/hljs/bash'),
      css: () => import('react-syntax-highlighter/dist/esm/languages/hljs/css'),
      ini: () => import('react-syntax-highlighter/dist/esm/languages/hljs/ini'),
      markdown: () => import('react-syntax-highlighter/dist/esm/languages/hljs/markdown'),
    };
    const loader = langModules[lang];
    if (loader) {
      const langMod = await loader();
      Light.registerLanguage(lang, langMod.default);
    }
  } catch { /* 静默降级 */ }
};

// hljs 亮色主题
const hljsStyle: Record<string, React.CSSProperties> = {
  'hljs': { display: 'block', overflowX: 'auto', padding: '1em', background: '#ffffff', color: '#383a42' },
  'hljs-comment': { color: '#a0a1a7', fontStyle: 'italic' },
  'hljs-quote': { color: '#a0a1a7', fontStyle: 'italic' },
  'hljs-keyword': { color: '#a626a4' },
  'hljs-selector-tag': { color: '#a626a4' },
  'hljs-addition': { color: '#50a14f' },
  'hljs-number': { color: '#986801' },
  'hljs-string': { color: '#50a14f' },
  'hljs-meta': { color: '#4078f2' },
  'hljs-literal': { color: '#0184bb' },
  'hljs-doctag': { color: '#a626a4' },
  'hljs-regexp': { color: '#50a14f' },
  'hljs-attr': { color: '#986801' },
  'hljs-attribute': { color: '#50a14f' },
  'hljs-builtin-name': { color: '#e45649' },
  'hljs-name': { color: '#e45649' },
  'hljs-section': { color: '#e45649' },
  'hljs-tag': { color: '#e45649' },
  'hljs-variable': { color: '#e45649' },
  'hljs-template-variable': { color: '#e45649' },
  'hljs-selector-id': { color: '#e45649' },
  'hljs-title': { color: '#4078f2' },
  'hljs-type': { color: '#4078f2' },
  'hljs-symbol': { color: '#4078f2' },
  'hljs-bullet': { color: '#4078f2' },
  'hljs-link': { color: '#4078f2' },
  'hljs-deletion': { color: '#e45649' },
  'hljs-emphasis': { fontStyle: 'italic' },
  'hljs-strong': { fontWeight: 'bold' },
};

// ========== Mock 用户身份（模拟当前用户所属组织） ==========
const CURRENT_USER_GROUP_IDS = ['grp-1', 'grp-2'];

// ========== Mock 下载量数据 ==========
const MOCK_DOWNLOAD_COUNTS: Record<string, number> = {
  'skill-0': 1286,
  'skill-1': 432,
  'skill-2': 867,
  'skill-3': 523,
  'skill-4': 198,
  'skill-5': 945,
  'skill-6': 312,
  'skill-7': 756,
  'skill-8': 89,
  'skill-9': 167,
};

// ========== 排序类型 ==========
type SortType = 'time' | 'downloads';
type ViewMode = 'card' | 'list';
type SkillLibraryTab = 'enterprise' | 'public';

const SKILL_LIBRARY_TABS = [
  { id: 'enterprise', label: '企业技能' },
  { id: 'public', label: '公共技能', dataGuide: 'tenant-public-skill-tab' },
] as const;

const PUBLIC_CATEGORY_LABELS: Record<string, string> = {
  'general-office': '通用办公',
  ops: '系统运维',
};

function flattenPublicSkillFiles(files: PublicSkillFile[]): Array<{ name: string; size: number; content?: string }> {
  const result: Array<{ name: string; size: number; content?: string }> = [];

  const walk = (items: PublicSkillFile[]) => {
    items.forEach((file) => {
      if (file.type === 'folder' && file.children?.length) {
        walk(file.children);
        return;
      }
      result.push({
        name: file.path || file.name,
        size: file.content?.length || 0,
        content: file.content,
      });
    });
  };

  walk(files);
  return result;
}

function toSkillFromPublicSkill(skill: PublicSkill): Skill {
  const latestVersion = skill.versions.find(version => version.isLatest) || skill.versions[0];

  return {
    id: skill.id,
    slug: skill.slug,
    name: skill.nameZh || skill.name,
    description: skill.descriptionZh || skill.description,
    version: skill.version,
    categories: [skill.category, ...skill.tags],
    scope: 'public',
    groupIds: [],
    uploadTime: latestVersion?.date ? new Date(latestVersion.date) : new Date(),
    content: skill.files.find(file => file.name.toLowerCase() === 'skill.md')?.content,
    versions: skill.versions.map(version => version.version),
    files: flattenPublicSkillFiles(skill.files),
    versionHistory: skill.versions.map(version => ({
      version: version.version,
      date: version.date,
      files: flattenPublicSkillFiles(skill.files),
    })),
  };
}

// ========== 过滤可见技能（应用范围过滤） ==========
function getVisibleSkills(skills: Skill[], userGroupIds: string[]): Skill[] {
  return skills.filter(skill => {
    if (skill.scope === 'public') return true;
    // scope=private: 用户组织与技能组织有交集才可见
    if (skill.scope === 'private' && skill.groupIds.length > 0) {
      return skill.groupIds.some(gId => userGroupIds.includes(gId));
    }
    return false;
  });
}

export default function SkillSquare() {
  // 初始化预设下发记录（仅首次、localStorage为空时生效）
  useEffect(() => { initMockDistributionRecords(); }, []);

  // ========== 列表状态 ==========
  const [activeLibraryTab, setActiveLibraryTab] = useState<SkillLibraryTab>('enterprise');
  const [showPublicSkillNewTag, setShowPublicSkillNewTag] = useState(false);
  const [searchQueries, setSearchQueries] = useState<Record<SkillLibraryTab, string>>({
    enterprise: '',
    public: '',
  });
  const [viewMode, setViewMode] = useState<ViewMode>('card');
  const [sortType, setSortType] = useState<SortType>('time');
  const [selectedCategories, setSelectedCategories] = useState<Record<SkillLibraryTab, string>>({
    enterprise: 'all',
    public: 'all',
  });
  const [skills] = useState<Skill[]>(MOCK_SKILLS);

  // ========== 详情状态 ==========
  const [selectedSkillId, setSelectedSkillId] = useState<string | null>(null);
  const [initialTab, setInitialTab] = useState<string>('overview');

  // ========== 发布 / 我的申请 / 申请下架 状态 ==========
  // OWNED_SKILL_IDS：当前员工"我上传的" skill id 集合（Demo 阶段用 mock 数据）；
  // 真实接入时应由后端返回本人上传的 skill 列表，此处仅演示"申请下架"入口的显现规则。
  const OWNED_SKILL_IDS = useMemo(() => new Set(['skill-0', 'skill-1']), []);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [myRequestsOpen, setMyRequestsOpen] = useState(false);
  /**
   * 下架申请目标：兼容两条触发路径 —— 卡片入口 & 「我的申请」里点已发布记录
   * 的下架按钮。后者的 skillName 不一定对应 MOCK_SKILLS，因此 skillId 为可选。
   */
  const [offshelfTarget, setOffshelfTarget] = useState<
    { skillId?: string; skillName: string; version?: string } | null
  >(null);
  const myRequestCount = useMyRequestCount();

  useEffect(() => {
    setShowPublicSkillNewTag(
      isTenantSkillSquarePublicNewTagActive() &&
      !isNewTagExpired(
        TENANT_SKILL_SQUARE_PUBLIC_UPDATE.releaseDate,
        TENANT_SKILL_SQUARE_PUBLIC_UPDATE.newTagVisibleDays,
      ),
    );
  }, []);

  const handleLibraryTabChange = (tab: SkillLibraryTab) => {
    setActiveLibraryTab(tab);
    setInitialTab('overview');

    if (tab === 'public') {
      markTenantSkillSquarePublicVisited();
      window.dispatchEvent(new Event(TENANT_SKILL_SQUARE_PUBLIC_VISITED_EVENT));

      if (showPublicSkillNewTag) {
        trackOnboarding('onboarding_click', {
          ...TENANT_SKILL_SQUARE_PUBLIC_ANALYTICS,
          component: 'new_tag',
          target: 'public-skill-tab',
        });
      }
    }
  };

  const skillLibraryTabs = useMemo(
    () => SKILL_LIBRARY_TABS.map(tab => (
      tab.id === 'public' && showPublicSkillNewTag
        ? { ...tab, suffix: <GuideNewTag className="ml-1" /> }
        : tab
    )),
    [showPublicSkillNewTag]
  );

  // 过滤可见技能
  const visibleSkills = useMemo(
    () => getVisibleSkills(skills, CURRENT_USER_GROUP_IDS),
    [skills]
  );

  const publicSkills = useMemo(() => PUBLIC_SKILLS.map(toSkillFromPublicSkill), []);
  const activeSkills = activeLibraryTab === 'enterprise' ? visibleSkills : publicSkills;
  const detailSkills = useMemo(() => [...visibleSkills, ...publicSkills], [visibleSkills, publicSkills]);
  const searchQuery = searchQueries[activeLibraryTab];
  const selectedCategory = selectedCategories[activeLibraryTab];
  const activeDescription = activeLibraryTab === 'enterprise'
    ? '一键选装企业内的优质技能。'
    : '浏览公共技能库中的精选技能，按需安装技能。';
  const activeCategoryOptions = useMemo(() => {
    if (activeLibraryTab === 'enterprise') {
      return DEFAULT_CATEGORIES.map(cat => ({ id: cat.id, name: cat.name }));
    }

    const usedCategories = new Set(PUBLIC_SKILLS.map(skill => skill.category));
    const configuredCategories = PUBLIC_SKILL_CATEGORIES
      .filter(cat => cat.id !== 'all' && usedCategories.has(cat.id))
      .map(cat => ({ id: cat.id, name: PUBLIC_CATEGORY_LABELS[cat.id] || cat.name }));
    const configuredIds = new Set(configuredCategories.map(cat => cat.id));
    const fallbackCategories = Array.from(usedCategories)
      .filter(id => !configuredIds.has(id))
      .map(id => ({ id, name: PUBLIC_CATEGORY_LABELS[id] || id }));

    return [...configuredCategories, ...fallbackCategories];
  }, [activeLibraryTab, publicSkills]);

  const downloadCounts = useMemo(() => {
    const publicCounts = Object.fromEntries(PUBLIC_SKILLS.map(skill => [skill.id, skill.downloads]));
    return { ...MOCK_DOWNLOAD_COUNTS, ...publicCounts };
  }, []);

  const setSearchQuery = (value: string) => {
    setSearchQueries(prev => ({ ...prev, [activeLibraryTab]: value }));
  };

  const setSelectedCategory = (value: string) => {
    setSelectedCategories(prev => ({ ...prev, [activeLibraryTab]: value }));
  };

  // 搜索 + 分类筛选
  const filteredSkills = useMemo(() => {
    let result = activeSkills;

    // 搜索
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      result = result.filter(
        s => s.name.toLowerCase().includes(q) || s.description.toLowerCase().includes(q)
      );
    }

    // 分类筛选
    if (selectedCategory !== 'all') {
      result = result.filter(s => s.categories.includes(selectedCategory));
    }

    // 排序
    if (sortType === 'time') {
      result = [...result].sort(
        (a, b) => new Date(b.uploadTime).getTime() - new Date(a.uploadTime).getTime()
      );
    } else {
      result = [...result].sort(
        (a, b) => (downloadCounts[b.id] || 0) - (downloadCounts[a.id] || 0)
      );
    }

    return result;
  }, [activeSkills, searchQuery, selectedCategory, sortType, downloadCounts]);

  // 刷新
  const [isRefreshing, setIsRefreshing] = useState(false);
  const handleRefresh = () => {
    setIsRefreshing(true);
    setTimeout(() => {
      setIsRefreshing(false);
      toast.success('列表已刷新');
    }, 1000);
  };

  // 如果选中了技能，渲染详情页
  if (selectedSkillId) {
    return (
      <TenantLayout>
        {/* 用户端单层 120px 骨架（SKILL-TENANT §6.1.1） */}
        <div className="min-w-[1200px]">
          <div className="max-w-[1920px] mx-auto page-enter">
            <div
              className="relative min-h-[calc(100vh-64px)] flex flex-col"
              style={{ paddingLeft: 120, paddingRight: 120, paddingBottom: 75 }}
            >
              <SkillSquareDetail
                  skillId={selectedSkillId}
                  skills={detailSkills}
                  onBack={() => { setSelectedSkillId(null); setInitialTab('overview'); }}
                  initialTab={initialTab}
                />
            </div>
          </div>
        </div>
      </TenantLayout>
    );
  }

  return (
    <TenantLayout>
      {/* 用户端单层 120px 骨架（SKILL-TENANT §6.1.1） */}
      <div className="min-w-[1200px]">
        <div className="max-w-[1920px] mx-auto page-enter">
          <div
            className="relative min-h-[calc(100vh-64px)]"
            style={{ paddingLeft: 120, paddingRight: 120, paddingBottom: 75 }}
          >
            {/* Hero 段 — 页面标题「技能广场」（对齐「模型额度」等同构页面的 <h1> 用法），
                LineTabs 仅用作标题下方一级导航（规范见 SKILL-GLOBAL-COMPONENTS.md §11.5：
                仅限页面标题下方的一级 Tab，字号固定 14px Medium，不可挪作标题、不可覆盖字号） */}
            <div className="relative pt-6">
              {/* 标题行：左侧 h1「技能广场」；右侧仅在「企业技能」Tab 显示「我的申请 / 发布 Skill」两按钮。
                  规则说明：
                  - 「发布 Skill」复用管控端 SkillUploadDialog，onConfirm 走 addPublishRequest，
                    不写入企业技能列表（走待审核链路，落到"我的申请"）；
                  - 「我的申请」右侧带待处理条数徽标，点击打开右侧 Sheet；
                  - 「公共技能」Tab 场景下这两个按钮均隐藏（发布/下架仅对企业范围有效）。 */}
              <div className="flex items-start justify-between gap-4 mb-4">
                <h1 className="font-sans font-medium text-[26px] leading-[35.56px] tracking-[-0.0427em] m-0 text-[var(--text-emphasis)]">
                  技能广场
                </h1>
                {activeLibraryTab === 'enterprise' && (
                  <div className="flex items-center gap-2 flex-shrink-0">
                    <Button
                      variant="tenant-outline"
                      size="claw-sm"
                      onClick={() => setMyRequestsOpen(true)}
                      className="gap-1.5"
                    >
                      <ClipboardList className="w-4 h-4" />
                      <span>我的申请</span>
                      {myRequestCount > 0 && (
                        <span className="inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full text-[11px] font-medium leading-none bg-[var(--tenant-primary)] text-white tabular-nums">
                          {myRequestCount}
                        </span>
                      )}
                    </Button>
                    <Button
                      variant="tenant-primary"
                      size="claw-sm"
                      onClick={() => setUploadOpen(true)}
                      className="gap-1.5"
                    >
                      <Upload className="w-4 h-4" />
                      发布 Skill
                    </Button>
                  </div>
                )}
              </div>
              <LineTabs
                tabs={skillLibraryTabs}
                active={activeLibraryTab}
                onChange={handleLibraryTabChange}
                description={activeDescription}
              />
            </div>

            {/* 内容段（搜索栏 / 分类 / 卡片网格） */}
            <div className="relative h-auto pb-6">

        {/* 搜索栏 + 筛选 */}
        <div className="relative flex h-10 flex-wrap gap-2 mb-4 items-center">
          {/* 搜索框 — 加长 */}
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-muted)]"/>
            <Input
              tenant
              placeholder="搜索技能名称或描述..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>

          {/* 排序 */}
          <Select value={sortType} onValueChange={(v) => setSortType(v as SortType)}>
            <SelectTrigger tenant className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="time">发布时间</SelectItem>
              <SelectItem value="downloads">下载量</SelectItem>
            </SelectContent>
          </Select>

          {/* 刷新按钮 */}
          <Button
            variant="tenant-outline"
            size="icon"
            onClick={handleRefresh}
            className="w-9 h-9"
          >
            <RefreshCw className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} />
          </Button>

          {/* 视图切换 — 统一 segment 样式（带图标+文字） */}
          <TenantSegmentGroup>
            <TenantSegmentOption active={viewMode === 'card'} onClick={() => setViewMode('card')}>
              <LayoutGrid className="w-4 h-4" />
              卡片视图
            </TenantSegmentOption>
            <TenantSegmentOption active={viewMode === 'list'} onClick={() => setViewMode('list')}>
              <List className="w-4 h-4" />
              列表视图
            </TenantSegmentOption>
          </TenantSegmentGroup>
        </div>

        {/* 分类横排按钮 */}
        <div className="flex items-center gap-2 mb-6 flex-wrap">
          <Button variant="tenant-plain" size="sm" data-state={selectedCategory === 'all' ? "active" : undefined} onClick={() => setSelectedCategory('all')}>
            全部
          </Button>
          {activeCategoryOptions.map(cat => (
            <Button key={cat.id} variant="tenant-plain" size="sm" data-state={selectedCategory === cat.id ? "active" : undefined} onClick={() => setSelectedCategory(cat.id)}>
              {cat.name}
            </Button>
          ))}
        </div>

        {/* 技能列表 */}
        {filteredSkills.length === 0 ? (
          <Empty className="border-0 py-24">
            <EmptyHeader>
              <EmptyMedia variant="default" />
              <EmptyTitle>暂无符合条件的技能</EmptyTitle>
              <EmptyDescription>
                {searchQuery || selectedCategory !== 'all'
                  ? '试试调整搜索关键词或切换分类'
                  : activeLibraryTab === 'public'
                    ? '公共技能上架后会在这里展示'
                    : '企业技能上架后会在这里展示'}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : viewMode === 'card' ? (
          /* 卡片视图：固定 3 列 */
          <div className="relative grid grid-cols-3 gap-4">
            {filteredSkills.map(skill => (
              <SkillCard
                key={skill.id}
                skill={skill}
                downloadCount={downloadCounts[skill.id] || 0}
                onClick={() => setSelectedSkillId(skill.id)}
                onDistStatusClick={() => { setInitialTab('distribution'); setSelectedSkillId(skill.id); }}
                isOwned={activeLibraryTab === 'enterprise' && OWNED_SKILL_IDS.has(skill.id)}
                hasPendingOffshelf={hasPendingOffshelfRequest(skill.id)}
                onRequestOffshelf={() =>
                  setOffshelfTarget({
                    skillId: skill.id,
                    skillName: skill.name,
                    version: skill.version,
                  })
                }
              />
            ))}
          </div>
        ) : (
          /* 列表视图 — 紧凑横排布局（外框对齐 AgentCard：12px 圆角 + 单层阴影） */
          <TenantCard padding="none" className="relative overflow-hidden">
            <div className="divide-y divide-[#F5F5F5]">
              {filteredSkills.map(skill => (
                <SkillListRow
                  key={skill.id}
                  skill={skill}
                  downloadCount={downloadCounts[skill.id] || 0}
                  onClick={() => setSelectedSkillId(skill.id)}
                  onDistStatusClick={() => { setInitialTab('distribution'); setSelectedSkillId(skill.id); }}
                />
              ))}
            </div>
          </TenantCard>
        )}
            </div>
            {/* /内容段 */}
          </div>
        </div>
      </div>

      {/* ============ 发布 Skill 弹窗 — 复用管控端 SkillUploadDialog ============
          说明：
          - onConfirm 收到管控端弹窗构造的 Skill 对象后，只调用 addPublishRequest
            将其加入"我的申请"（Mock 待审核链路），不写入 MOCK_SKILLS。
          - existingSlugs 传当前 MOCK_SKILLS 的 slug 列表，避免和已上架技能重名。
          - 不传 lockedScope：员工可选公共/分组范围，最终范围以管理员审核为准。
          - successMessage：员工端发布走待审核链路，覆盖弹窗默认的「技能发布成功」，
            改用「Skill 已提交，等待管理员审核」，避免误导用户以为已经上架。
          - hideDefaultSecuritySetting：「设置上传/更新时默认提交安全检测」是全局默认
            设置的开关，属于管控范畴，员工无权修改，因此隐藏。 */}
      <SkillUploadDialog
        open={uploadOpen}
        onOpenChange={setUploadOpen}
        existingSlugs={MOCK_SKILLS.map(s => s.slug)}
        successMessage="Skill 已提交，等待管理员审核"
        hideDefaultSecuritySetting
        onConfirm={(skill) => {
          addPublishRequest({
            skillName: skill.name,
            skillId: skill.id,
            version: skill.version,
          });
        }}
      />

      {/* ============ 我的申请 Sheet ============
          - onCardClick：published / rejected / withdrawn 三态可点击，统一打开
            发布弹窗；published 语义上是"更新"（管控端弹窗本身就承载了更新能力，
            用户重新上传即触发版本迭代），rejected / withdrawn 语义上是"重新提交"，
            Mock 阶段两条路径共用同一个弹窗，最终以后端字段区分。
          - onOffshelfClick：已发布记录的"下架"按钮，等价于卡片入口触发的下架流程。 */}
      <MyRequestsSheet
        open={myRequestsOpen}
        onOpenChange={setMyRequestsOpen}
        onCardClick={() => {
          setMyRequestsOpen(false);
          setUploadOpen(true);
        }}
        onOffshelfClick={(req) => {
          setMyRequestsOpen(false);
          setOffshelfTarget({
            skillId: req.skillId,
            skillName: req.skillName,
            version: req.version,
          });
        }}
      />

      {/* ============ 申请下架 Dialog ============
          offshelfTarget 非空时打开，关闭时清空。兼容"技能广场卡片"与
          "我的申请 · 已发布记录下架"两条触发路径。 */}
      {offshelfTarget && (
        <OffshelfSkillDialog
          open={!!offshelfTarget}
          onOpenChange={(open) => { if (!open) setOffshelfTarget(null); }}
          skillId={offshelfTarget.skillId}
          skillName={offshelfTarget.skillName}
          version={offshelfTarget.version}
        />
      )}
    </TenantLayout>
  );
}

// ========== 卡片组件 ==========
function SkillCard({
  skill,
  downloadCount,
  onClick,
  onDistStatusClick,
  isOwned = false,
  hasPendingOffshelf = false,
  onRequestOffshelf,
}: {
  skill: Skill;
  downloadCount: number;
  onClick: () => void;
  onDistStatusClick?: () => void;
  /** 是否是"我上传的" skill（仅企业技能场景可能为 true） */
  isOwned?: boolean;
  /** 是否已有该 skill 的下架审核在进行中（存在则禁用二次申请） */
  hasPendingOffshelf?: boolean;
  /** 触发"申请下架"弹窗 */
  onRequestOffshelf?: () => void;
}) {
  const [distributeOpen, setDistributeOpen] = useState(false);

  // 获取该技能的最新下发状态
  const latestRecord = getDistributionRecords(skill.id)?.[0];
  const distStatus = latestRecord?.status;
  const isDistributing = distStatus === 'distributing' || distStatus === 'deleting';

  const handleDistributeClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (isDistributing) return; // 下发中禁止点击
    setDistributeOpen(true);
  };

  const handleDistributionStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: skill.id,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'distributing',
      instances: selectedInstancesData.map(inst => ({
        id: inst.id,
        name: inst.name,
        createdBy: inst.createdBy || 'admin',
        distributionStatus: 'distributing' as DistributionStatus,
      })),
    };
    addDistributionRecord(newRecord);
    setDistributeOpen(false);
    toast.success(`已开始下发「${skill.name}」到 ${selectedInstanceIds.length} 个实例`);
    simulateDistribution(recordId, selectedInstanceIds.length);
  };

  const initial = getSkillInitial(skill.name);

  return (
    <>
      <TenantCard
        interactive
        className="group relative cursor-pointer gap-4"
        onClick={onClick}
      >
        {/* ===== 头部行：图标(字母) + 文本块（标题行含下载量 / 版本号） ===== */}
        <div className="flex items-center gap-4">
          {initial && (
            <div
              className="flex items-center justify-center flex-shrink-0 rounded w-12 h-12"
              style={{ background: getLetterColor(initial).bg }}
            >
              <span className="font-bold text-sm" style={{ color: getLetterColor(initial).text }}>{initial}</span>
            </div>
          )}
          <div className="flex flex-col gap-1 min-w-0 flex-1">
            {/* 标题行：名称 + 状态图标 + 下载量（下载量纳入此行，与标题文字水平居中对齐） */}
            <div className="flex items-center gap-2 min-w-0">
              <CardTitle
                className="truncate transition-colors group-hover:text-[var(--text-brand)]"
                title={skill.name}
              >
                {skill.name}
              </CardTitle>
              {/* 下发状态图标 */}
              {distStatus && <DistributionStatusIcon status={distStatus} latestRecord={latestRecord} onClick={() => onDistStatusClick?.()} />}
              {/* 下载量：推到行尾，和标题同一行居中对齐 */}
              <div className="flex items-center gap-1 flex-shrink-0 ml-auto text-[var(--text-muted)]">
                <Download className="w-3 h-3" />
                <MetaText as="span" className="tabular-nums">{formatDownloadCount(downloadCount)}</MetaText>
              </div>
            </div>
            <MetaText>v{skill.version}</MetaText>
          </div>
        </div>

        {/* ===== 描述 — 始终占 3 行高度（line-clamp-3 + min-h），锁死底部行位置 ===== */}
        <CompactText as="p" className="line-clamp-3 min-h-[calc(1.5em*3)]">
          {skill.description}
        </CompactText>

        {/* ===== 底部行：发布时间 +（我上传的：申请下架/下架审核中）+ 下发按钮 ===== */}
        <div className="flex items-center justify-between gap-2">
          <MetaText className="truncate">{formatDate(skill.uploadTime)}</MetaText>
          {/* 「申请下架」入口 — 仅对"我上传的" skill 显示，已在下架审核中时禁用 */}
          {isOwned && (
            hasPendingOffshelf ? (
              <span
                className="ml-auto mr-1 text-[12px] leading-none text-[var(--text-weak)] cursor-not-allowed select-none"
                onClick={(e) => e.stopPropagation()}
              >
                下架审核中
              </span>
            ) : (
              <button
                type="button"
                className="ml-auto mr-1 text-[12px] leading-none text-[var(--tenant-primary)] hover:underline"
                onClick={(e) => {
                  e.stopPropagation();
                  onRequestOffshelf?.();
                }}
              >
                申请下架
              </button>
            )
          )}
          {isDistributing ? (
            <Tooltip delayDuration={300}>
              <TooltipTrigger asChild>
                <span
                  className="inline-flex items-center justify-center size-8 rounded-[var(--radius-lg)] border border-[var(--border)] text-[var(--text-weak)] cursor-not-allowed flex-shrink-0"
                  onClick={(e) => e.stopPropagation()}
                >
                  <Plus className="w-4 h-4" />
                </span>
              </TooltipTrigger>
              <TooltipContent><span className="text-xs">请等待下发完成</span></TooltipContent>
            </Tooltip>
          ) : (
            <Button
              variant="tenant-outline"
              size="icon-sm"
              onClick={handleDistributeClick}
              className="flex-shrink-0"
            >
              <Plus className="w-4 h-4" />
            </Button>
          )}
        </div>
      </TenantCard>

      {/* 下发弹窗 */}
      <BatchDistributeDialog
        open={distributeOpen}
        onOpenChange={setDistributeOpen}
        skillName={skill.name}
        skillVersion={skill.version}
        onDistributionStart={handleDistributionStart}
        title="下发技能"
        showScopeFilter={false}
        instances={getMyInstances()}
        hideCreatorAndGroup
      />
    </>
  );
}

// ========== 列表行组件 ==========
function SkillListRow({
  skill,
  downloadCount,
  onClick,
  onDistStatusClick,
}: {
  skill: Skill;
  downloadCount: number;
  onClick: () => void;
  onDistStatusClick?: () => void;
}) {
  const [distributeOpen, setDistributeOpen] = useState(false);

  const latestRecord = getDistributionRecords(skill.id)?.[0];
  const distStatus = latestRecord?.status;
  const isDistributing = distStatus === 'distributing' || distStatus === 'deleting';

  const handleDistributeClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (isDistributing) return;
    setDistributeOpen(true);
  };

  const handleDistributionStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: skill.id,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'distributing',
      instances: selectedInstancesData.map(inst => ({
        id: inst.id,
        name: inst.name,
        createdBy: inst.createdBy || 'admin',
        distributionStatus: 'distributing' as DistributionStatus,
      })),
    };
    addDistributionRecord(newRecord);
    setDistributeOpen(false);
    toast.success(`已开始下发「${skill.name}」到 ${selectedInstanceIds.length} 个实例`);
    simulateDistribution(recordId, selectedInstanceIds.length);
  };

  const initial = getSkillInitial(skill.name);
  // 格式化短日期 250303
  const shortDate = (() => {
    const d = typeof skill.uploadTime === 'string' ? new Date(skill.uploadTime) : skill.uploadTime;
    const y = String(d.getFullYear()).slice(2);
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${y}${m}${day}`;
  })();

  return (
    <>
      <div
        className="flex items-center px-5 py-5 hover:bg-[var(--accent)] transition-colors cursor-pointer gap-4"
        onClick={onClick}
      >
        {/* 左：图标(字母) + 名称 + 描述 */}
        <div className="flex items-center gap-3 flex-1 min-w-0">
          {initial && (
            <div
              className="w-12 h-12 rounded flex items-center justify-center flex-shrink-0"
              style={{ background: getLetterColor(initial).bg }}
            >
              <span className="text-sm font-bold" style={{ color: getLetterColor(initial).text }}>{initial}</span>
            </div>
          )}
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="font-semibold truncate text-[var(--text-emphasis)]">{skill.name}</span>
            </div>
            <p className="text-sm truncate mt-1 text-[var(--text-muted)]">{skill.description}</p>
          </div>
        </div>

        {/* 下载量 */}
        <div className="flex items-center gap-1 flex-shrink-0 text-xs tabular-nums whitespace-nowrap text-[var(--text-weak)]">
          <Download className="w-3 h-3" />
          {formatDownloadCount(downloadCount)}
        </div>

        {/* 版本+日期 */}
        <div className="flex-shrink-0 text-xs tabular-nums whitespace-nowrap text-[var(--text-weak)]">
          v{skill.version}({shortDate})
        </div>

        {/* 下发状态图标 — 未下发时也显示置灰图标占位 */}
        {distStatus ? (
          <DistributionStatusIcon status={distStatus} latestRecord={latestRecord} onClick={() => onDistStatusClick?.()} />
        ) : (
          <Tooltip delayDuration={300}>
            <TooltipTrigger asChild>
              <span className="inline-flex cursor-default" onClick={(e) => e.stopPropagation()}>
                <Circle className="w-3.5 h-3.5 text-[var(--border)] hover:text-[var(--text-brand)] transition-colors" />
              </span>
            </TooltipTrigger>
            <TooltipContent><span className="text-xs">还没下发过</span></TooltipContent>
          </Tooltip>
        )}

        {/* + 按钮 */}
        {isDistributing ? (
          <Tooltip delayDuration={300}>
            <TooltipTrigger asChild>
              <span
                className="w-7 h-7 rounded-[var(--radius-lg)] border border-[var(--border)] flex items-center justify-center cursor-not-allowed flex-shrink-0 text-[var(--text-muted)]"
                onClick={(e) => e.stopPropagation()}
              >
                <Plus className="w-3.5 h-3.5" />
              </span>
            </TooltipTrigger>
            <TooltipContent><span className="text-xs">请等待下发完成</span></TooltipContent>
          </Tooltip>
        ) : (
          <Button
            variant="tenant-outline"
            size="icon"
            onClick={handleDistributeClick}
            className="w-7 h-7 flex-shrink-0"
          >
            <Plus className="w-3.5 h-3.5" />
          </Button>
        )}
      </div>

      <BatchDistributeDialog
        open={distributeOpen}
        onOpenChange={setDistributeOpen}
        skillName={skill.name}
        skillVersion={skill.version}
        onDistributionStart={handleDistributionStart}
        title="下发技能"
        showScopeFilter={false}
        instances={getMyInstances()}
        hideCreatorAndGroup
      />
    </>
  );
}

// ========== 详情页组件 ==========
function SkillSquareDetail({
  skillId,
  skills,
  onBack,
  initialTab = 'overview',
}: {
  skillId: string;
  skills: Skill[];
  onBack: () => void;
  initialTab?: string;
}) {
  const [distributeDialogOpen, setDistributeDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [isDownloading, setIsDownloading] = useState(false);
  const [expandedFile, setExpandedFile] = useState<string | null>('SKILL.md');
  const [expandedDirs, setExpandedDirs] = useState<Set<string>>(new Set());
  const [activeTab, setActiveTab] = useState(initialTab);
  const [selectedVersion, setSelectedVersion] = useState<string>('');
  const [fileViewMode, setFileViewMode] = useState<'preview' | 'source'>('preview');

  // 下发记录
  const [distributionRecords, setDistributionRecords] = useState<CachedDistributionRecord[]>([]);
  const [activeDistributionId, setActiveDistributionId] = useState<string | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [statusFilter, setStatusFilter] = useState<'all' | DistributionStatus>('all');
  const [detailSearchQuery, setDetailSearchQuery] = useState('');
  /** 记录类型筛选：全部/下发记录/卸载记录 */
  const [recordTypeFilter, setRecordTypeFilter] = useState<'all' | 'distribute' | 'delete'>('all');

  const refreshRecords = useCallback(() => {
    setDistributionRecords(getDistributionRecords(skillId));
  }, [skillId]);

  useEffect(() => {
    refreshRecords();
    const handler = () => refreshRecords();
    window.addEventListener('distribution-cache-updated', handler);
    return () => window.removeEventListener('distribution-cache-updated', handler);
  }, [refreshRecords]);

  const hasInProgress = distributionRecords.some(r => r.status === 'distributing' || r.status === 'deleting');

  const skill = useMemo(() => skills.find(s => s.id === skillId), [skillId, skills]);

  useEffect(() => {
    if (skill?.versions && skill.versions.length > 0 && !selectedVersion) {
      setSelectedVersion(skill.versions[0]);
    }
  }, [skill?.versions, selectedVersion]);

  useEffect(() => {
    if (selectedVersion) {
      setExpandedFile('SKILL.md');
      setExpandedDirs(new Set());
    }
  }, [selectedVersion]);

  // 文件列表处理
  const currentVersionFiles = useMemo(() => {
    if (!skill) return [];
    if (!selectedVersion || selectedVersion === skill.versions?.[0]) {
      return skill.files || [];
    }
    const versionRecord = skill.versionHistory?.find(v => v.version === selectedVersion);
    if (versionRecord?.files && versionRecord.files.length > 0) {
      return versionRecord.files;
    }
    return skill.files || [];
  }, [skill, selectedVersion]);

  const { processedFiles, strippedPrefix } = useMemo(() => {
    const rawFiles = currentVersionFiles;
    if (rawFiles.length === 0) return { processedFiles: rawFiles, strippedPrefix: '' };
    const topDirs = new Set<string>();
    let topFileCount = 0;
    for (const f of rawFiles) {
      const parts = f.name.split('/');
      if (parts.length > 1) topDirs.add(parts[0]);
      else topFileCount++;
    }
    if (topDirs.size === 1 && topFileCount === 0) {
      const prefix = Array.from(topDirs)[0] + '/';
      return {
        processedFiles: rawFiles.map(f => ({ ...f, name: f.name.slice(prefix.length) })),
        strippedPrefix: prefix,
      };
    }
    return { processedFiles: rawFiles, strippedPrefix: '' };
  }, [currentVersionFiles]);

  const VIEWABLE_EXTENSIONS = ['.md', '.xml', '.json', '.txt', '.yaml', '.yml', '.toml', '.ini', '.cfg', '.conf', '.sh', '.bat', '.py', '.js', '.ts', '.css', '.html', '.htm', '.svg', '.env', '.gitignore', '.dockerfile'];

  const isViewableFile = (name: string) => {
    const lower = name.toLowerCase();
    if (!lower.includes('.') && !lower.includes('/')) return true;
    return VIEWABLE_EXTENSIONS.some(ext => lower.endsWith(ext));
  };

  const isMarkdownFile = (name: string) => {
    const lower = name.toLowerCase();
    return lower.endsWith('.md') || lower.endsWith('.mdx');
  };

  const getFileLanguage = (name: string): string => {
    const ext = name.split('.').pop()?.toLowerCase() || '';
    const langMap: Record<string, string> = {
      json: 'json', xml: 'xml', yaml: 'yaml', yml: 'yaml',
      toml: 'toml', py: 'python', js: 'javascript', ts: 'typescript',
      css: 'css', html: 'html', htm: 'html', sh: 'bash', bat: 'batch',
      svg: 'xml', ini: 'ini', cfg: 'ini', conf: 'ini',
    };
    return langMap[ext] || 'text';
  };

  const toggleDir = (dirName: string) => {
    setExpandedDirs(prev => {
      const next = new Set(prev);
      if (next.has(dirName)) next.delete(dirName);
      else next.add(dirName);
      return next;
    });
  };

  useEffect(() => {
    if (processedFiles.length) {
      const dirs = new Set<string>();
      for (const file of processedFiles) {
        const parts = file.name.split('/');
        if (parts.length > 1) dirs.add(parts[0]);
      }
      setExpandedDirs(dirs);
    }
  }, [processedFiles]);

  const findFileInTree = (files: any[], targetName: string): any => {
    for (const f of files) {
      if (f.name === targetName || f.path === targetName) return f;
      if (f.children && f.children.length > 0) {
        const found = findFileInTree(f.children, targetName);
        if (found) return found;
      }
    }
    return null;
  };

  const getFileContent = (fileName: string): string => {
    const versionFiles = currentVersionFiles;
    if (fileName === 'SKILL.md' || fileName.toLowerCase() === 'skill.md') {
      const skillMdFile = versionFiles.find(f => f.name.toLowerCase() === 'skill.md' || f.name.toLowerCase().endsWith('/skill.md'));
      if (skillMdFile?.content) return skillMdFile.content;
      if (!selectedVersion || selectedVersion === skill?.versions?.[0]) {
        return skill?.content || '';
      }
      return '';
    }
    const originalName = strippedPrefix ? strippedPrefix + fileName : fileName;
    const file = findFileInTree(versionFiles, originalName);
    if (file?.content) return file.content;
    const file2 = findFileInTree(versionFiles, fileName);
    if (file2?.content) return file2.content;
    return '';
  };

  const renderFileTree = (files: Array<{ name: string; size?: number; content?: string }>) => {
    const sorted = [...files].sort((a, b) => a.name.localeCompare(b.name));
    const renderedDirs = new Set<string>();
    const result: React.ReactNode[] = [];

    for (const file of sorted) {
      const parts = file.name.split('/');
      const isDir = file.name.endsWith('/');
      const isNested = parts.length > 1 && !isDir;
      const canView = !isDir && isViewableFile(file.name);

      if (isNested) {
        for (let i = 1; i < parts.length; i++) {
          const dirPath = parts.slice(0, i).join('/');
          if (!renderedDirs.has(dirPath)) {
            renderedDirs.add(dirPath);
            const depth = i - 1;
            const isExpanded = expandedDirs.has(dirPath);
            let ancestorsExpanded = true;
            for (let j = 1; j < i; j++) {
              const ancestor = parts.slice(0, j).join('/');
              if (!expandedDirs.has(ancestor)) { ancestorsExpanded = false; break; }
            }
            if (!ancestorsExpanded) continue;
            result.push(
              <Button
                key={`dir-${dirPath}`}
                variant="ghost"
                onClick={() => toggleDir(dirPath)}
                className="w-full flex items-center gap-1.5 h-8 px-2 text-sm text-[var(--text-emphasis)] hover:bg-[var(--accent)] rounded-[var(--radius-lg)] justify-start"
                style={{ paddingLeft: `${8 + depth * 16}px` }}
              >
                {isExpanded
                  ? <ChevronDown className="w-3.5 h-3.5 text-[var(--text-muted)] flex-shrink-0" />
                  : <ChevronRight className="w-3.5 h-3.5 text-[var(--text-muted)] flex-shrink-0" />
                }
                {isExpanded ? <FolderOpen className="w-3.5 h-3.5 text-[var(--text-muted)] flex-shrink-0" /> : <Folder className="w-3.5 h-3.5 text-[var(--text-muted)] flex-shrink-0" />}
                <span className="truncate font-medium">{parts[i - 1]}</span>
              </Button>
            );
          }
        }
        let allParentsExpanded = true;
        for (let i = 1; i < parts.length; i++) {
          const ancestor = parts.slice(0, i).join('/');
          if (!expandedDirs.has(ancestor)) { allParentsExpanded = false; break; }
        }
        if (!allParentsExpanded) continue;
      }
      if (isDir) continue;
      const depth = parts.length - 1;
      result.push(
        <Button
          key={file.name}
          variant="ghost"
          onClick={() => canView && setExpandedFile(expandedFile === file.name ? null : file.name)}
          disabled={!canView}
          className={`w-full flex items-center gap-1.5 h-8 px-2 text-sm rounded-[var(--radius-lg)] justify-start ${
            expandedFile === file.name
              ? 'bg-[var(--accent)] text-[var(--text-emphasis)] font-medium'
              : canView ? 'hover:bg-[var(--accent)] text-[var(--text-emphasis)]' : 'text-[var(--text-weak)] opacity-60'
          }`}
          style={{ paddingLeft: `${8 + depth * 16}px` }}
        >
          <FileText className="w-3.5 h-3.5 text-[var(--text-muted)] flex-shrink-0" />
          <span className="truncate">{parts[parts.length - 1]}</span>
        </Button>
      );
    }
    return result;
  };

  // 下载
  const handleDownload = async () => {
    if (!skill) return;
    setIsDownloading(true);
    try {
      await downloadSkillAsZip(skill);
      toast.success(`「${skill.name}」下载完成`);
    } catch {
      toast.error('下载失败，请重试');
    } finally {
      setIsDownloading(false);
    }
  };

  // 下发处理
  const handleDistributionStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'distributing',
      instances: selectedInstancesData.map(inst => ({
        id: inst.id,
        name: inst.name,
        createdBy: inst.createdBy || 'admin',
        distributionStatus: 'distributing' as DistributionStatus,
      })),
    };
    addDistributionRecord(newRecord);
    setActiveDistributionId(recordId);
    setDistributeDialogOpen(false);
    simulateDistribution(recordId, selectedInstanceIds.length);
  };

  // 卸载处理
  const handleDeleteStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'deleting',
      type: 'delete',
      instances: selectedInstancesData.map(inst => ({
        id: inst.id,
        name: inst.name,
        createdBy: inst.createdBy || 'admin',
        distributionStatus: 'distributing' as DistributionStatus,
      })),
    };
    addDistributionRecord(newRecord);
    setActiveDistributionId(recordId);
    setDeleteDialogOpen(false);
    toast.success(`已开始卸载「${skill?.name}」，共 ${selectedInstanceIds.length} 个实例`);
    simulateDelete(recordId, selectedInstanceIds.length);
  };

  // 聚合已下发成功的实例（用于卸载弹窗）
  const distributedInstancesForDelete = useMemo(() => {
    const myInstances = getMyInstances();
    const successRecords = distributionRecords.filter(r => r.type !== 'delete' && (r.status === 'success' || r.status === 'failed'));
    const instanceMap = new Map<string, { id: string; name: string; createdBy: string; distributedVersion?: string; deleteStatus?: 'not_deleted' | 'delete_failed'; deleteFailReason?: string }>();
    successRecords.forEach(record => {
      record.instances.forEach(inst => {
        if (inst.distributionStatus === 'success' && myInstances.some(mi => mi.id === inst.id)) {
          if (!instanceMap.has(inst.id)) {
            instanceMap.set(inst.id, {
              id: inst.id,
              name: inst.name,
              createdBy: inst.createdBy,
              distributedVersion: skill?.version,
              deleteStatus: 'not_deleted',
            });
          }
        }
      });
    });
    // 检查卸载记录，标记已卸载失败的
    const deleteRecords = distributionRecords.filter(r => r.type === 'delete' && r.status !== 'deleting');
    deleteRecords.forEach(record => {
      record.instances.forEach(inst => {
        if (inst.distributionStatus === 'failed' && instanceMap.has(inst.id)) {
          const existing = instanceMap.get(inst.id)!;
          existing.deleteStatus = 'delete_failed';
          existing.deleteFailReason = inst.failReason;
        } else if (inst.distributionStatus === 'success') {
          // 卸载成功的从列表中移除
          instanceMap.delete(inst.id);
        }
      });
    });
    return Array.from(instanceMap.values());
  }, [distributionRecords, skill?.version]);

  // 按类型筛选的记录
  const filteredRecordsByType = useMemo(() => {
    if (recordTypeFilter === 'all') return distributionRecords;
    if (recordTypeFilter === 'distribute') return distributionRecords.filter(r => r.type !== 'delete');
    return distributionRecords.filter(r => r.type === 'delete');
  }, [distributionRecords, recordTypeFilter]);

  const getCategoryName = (catId: string) => {
    return DEFAULT_CATEGORIES.find(c => c.id === catId)?.name || catId;
  };

  const activeDistribution = distributionRecords.find(r => r.id === activeDistributionId);
  const filteredInstances = activeDistribution
    ? activeDistribution.instances.filter(inst => {
        const matchesStatus = statusFilter === 'all' || inst.distributionStatus === statusFilter;
        const searchLower = detailSearchQuery.toLowerCase();
        const matchesSearch = !detailSearchQuery ||
          inst.name.toLowerCase().includes(searchLower) ||
          inst.id.toLowerCase().includes(searchLower);
        return matchesStatus && matchesSearch;
      })
    : [];

  if (!skill) {
    return (
      <div className="text-center py-12">
        <p>技能未找到</p>
        <Button onClick={onBack} className="mt-4" variant="tenant-outline">返回列表</Button>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col relative">
      {/* ======== Header（参照设计稿）======== */}
      {/* 返回按钮在最上面，icon + 文字形式，左对齐头像 */}
      <header className="relative flex flex-col gap-4 py-6">
        {/* 返回行 */}
        <button
          onClick={onBack}
          className="inline-flex items-center gap-1.5 text-sm leading-5 text-[var(--text-secondary)] hover:text-[var(--text-brand)] transition-colors self-start"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>返回</span>
        </button>

        {/* 主信息行：头像顶对齐标题 + 右侧按钮底对齐 */}
        <div className="flex items-start justify-between gap-6">
          <div className="flex items-start gap-3">
            {/* 技能图标（首字母圆形） */}
            {(() => {
              const initial = getSkillInitial(skill.name) || 'A';
              const colors = getLetterColor(initial);
              return (
                <div
                  className="w-12 h-12 rounded flex items-center justify-center text-xl font-semibold shrink-0"
                  style={{ background: colors.bg, color: colors.text }}
                >
                  {initial}
                </div>
              );
            })()}

            <div className="flex flex-col gap-1.5">
              <div className="flex items-center gap-3">
                <h1 className="text-[22px] font-semibold leading-7 tracking-[-0.02em] text-[var(--text-emphasis)]">
                  {skill.name}
                </h1>
                <span className="inline-flex items-center px-1.5 py-[2px] rounded-[var(--radius-sm)] border border-[var(--border)] bg-[var(--accent)] text-xs leading-[18px] text-[var(--text-secondary)]">
                  v{skill.version}
                </span>
              </div>
              {/* 元信息行 */}
              <MetaText as="div" className="flex items-center flex-wrap gap-2">
                <span>slug: {skill.slug}</span>
                <span>|</span>
                {skill.categories && skill.categories.length > 0 && (
                  <>
                    <span>分类：{skill.categories.map((catId: string) => getCategoryName(catId)).join('、')}</span>
                    <span>|</span>
                  </>
                )}
                <span className="inline-flex items-center gap-1">
                  <Download className="w-3 h-3 text-[var(--text-muted)]"/>
                  {formatDownloadCount(MOCK_DOWNLOAD_COUNTS[skill.id] || 0)}
                </span>
                <span>|</span>
                <span>{formatDate(skill.uploadTime)} 发布</span>
              </MetaText>
              {skill.description && (
                <CompactText as="p" className="mt-1">
                  {skill.description}
                </CompactText>
              )}
            </div>
          </div>

          {/* 右：操作按钮组 */}
          <div className="flex items-center gap-2 shrink-0">
            <Button variant="tenant-outline" size="claw" onClick={handleDownload} disabled={isDownloading}>
              {isDownloading ? <Loader className="w-4 h-4 animate-spin" /> : <Download className="w-4 h-4" />}
              下载
            </Button>
            <Button
              variant="tenant-primary"
              size="claw"
              onClick={() => setDistributeDialogOpen(true)}
              disabled={hasInProgress}
            >
              <Plus className="w-4 h-4" />
              {hasInProgress ? '下发中...' : '下发'}
            </Button>
          </div>
        </div>
      </header>

      {/* ======== Tab 导航 + 主要内容 ======== */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full flex-1 flex flex-col">
        {/* Tab 导航段 */}
        <div className="relative py-4">
          <TenantSegmentList>
            <TenantSegmentItem value="overview">概述</TenantSegmentItem>
            <TenantSegmentItem value="files">文件列表</TenantSegmentItem>
            <TenantSegmentItem value="distribution">下发和卸载记录</TenantSegmentItem>
          </TenantSegmentList>
        </div>

        {/* 主要内容段（flex-1：内容不足一屏时撑开，把底部分隔栏顶到底部） */}
        <div className="py-0 flex-1">
          {/* 概述 Tab */}
          <TabsContent value="overview" className="mt-0 p-0">
            <TenantCard padding="none" className="p-6">
              <MDXRenderer content={(() => {
                if (!selectedVersion || selectedVersion === skill.versions?.[0]) {
                  return skill.content || '';
                }
                const versionFiles = currentVersionFiles;
                const skillMdFile = versionFiles.find(f => f.name.toLowerCase() === 'skill.md' || f.name.toLowerCase().endsWith('/skill.md'));
                return skillMdFile?.content || skill.content || '';
              })()} />
            </TenantCard>
          </TabsContent>

          {/* 文件列表 Tab */}
          <TabsContent value="files" className="mt-0 p-0">
            <TenantCard padding="none" className="flex h-[47rem] overflow-hidden">
              {/* 左列：版本选择 */}
              <div className="w-[14%] min-w-[120px] border-r border-[var(--border)] flex flex-col">
                <div className="h-12 px-3 border-b border-[var(--border)] flex items-center">
                  <p className="text-sm font-medium text-[var(--text-emphasis)]">版本</p>
                </div>
                <div className="flex-1 overflow-y-auto">
                  {skill.versions?.map((ver: string, idx: number) => {
                    const isLatest = idx === 0;
                    const isSelected = selectedVersion === ver;
                    const versionRecord = skill.versionHistory?.find(v => v.version === ver);
                    const dateStr = versionRecord?.date || '';
                    return (
                      <Button
                        key={ver}
                        variant="ghost"
                        onClick={() => setSelectedVersion(ver)}
                        className={`w-full px-3 py-3.5 border-b border-[var(--border)] rounded-none h-auto justify-start items-center ${
                          isSelected ? 'bg-[var(--accent)]' : 'hover:bg-[var(--accent)]'
                        }`}
                      >
                        <div className="flex items-center gap-1.5">
                          <span className="text-[14px] font-semibold text-[var(--text-emphasis)]">
                            {ver}
                          </span>
                          {isLatest && (
                            <span className="inline-flex h-[18px] items-center justify-center rounded-[var(--radius-sm)] border border-[var(--text-brand)] px-1 text-[10px] font-semibold leading-none tracking-[0.015em] text-[var(--text-brand)]">
                              New
                            </span>
                          )}
                          <span className="text-[12px] text-[var(--text-weak)]">{dateStr}</span>
                          <Tooltip delayDuration={300}>
                            <TooltipTrigger asChild>
                              <span className="cursor-pointer inline-flex items-center" onClick={(e) => e.stopPropagation()}>
                                <Info className="w-3 h-3 text-[var(--text-weak)] hover:text-[var(--text-emphasis)]" />
                              </span>
                            </TooltipTrigger>
                            <TooltipContent side="right" className="max-w-[260px] p-3 text-xs text-[var(--text-emphasis)]">
                              <p className="font-medium mb-1.5 text-xs text-[var(--text-emphasis)]">更新说明</p>
                              <p className="whitespace-pre-line leading-relaxed text-xs text-[var(--text-secondary)]">{versionRecord?.changeLog || '暂无更新说明'}</p>
                            </TooltipContent>
                          </Tooltip>
                        </div>
                      </Button>
                    );
                  })}
                </div>
              </div>

              {/* 中列：文件列表 */}
              <div className="w-[22%] min-w-[160px] border-r border-[var(--border)] flex flex-col">
                <div className="h-12 px-3 border-b border-[var(--border)] flex items-center justify-between">
                  <p className="text-sm font-medium text-[var(--text-emphasis)]">{selectedVersion || skill.version}</p>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={handleDownload}
                    disabled={isDownloading}
                    className="w-7 h-7 text-[var(--text-muted)] hover:text-[var(--text-emphasis)]"
                    title="下载此版本 ZIP"
                  >
                    {isDownloading ? <Loader className="w-3.5 h-3.5 animate-spin" /> : <Download className="w-3.5 h-3.5" />}
                  </Button>
                </div>
                <div className="flex-1 overflow-y-auto px-3 py-2">
                  {renderFileTree(processedFiles)}
                </div>
              </div>

              {/* 右列：文件详情 */}
              <div className="flex-1 flex flex-col bg-[var(--card)]">
                {expandedFile ? (
                  <>
                    <div className="h-12 px-3 border-b border-[var(--border)] flex items-center justify-between">
                      <p className="text-sm font-medium text-[var(--text-emphasis)]">{expandedFile}</p>
                      {/* 内嵌 Segmented Control（预览/源码，统一 segment 样式） */}
                      <TenantSegmentGroup className="h-7">
                        <TenantSegmentOption
                          className="h-7 gap-1 text-xs"
                          active={fileViewMode === 'preview'}
                          onClick={() => setFileViewMode('preview')}
                        >
                          <Eye className="w-3 h-3" />
                          预览
                        </TenantSegmentOption>
                        <TenantSegmentOption
                          className="h-7 gap-1 text-xs"
                          active={fileViewMode === 'source'}
                          onClick={() => setFileViewMode('source')}
                        >
                          <Code className="w-3 h-3" />
                          源码
                        </TenantSegmentOption>
                      </TenantSegmentGroup>
                    </div>
                    <div className="flex-1 overflow-y-auto">
                      {(() => {
                        const content = getFileContent(expandedFile);
                        if (!content) {
                          return (
                            <div className="flex items-center justify-center h-full text-[var(--text-weak)]">
                              <p className="text-sm">文件内容暂无</p>
                            </div>
                          );
                        }
                        if (fileViewMode === 'source') {
                          const lang = getFileLanguage(expandedFile);
                          registerLanguage(lang);
                          return (
                            <Suspense fallback={
                              <CodeText
                                as="pre"
                                tone="secondary"
                                className="overflow-x-auto whitespace-pre-wrap break-words bg-gray-50/50 p-3 m-0"
                              >
                                {content}
                              </CodeText>
                            }>
                              <SyntaxHighlighter
                                language={lang}
                                style={hljsStyle}
                                showLineNumbers
                                lineNumberStyle={{ color: '#A3A3A3', fontSize: '11px', minWidth: '2.5em', paddingRight: '1em', userSelect: 'none' }}
                                customStyle={{ margin: 0, padding: '12px 0', fontSize: '12px', lineHeight: '1.6', background: '#ffffff', borderRadius: 0 }}
                                wrapLongLines
                              >
                                {content}
                              </SyntaxHighlighter>
                            </Suspense>
                          );
                        }
                        if (isMarkdownFile(expandedFile)) {
                          return (
                            <div className="p-4">
                              <MDXRenderer content={content} />
                            </div>
                          );
                        }
                        const previewLang = getFileLanguage(expandedFile);
                        registerLanguage(previewLang);
                        return (
                          <Suspense fallback={
                            <CodeText
                              as="pre"
                              tone="secondary"
                              className="overflow-x-auto whitespace-pre-wrap break-words bg-gray-50/50 p-3 m-0"
                            >
                              {content}
                            </CodeText>
                          }>
                            <SyntaxHighlighter
                              language={previewLang}
                              style={hljsStyle}
                              showLineNumbers
                              lineNumberStyle={{ color: '#A3A3A3', fontSize: '11px', minWidth: '2.5em', paddingRight: '1em', userSelect: 'none' }}
                              customStyle={{ margin: 0, padding: '12px 0', fontSize: '12px', lineHeight: '1.6', background: '#ffffff', borderRadius: 0 }}
                              wrapLongLines
                            >
                              {content}
                            </SyntaxHighlighter>
                          </Suspense>
                        );
                      })()}
                    </div>
                  </>
                ) : (
                  <div className="flex items-center justify-center h-full text-[var(--text-muted)]">
                    <p className="text-sm">选择一个文件查看内容</p>
                  </div>
                )}
              </div>
            </TenantCard>
          </TabsContent>

          {/* 下发和卸载记录 Tab */}
          <TabsContent value="distribution" className="mt-0 p-0">
            <TenantCard padding="none" className="p-6">
              {/* 顶部行：标题 + 类型筛选 + 批量卸载按钮 */}
              <div className="flex items-center justify-between gap-3 flex-wrap">
                <div className="flex items-center gap-3">
                  <h3 className="font-semibold text-[var(--text-emphasis)]">下发和卸载记录</h3>
                  <TenantSegmentGroup size="sm">
                    <TenantSegmentOption
                      active={recordTypeFilter === 'all'}
                      onClick={() => setRecordTypeFilter('all')}
                    >
                      全部
                    </TenantSegmentOption>
                    <TenantSegmentOption
                      active={recordTypeFilter === 'distribute'}
                      onClick={() => setRecordTypeFilter('distribute')}
                    >
                      下发
                    </TenantSegmentOption>
                    <TenantSegmentOption
                      active={recordTypeFilter === 'delete'}
                      onClick={() => setRecordTypeFilter('delete')}
                    >
                      卸载
                    </TenantSegmentOption>
                  </TenantSegmentGroup>
                </div>
                <Button
                  variant="tenant-outline"
                  size="sm"
                  onClick={() => setDeleteDialogOpen(true)}
                  disabled={distributedInstancesForDelete.length === 0}
                >
                  <Trash2 className="w-4 h-4" />
                  批量卸载
                </Button>
              </div>

              <div className="space-y-3 mt-4">
                {filteredRecordsByType.length === 0 ? (
                  <Empty>
                    <EmptyMedia />
                    <EmptyHeader>
                      <EmptyTitle>
                        {recordTypeFilter === 'distribute'
                          ? '还没有下发记录'
                          : recordTypeFilter === 'delete'
                          ? '还没有卸载记录'
                          : '还没有下发或卸载记录'}
                      </EmptyTitle>
                      <EmptyDescription>
                        {recordTypeFilter === 'delete'
                          ? '使用上方「批量卸载」按钮可从已下发的实例中卸载该技能。'
                          : '使用上方「下发」或「批量卸载」按钮，记录将显示在这里。'}
                      </EmptyDescription>
                    </EmptyHeader>
                  </Empty>
                ) : (
                  <div className="space-y-3">
                    {filteredRecordsByType.map((record, idx) => {
                      const isDelete = record.type === 'delete';
                      const progress = record.totalCount > 0 ? Math.round((record.successCount / record.totalCount) * 100) : 0;
                      const inProgressLabel = isDelete ? '卸载中' : '下发中';
                      const finishedLabel = isDelete ? '卸载完成' : '下发完成';
                      return (
                        <TenantCard key={record.id} padding="none" className="p-4">
                          <div className="flex items-start justify-between mb-3">
                            <div className="flex items-center gap-2">
                              {isDelete && (
                                <Trash2 className="w-3.5 h-3.5 text-[var(--text-muted)]" />
                              )}
                              <p className="text-sm font-semibold text-[var(--text-emphasis)]">
                                #{idx + 1} · v{skill.version} {new Date(record.timestamp).toLocaleString('zh-CN')}
                              </p>
                            </div>
                            <div className="flex items-center gap-2">
                              <span className="inline-block px-3 py-1 rounded-[var(--radius-sm)] text-xs font-medium bg-[var(--accent)] text-[var(--text-emphasis)]">
                                {record.status === 'distributing'
                                  ? `${inProgressLabel} ${progress}%`
                                  : `${finishedLabel}，${record.successCount}个成功，${record.failedCount}个失败`}
                              </span>
                              <Button
                                size="sm"
                                variant="link"
                                onClick={() => {
                                  setActiveDistributionId(record.id);
                                  setStatusFilter('all');
                                  setDetailSearchQuery('');
                                  setDetailsOpen(true);
                                }}
                                className="h-auto py-1 px-2"
                              >
                                查看详情
                              </Button>
                            </div>
                          </div>

                          {record.status === 'distributing' && (
                            <div className="w-full rounded h-1.5 bg-[var(--accent)]">
                              <div
                                className="h-1.5 rounded transition-all duration-300 bg-[var(--text-brand)]"
                                style={{ width: `${progress}%` }}
                              />
                            </div>
                          )}
                        </TenantCard>
                      );
                    })}
                  </div>
                )}
              </div>
            </TenantCard>
          </TabsContent>
        </div>
      </Tabs>

      {/* 下发弹窗 — 用户端简化版 */}
      <BatchDistributeDialog
        open={distributeDialogOpen}
        onOpenChange={setDistributeDialogOpen}
        skillName={skill.name}
        skillVersion={skill.version}
        onDistributionStart={handleDistributionStart}
        title="下发技能"
        showScopeFilter={false}
        instances={getMyInstances()}
        hideCreatorAndGroup
      />

      {/* 卸载弹窗 — 用户端简化版 */}
      <BatchDeleteDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        skillName={skill.name}
        skillVersion={skill.version}
        distributedInstances={distributedInstancesForDelete}
        groups={[]}
        onDeleteStart={handleDeleteStart}
        showScopeFilter={false}
        hideCreatorAndGroup
      />

      {/* 下发/卸载详情弹窗 */}
      <Dialog open={detailsOpen} onOpenChange={setDetailsOpen}>
        <DialogContent size="lg">
          <DialogHeader>
            <DialogTitle>{activeDistribution?.type === 'delete' ? '卸载详情' : '下发详情'}</DialogTitle>
          </DialogHeader>

          {activeDistribution && (
            <DialogBody className="px-6 pb-6">
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <div className="relative flex-1">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-muted)]"/>
                    <Input
                      tenant
                      placeholder="搜索实例名称/ID..."
                      value={detailSearchQuery}
                      onChange={(e) => setDetailSearchQuery(e.target.value)}
                      className="pl-10 h-9"
                    />
                  </div>
                  <Select value={statusFilter} onValueChange={(value: any) => setStatusFilter(value)}>
                    <SelectTrigger tenant className="w-24 h-9">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">全部</SelectItem>
                      <SelectItem value="success">成功</SelectItem>
                      <SelectItem value="failed">失败</SelectItem>
                      <SelectItem value="distributing">{activeDistribution.type === 'delete' ? '卸载中' : '下发中'}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div className="border border-[var(--border)] rounded-[var(--radius-card)] overflow-hidden">
                  <div className="overflow-y-auto max-h-72">
                    <Table className="table-fixed">
                      <TableHeader className="bg-[var(--accent)] border-b border-[var(--border)] sticky top-0 z-10">
                        <TableRow className="hover:bg-transparent">
                          <TableHead className="w-[25%] px-4 py-2.5 text-xs font-medium text-[var(--text-muted)]">实例名称</TableHead>
                          <TableHead className="w-[30%] px-4 py-2.5 text-xs font-medium text-[var(--text-muted)]">实例ID</TableHead>
                          <TableHead className="w-[18%] px-4 py-2.5 text-xs font-medium text-[var(--text-muted)]">状态</TableHead>
                          <TableHead className="w-[27%] px-4 py-2.5 text-xs font-medium text-[var(--text-muted)]">失败原因</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {filteredInstances.length === 0 ? (
                          <TableRow className="hover:bg-transparent">
                            <TableCell colSpan={4} className="px-4 py-10 text-center text-sm text-[var(--text-weak)]">
                              暂无符合条件的记录
                            </TableCell>
                          </TableRow>
                        ) : (
                          filteredInstances.map(instance => {
                            const statusToneClass =
                              instance.distributionStatus === 'success' ? 'text-[var(--text-success)]' :
                              instance.distributionStatus === 'failed' ? 'text-[var(--text-danger)]' :
                              instance.distributionStatus === 'distributing' ? 'text-[var(--text-brand)]' :
                              'text-[var(--text-muted)]';
                            return (
                              <TableRow key={instance.id} className="border-b border-[var(--border)] last:border-b-0 hover:bg-[var(--accent)]">
                                <TableCell className="px-4 py-2.5 text-sm truncate text-[var(--text-emphasis)]">{instance.name}</TableCell>
                                <TableCell className="px-4 py-2.5 text-sm font-mono truncate text-[var(--text-muted)]">{instance.id}</TableCell>
                                <TableCell className="px-4 py-2.5">
                                  <span className={`text-xs font-medium ${statusToneClass}`}>
                                    {DISTRIBUTION_STATUS_MAP[instance.distributionStatus]?.label || '未下发'}
                                  </span>
                                </TableCell>
                                <TableCell className="px-4 py-2.5 text-sm truncate text-[var(--text-weak)]">
                                  {instance.distributionStatus === 'failed'
                                    ? (instance.failReason || '连接超时')
                                    : '-'}
                                </TableCell>
                              </TableRow>
                            );
                          })
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </div>
              </div>
            </DialogBody>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ========== 下发状态图标 ==========
function DistributionStatusIcon({ status, latestRecord, onClick }: {
  status: DistributionStatus | 'deleting';
  latestRecord?: CachedDistributionRecord;
  onClick?: (e: React.MouseEvent) => void;
}) {
  const total = latestRecord?.totalCount || 0;
  const success = latestRecord?.successCount || 0;
  const isDelete = latestRecord?.type === 'delete';

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    onClick?.(e);
  };

  if (status === 'deleting') {
    return (
      <Tooltip delayDuration={300}>
        <TooltipTrigger asChild>
          <span className="inline-flex cursor-pointer" onClick={handleClick}>
            <Loader className="w-3.5 h-3.5 text-[var(--text-danger)] animate-spin" />
          </span>
        </TooltipTrigger>
        <TooltipContent><span className="text-xs">卸载中</span></TooltipContent>
      </Tooltip>
    );
  }
  if (status === 'distributing') {
    return (
      <Tooltip delayDuration={300}>
        <TooltipTrigger asChild>
          <span className="inline-flex cursor-pointer" onClick={handleClick}>
            <Loader className="w-3.5 h-3.5 text-[var(--text-brand)] animate-spin" />
          </span>
        </TooltipTrigger>
        <TooltipContent><span className="text-xs">下发中</span></TooltipContent>
      </Tooltip>
    );
  }
  if (status === 'success') {
    return (
      <Tooltip delayDuration={300}>
        <TooltipTrigger asChild>
          <span className="inline-flex cursor-pointer" onClick={handleClick}>
            <CheckCircle className="w-3.5 h-3.5 text-[var(--text-success)]" />
          </span>
        </TooltipTrigger>
        <TooltipContent><span className="text-xs">{isDelete ? `已卸载（${success}/${total}成功）` : `已下发（${success}/${total}成功）`}</span></TooltipContent>
      </Tooltip>
    );
  }
  if (status === 'failed') {
    return (
      <Tooltip delayDuration={300}>
        <TooltipTrigger asChild>
          <span className="inline-flex cursor-pointer" onClick={handleClick}>
            <XCircle className="w-3.5 h-3.5 text-[var(--text-danger)]" />
          </span>
        </TooltipTrigger>
        <TooltipContent><span className="text-xs">{isDelete ? `卸载失败（${success}/${total}成功）` : `下发失败（${success}/${total}成功）`}</span></TooltipContent>
      </Tooltip>
    );
  }
  return null;
}

// ========== 工具函数 ==========

/** 格式化日期 */
function formatDate(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date;
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

/** 获取当前用户自己的实例（模拟：用户组织有交集的实例） */
function getMyInstances() {
  return MOCK_OPENCLAW_INSTANCES.filter(inst =>
    inst.groupIds?.some(gId => CURRENT_USER_GROUP_IDS.includes(gId))
  );
}

/** 模拟下发进度 */
function simulateDistribution(recordId: string, totalCount: number) {
  const FAIL_REASONS = ['连接超时', '实例离线', '版本冲突', '磁盘空间不足'];
  let completed = 0;
  const interval = setInterval(() => {
    completed += Math.floor(Math.random() * 3) + 1;
    if (completed >= totalCount) {
      completed = totalCount;
      clearInterval(interval);
      const failedCount = Math.floor(Math.random() * 2);
      const successCount = totalCount - failedCount;
      updateDistributionRecord(recordId, (record) => ({
        ...record,
        successCount,
        failedCount,
        inProgressCount: 0,
        status: (failedCount === 0 ? 'success' : 'failed') as DistributionStatus,
        instances: record.instances.map((inst, idx) => ({
          ...inst,
          distributionStatus: (idx < successCount ? 'success' : 'failed') as DistributionStatus,
          failReason: idx >= successCount ? FAIL_REASONS[Math.floor(Math.random() * FAIL_REASONS.length)] : undefined,
        })),
      }));
    } else {
      updateDistributionRecord(recordId, (record) => ({
        ...record,
        successCount: completed,
        inProgressCount: totalCount - completed,
      }));
    }
  }, 800);
}

/** 模拟卸载进度 */
function simulateDelete(recordId: string, totalCount: number) {
  const FAIL_REASONS = ['卸载超时', '实例繁忙', '权限不足', '进程占用中'];
  let completed = 0;
  const interval = setInterval(() => {
    completed += Math.floor(Math.random() * 2) + 1;
    if (completed >= totalCount) {
      completed = totalCount;
      clearInterval(interval);
      // 必定产生至少1个失败用于验证
      const failedCount = Math.max(1, Math.floor(Math.random() * 2));
      const successCount = totalCount - failedCount;
      updateDistributionRecord(recordId, (record) => ({
        ...record,
        successCount,
        failedCount,
        inProgressCount: 0,
        status: 'failed' as DistributionStatus,
        instances: record.instances.map((inst, idx) => ({
          ...inst,
          distributionStatus: (idx < successCount ? 'success' : 'failed') as DistributionStatus,
          failReason: idx >= successCount ? FAIL_REASONS[Math.floor(Math.random() * FAIL_REASONS.length)] : undefined,
        })),
      }));
    } else {
      updateDistributionRecord(recordId, (record) => ({
        ...record,
        successCount: completed,
        inProgressCount: totalCount - completed,
      }));
    }
  }, 1000);
}
