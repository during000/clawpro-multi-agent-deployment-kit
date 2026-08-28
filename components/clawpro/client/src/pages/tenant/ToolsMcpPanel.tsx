/**
 * ToolsMcpPanel - 用户端工具管理 Tab · MCP 配置面板
 * 三列布局：第一列 MCP 配置列表 | 第二列留空 | 第三列留空
 */
import { useState, useCallback, useRef, useEffect, useLayoutEffect } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Empty, EmptyHeader, EmptyDescription, EmptyContent, EmptyMedia } from "@/components/ui/empty";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Textarea } from "@/components/ui/textarea";
import { TenantSection } from "@/components/ui/TenantSection";
import { TenantCard } from "@/components/ui/Surface";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import {
  Plus,
  CheckCircle2,
  XCircle,
  AlignLeft,
  Copy,
} from "lucide-react";

// ─── 自定义图标（设计稿：工具管理） ─────────────────────────────────────────

const Search = ({ className, style }: { className?: string; style?: React.CSSProperties }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className} style={style}>
    <path d="M14.5306 13.4693L11.5625 10.4999C12.4524 9.34021 12.8679 7.88541 12.7247 6.43063C12.5814 4.97585 11.8902 3.63002 10.7912 2.66616C9.69212 1.7023 8.2676 1.19257 6.80657 1.24039C5.34554 1.2882 3.95739 1.88998 2.92373 2.92364C1.89007 3.9573 1.2883 5.34544 1.24048 6.80648C1.19266 8.26751 1.70239 9.69203 2.66625 10.7911C3.63011 11.8901 4.97594 12.5814 6.43072 12.7246C7.8855 12.8678 9.3403 12.4524 10.5 11.5624L13.4706 14.5337C13.5404 14.6034 13.6232 14.6588 13.7144 14.6965C13.8055 14.7343 13.9032 14.7537 14.0019 14.7537C14.1005 14.7537 14.1982 14.7343 14.2894 14.6965C14.3805 14.6588 14.4634 14.6034 14.5331 14.5337C14.6029 14.4639 14.6582 14.3811 14.696 14.2899C14.7337 14.1988 14.7532 14.1011 14.7532 14.0024C14.7532 13.9038 14.7337 13.8061 14.696 13.7149C14.6582 13.6238 14.6029 13.5409 14.5331 13.4712L14.5306 13.4693ZM2.75001 6.99991C2.75001 6.15934 2.99926 5.33765 3.46626 4.63874C3.93326 3.93983 4.59702 3.3951 5.3736 3.07343C6.15019 2.75175 7.00472 2.66759 7.82914 2.83158C8.65356 2.99556 9.41084 3.40034 10.0052 3.99471C10.5996 4.58908 11.0044 5.34636 11.1683 6.17078C11.3323 6.9952 11.2482 7.84973 10.9265 8.62632C10.6048 9.40291 10.0601 10.0667 9.36118 10.5337C8.66227 11.0007 7.84058 11.2499 7.00001 11.2499C5.87319 11.2488 4.79286 10.8006 3.99608 10.0038C3.1993 9.20706 2.75116 8.12673 2.75001 6.99991Z" fill="currentColor"/>
  </svg>
);

const RefreshCw = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M3 8.00001C3 6.67392 3.52678 5.40215 4.46447 4.46447C5.40215 3.52678 6.67392 3 8.00001 3C9.39781 3.00526 10.7395 3.55068 11.7445 4.52222L13 5.77778" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M13.0002 3V5.77778H10.2224" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M13 8C13 9.32608 12.4732 10.5979 11.5355 11.5355C10.5979 12.4732 9.32609 13 8.00001 13C6.6022 12.9947 5.26054 12.4493 4.25556 11.4778L3 10.2222" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M5.77778 10.2224H3V13.0002" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
  </svg>
);

const RefreshCwBold = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M14.25 3.00011V6.00011C14.25 6.19902 14.171 6.38978 14.0303 6.53044C13.8897 6.67109 13.6989 6.75011 13.5 6.75011H10.5C10.3011 6.75011 10.1103 6.67109 9.96967 6.53044C9.82902 6.38978 9.75 6.19902 9.75 6.00011C9.75 5.80119 9.82902 5.61043 9.96967 5.46978C10.1103 5.32912 10.3011 5.25011 10.5 5.25011H11.6875L11.2 4.76261C10.3174 3.87562 9.11879 3.37523 7.8675 3.37136H7.84062C6.6005 3.36864 5.40917 3.85429 4.52437 4.72323C4.38215 4.8623 4.19051 4.93918 3.9916 4.93696C3.7927 4.93473 3.60282 4.85358 3.46375 4.71136C3.32468 4.56913 3.2478 4.37749 3.25002 4.17858C3.25225 3.97968 3.3334 3.7898 3.47563 3.65073C4.64076 2.50664 6.20957 1.8674 7.8425 1.87136H7.875C9.52234 1.87585 11.1005 2.53431 12.2625 3.70198L12.75 4.18761V3.00011C12.75 2.80119 12.829 2.61043 12.9697 2.46978C13.1103 2.32912 13.3011 2.25011 13.5 2.25011C13.6989 2.25011 13.8897 2.32912 14.0303 2.46978C14.171 2.61043 14.25 2.80119 14.25 3.00011ZM11.4756 11.277C10.5904 12.1464 9.39827 12.6321 8.1575 12.6289H8.13062C6.87933 12.625 5.68074 12.1246 4.79812 11.2376L4.3125 10.7501H5.5C5.69891 10.7501 5.88968 10.6711 6.03033 10.5304C6.17098 10.3898 6.25 10.199 6.25 10.0001C6.25 9.80119 6.17098 9.61043 6.03033 9.46978C5.88968 9.32912 5.69891 9.25011 5.5 9.25011H2.5C2.30109 9.25011 2.11032 9.32912 1.96967 9.46978C1.82902 9.61043 1.75 9.80119 1.75 10.0001V13.0001C1.75 13.199 1.82902 13.3898 1.96967 13.5304C2.11032 13.6711 2.30109 13.7501 2.5 13.7501C2.69891 13.7501 2.88968 13.6711 3.03033 13.5304C3.17098 13.3898 3.25 13.199 3.25 13.0001V11.8126L3.7375 12.3001C4.89983 13.4671 6.47793 14.1249 8.125 14.1289H8.16C9.79293 14.1328 11.3617 13.4936 12.5269 12.3495C12.5973 12.2806 12.6535 12.1986 12.6922 12.108C12.7309 12.0174 12.7514 11.9201 12.7525 11.8216C12.7536 11.7231 12.7353 11.6254 12.6986 11.534C12.6619 11.4426 12.6076 11.3593 12.5387 11.2889C12.4699 11.2184 12.3878 11.1623 12.2973 11.1236C12.2067 11.0848 12.1094 11.0644 12.0109 11.0633C11.9124 11.0622 11.8147 11.0805 11.7233 11.1171C11.6318 11.1538 11.5485 11.2081 11.4781 11.277H11.4756Z" fill="currentColor"/>
  </svg>
);

const Globe = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M8 1.25C6.66498 1.25 5.35994 1.64588 4.2499 2.38758C3.13987 3.12928 2.27471 4.18349 1.76382 5.41689C1.25292 6.65029 1.11925 8.00749 1.3797 9.31686C1.64015 10.6262 2.28303 11.829 3.22703 12.773C4.17104 13.717 5.37377 14.3598 6.68314 14.6203C7.99252 14.8808 9.34971 14.7471 10.5831 14.2362C11.8165 13.7253 12.8707 12.8601 13.6124 11.7501C14.3541 10.6401 14.75 9.33502 14.75 8C14.748 6.2104 14.0362 4.49466 12.7708 3.22922C11.5053 1.96378 9.78961 1.25199 8 1.25ZM13.1956 7.25H11.2225C11.1233 5.77922 10.6652 4.35517 9.88813 3.1025C10.7573 3.4391 11.5214 4.00043 12.1026 4.72913C12.6837 5.45783 13.0609 6.32775 13.1956 7.25ZM8 12.9375C7.415 12.2619 6.47125 10.8669 6.28438 8.75H9.71813C9.62476 9.91793 9.25847 11.0476 8.64875 12.0481C8.45694 12.3617 8.23997 12.6591 8 12.9375ZM6.28438 7.25C6.37775 6.08207 6.74403 4.95238 7.35375 3.95187C7.54477 3.63843 7.76089 3.341 8 3.0625C8.585 3.73813 9.52875 5.13313 9.71563 7.25H6.28438ZM6.11188 3.1025C5.33482 4.35517 4.87666 5.77922 4.7775 7.25H2.80438C2.93913 6.32775 3.31634 5.45783 3.89745 4.72913C4.47856 4.00043 5.24274 3.4391 6.11188 3.1025ZM2.80438 8.75H4.7775C4.87666 10.2208 5.33482 11.6448 6.11188 12.8975C5.24274 12.5609 4.47856 11.9996 3.89745 11.2709C3.31634 10.5422 2.93913 9.67225 2.80438 8.75ZM9.88813 12.8975C10.6652 11.6448 11.1233 10.2208 11.2225 8.75H13.1956C13.0609 9.67225 12.6837 10.5422 12.1026 11.2709C11.5214 11.9996 10.7573 12.5609 9.88813 12.8975Z" fill="currentColor"/>
  </svg>
);

const IWikiIcon = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M13.5306 4.96938L10.0306 1.46938C9.96092 1.39975 9.87818 1.34454 9.78714 1.3069C9.69609 1.26926 9.59852 1.24992 9.5 1.25H3.5C3.16848 1.25 2.85054 1.3817 2.61612 1.61612C2.3817 1.85054 2.25 2.16848 2.25 2.5V13.5C2.25 13.8315 2.3817 14.1495 2.61612 14.3839C2.85054 14.6183 3.16848 14.75 3.5 14.75H12.5C12.8315 14.75 13.1495 14.6183 13.3839 14.3839C13.6183 14.1495 13.75 13.8315 13.75 13.5V5.5C13.7501 5.40148 13.7307 5.30391 13.6931 5.21286C13.6555 5.12182 13.6003 5.03908 13.5306 4.96938ZM10 3.5625L11.4375 5H10V3.5625ZM3.75 13.25V2.75H8.5V5.75C8.5 5.94891 8.57902 6.13968 8.71967 6.28033C8.86032 6.42098 9.05109 6.5 9.25 6.5H12.25V13.25H3.75ZM10.25 9.5C10.25 9.69891 10.171 9.88968 10.0303 10.0303C9.88968 10.171 9.69891 10.25 9.5 10.25H6.5C6.30109 10.25 6.11032 10.171 5.96967 10.0303C5.82902 9.88968 5.75 9.69891 5.75 9.5C5.75 9.30109 5.82902 9.11032 5.96967 8.96967C6.11032 8.82902 6.30109 8.75 6.5 8.75H9.5C9.69891 8.75 9.88968 8.82902 10.0303 8.96967C10.171 9.11032 10.25 9.30109 10.25 9.5Z" fill="currentColor"/>
  </svg>
);

const TapdIcon = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M5.75 9C5.75 8.80109 5.82902 8.61033 5.96967 8.46968C6.11032 8.32902 6.30109 8.25 6.5 8.25H9.5C9.69891 8.25 9.88968 8.32902 10.0303 8.46968C10.171 8.61033 10.25 8.80109 10.25 9C10.25 9.19892 10.171 9.38968 10.0303 9.53033C9.88968 9.67099 9.69891 9.75 9.5 9.75H6.5C6.30109 9.75 6.11032 9.67099 5.96967 9.53033C5.82902 9.38968 5.75 9.19892 5.75 9ZM14.75 5.5V12.5556C14.7497 12.8723 14.6237 13.1759 14.3998 13.3998C14.1759 13.6237 13.8723 13.7497 13.5556 13.75H2.46125C2.14016 13.7495 1.83236 13.6217 1.60531 13.3947C1.37827 13.1676 1.2505 12.8598 1.25 12.5388V3.25C1.25 2.91848 1.3817 2.60054 1.61612 2.36612C1.85054 2.1317 2.16848 2 2.5 2H5.77563C5.95268 1.99953 6.12781 2.03666 6.28943 2.10896C6.45106 2.18126 6.59547 2.28707 6.71313 2.41938L8.33813 4.25H13.5C13.8315 4.25 14.1495 4.3817 14.3839 4.61612C14.6183 4.85054 14.75 5.16848 14.75 5.5ZM2.75 4.25H6.33L5.66313 3.5H2.75V4.25ZM13.25 5.75H2.75V12.25H13.25V5.75Z" fill="currentColor"/>
  </svg>
);

/** 根据 serverName 返回对应的 MCP 图标 */
function getMcpIcon(serverName: string, className: string) {
  switch (serverName) {
    case "iwiki": return <IWikiIcon className={className} />;
    case "tapd": return <TapdIcon className={className} />;
    case "gongfeng": return <Globe className={className} />;
    default: return <Terminal className={className} />;
  }
}

const Terminal = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M3.21937 6.5306L1.21937 4.5306C1.14945 4.46092 1.09397 4.37813 1.05612 4.28696C1.01827 4.1958 0.998779 4.09806 0.998779 3.99935C0.998779 3.90064 1.01827 3.8029 1.05612 3.71173C1.09397 3.62057 1.14945 3.53778 1.21937 3.4681L3.21937 1.4681C3.28914 1.39833 3.37196 1.34299 3.46311 1.30524C3.55426 1.26748 3.65196 1.24805 3.75062 1.24805C3.84928 1.24805 3.94698 1.26748 4.03813 1.30524C4.12928 1.34299 4.21211 1.39833 4.28187 1.4681C4.35164 1.53786 4.40698 1.62069 4.44473 1.71184C4.48249 1.80299 4.50192 1.90069 4.50192 1.99935C4.50192 2.09801 4.48249 2.19571 4.44473 2.28686C4.40698 2.37801 4.35164 2.46083 4.28187 2.5306L2.8125 3.99997L4.28062 5.46935C4.42152 5.61024 4.50067 5.80134 4.50067 6.0006C4.50067 6.19986 4.42152 6.39095 4.28062 6.53185C4.13972 6.67274 3.94863 6.7519 3.74937 6.7519C3.55011 6.7519 3.35902 6.67274 3.21812 6.53185L3.21937 6.5306ZM6.21937 6.5306C6.28905 6.60052 6.37184 6.656 6.46301 6.69385C6.55417 6.7317 6.65191 6.75119 6.75062 6.75119C6.84933 6.75119 6.94707 6.7317 7.03823 6.69385C7.1294 6.656 7.21219 6.60052 7.28187 6.5306L9.28187 4.5306C9.35179 4.46092 9.40727 4.37813 9.44512 4.28696C9.48298 4.1958 9.50246 4.09806 9.50246 3.99935C9.50246 3.90064 9.48298 3.8029 9.44512 3.71173C9.40727 3.62057 9.35179 3.53778 9.28187 3.4681L7.28187 1.4681C7.14097 1.3272 6.94988 1.24805 6.75062 1.24805C6.55136 1.24805 6.36027 1.3272 6.21937 1.4681C6.07847 1.60899 5.99932 1.80009 5.99932 1.99935C5.99932 2.19861 6.07847 2.3897 6.21937 2.5306L7.6875 3.99997L6.21937 5.46935C6.14964 5.539 6.09432 5.62172 6.05658 5.71277C6.01883 5.80382 5.99941 5.90141 5.99941 5.99997C5.99941 6.09853 6.01883 6.19613 6.05658 6.28718C6.09432 6.37823 6.14964 6.46094 6.21937 6.5306ZM12.5 2.24997H11.25C11.0511 2.24997 10.8603 2.32899 10.7197 2.46964C10.579 2.61029 10.5 2.80106 10.5 2.99997C10.5 3.19889 10.579 3.38965 10.7197 3.5303C10.8603 3.67096 11.0511 3.74997 11.25 3.74997H12.25V12.25H3.75V8.74997C3.75 8.55106 3.67098 8.3603 3.53033 8.21964C3.38967 8.07899 3.19891 7.99997 3 7.99997C2.80108 7.99997 2.61032 8.07899 2.46967 8.21964C2.32901 8.3603 2.25 8.55106 2.25 8.74997V12.5C2.25 12.8315 2.38169 13.1494 2.61611 13.3839C2.85053 13.6183 3.16848 13.75 3.5 13.75H12.5C12.8315 13.75 13.1495 13.6183 13.3839 13.3839C13.6183 13.1494 13.75 12.8315 13.75 12.5V3.49997C13.75 3.16845 13.6183 2.85051 13.3839 2.61609C13.1495 2.38167 12.8315 2.24997 12.5 2.24997Z" fill="currentColor"/>
  </svg>
);

const Code2 = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M12 10.6666L14.6667 7.99992L12 5.33325" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M3.99992 5.33325L1.33325 7.99992L3.99992 10.6666" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M9.66659 2.66675L6.33325 13.3334" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
  </svg>
);

const Trash2 = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M6.88892 7.55542V10.8888" stroke="currentColor" strokeWidth="0.833333" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M9.11108 7.55542V10.8888" stroke="currentColor" strokeWidth="0.833333" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M11.8889 4.77783V12.5556C11.8889 12.8503 11.7718 13.1329 11.5634 13.3413C11.3551 13.5497 11.0724 13.6667 10.7778 13.6667H5.2222C4.92751 13.6667 4.64489 13.5497 4.43652 13.3413C4.22815 13.1329 4.11108 12.8503 4.11108 12.5556V4.77783" stroke="currentColor" strokeWidth="0.833333" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M3 4.77783H13" stroke="currentColor" strokeWidth="0.833333" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M5.77783 4.77764V3.66653C5.77783 3.06135 6.28376 2.55542 6.88894 2.55542H9.11117C9.71635 2.55542 10.2223 3.06135 10.2223 3.66653V4.77764" stroke="currentColor" strokeWidth="0.833333" strokeLinecap="round" strokeLinejoin="round"/>
  </svg>
);
import { Switch } from "@/components/ui/switch";

// ── 类型定义 ──────────────────────────────────────────

/** 用户端 MCP 配置项 */
interface UserMCP {
  id: string;
  serverName: string;
  displayName: string;
  description: string;
  /** 连接类型：stdio=本地命令，sse/streamable-http=远程服务 */
  transportType: "stdio" | "sse" | "streamable-http";
  /** 连接状态 */
  status: "connected" | "failed";
  /** 是否启用 */
  enabled: boolean;
  /** 工具列表（仅连接成功时有值） */
  tools: string[];
  /** 报错信息（仅连接失败时有值） */
  errorMessage?: string;
  /** 完整 JSON 配置 */
  configJson: string;
  /** 用户自填参数 */
  userParams: Record<string, string>;
}

/** 企业 MCP 模板（供选配） */
interface EnterpriseMCPTemplate {
  id: string;
  serverName: string;
  displayName: string;
  description: string;
  /** 需用户填写的参数名列表 */
  userRequiredParams: string[];
  /** 完整 JSON 配置模板 */
  configJsonTemplate: string;
}

// ── Mock 数据 ──────────────────────────────────────────

const MOCK_ENTERPRISE_MCP_TEMPLATES: EnterpriseMCPTemplate[] = [
  {
    id: "tpl-1",
    serverName: "gongfeng",
    displayName: "工蜂 MCP 服务",
    description: "通过 MCP 协议连接工蜂代码仓库，支持代码搜索、文件浏览、PR 管理等操作",
    userRequiredParams: ["your-gongfeng-token"],
    configJsonTemplate: JSON.stringify({
      mcp: {
        servers: {
          gongfeng: {
            url: "https://gongfeng.example.com/mcp/sse",
            transportType: "sse",
            headers: { Authorization: "<your-gongfeng-token>" },
            timeout: 60,
          },
        },
      },
    }, null, 2),
  },
  {
    id: "tpl-2",
    serverName: "iwiki",
    displayName: "iWiki 文档服务",
    description: "连接 iWiki 知识库平台，支持文档搜索、内容获取等操作",
    userRequiredParams: ["your-iwiki-token"],
    configJsonTemplate: JSON.stringify({
      mcp: {
        servers: {
          iwiki: {
            url: "https://iwiki.example.com/mcp",
            transportType: "streamable-http",
            headers: { Authorization: "<your-iwiki-token>" },
            timeout: 60,
          },
        },
      },
    }, null, 2),
  },
  {
    id: "tpl-3",
    serverName: "tapd",
    displayName: "TAPD 项目管理",
    description: "连接 TAPD 项目管理平台，支持需求查询、缺陷管理、迭代跟踪等功能",
    userRequiredParams: ["your-tapd-token"],
    configJsonTemplate: JSON.stringify({
      mcp: {
        servers: {
          tapd: {
            url: "https://tapd.example.com/mcp/sse",
            transportType: "sse",
            headers: { Authorization: "Bearer <your-tapd-token>" },
            timeout: 90,
          },
        },
      },
    }, null, 2),
  },
  {
    id: "tpl-4",
    serverName: "cos-storage",
    displayName: "COS 对象存储",
    description: "通过 MCP 协议访问腾讯云 COS 存储桶，支持文件上传、下载、列表等操作",
    userRequiredParams: ["Authorization"],
    configJsonTemplate: JSON.stringify({
      mcp: {
        servers: {
          "cos-storage": {
            url: "https://cos-mcp.example.com/sse",
            transportType: "sse",
            headers: {
              "Authorization": "<Authorization>",
            },
            timeout: 120,
          },
        },
      },
    }, null, 2),
  },
  {
    id: "tpl-5",
    serverName: "wedata",
    displayName: "WeData 数据开发",
    description: "连接 WeData 数据开发治理平台，支持任务查询、数据预览、血缘分析等",
    userRequiredParams: [],
    configJsonTemplate: JSON.stringify({
      mcp: {
        servers: {
          wedata: {
            url: "https://wedata-mcp.example.com/mcp",
            transportType: "streamable-http",
            timeout: 60,
          },
        },
      },
    }, null, 2),
  },
];

const INITIAL_USER_MCPS: UserMCP[] = [
  {
    id: "u-0",
    serverName: "iwiki",
    displayName: "iWiki 文档服务",
    description: "连接 iWiki 知识库平台，支持文档搜索、内容获取等操作",
    transportType: "streamable-http",
    status: "connected",
    enabled: true,
    tools: ["tool_1", "tool_2", "tool_3"],
    configJson: JSON.stringify({
      mcp: {
        servers: {
          iwiki: {
            url: "https://iwiki.example.com/mcp",
            transportType: "streamable-http",
            headers: { Authorization: "<your-iwiki-token>" },
            timeout: 60,
          },
        },
      },
    }, null, 2),
    userParams: { "your-iwiki-token": "<your-iwiki-token>" },
  },
  {
    id: "u-1",
    serverName: "gongfeng",
    displayName: "工蜂 MCP 服务",
    description: "通过 MCP 协议连接工蜂代码仓库，支持代码搜索、文件浏览、PR 管理等操作",
    transportType: "sse",
    status: "connected",
    enabled: true,
    tools: ["search_projects", "get_blob_content", "create_merge_request", "list_branches", "get_commit_info", "get_file_tree", "compare_branches", "list_merge_requests", "get_pipeline_status", "trigger_build"],
    configJson: JSON.stringify({
      mcp: {
        servers: {
          gongfeng: {
            url: "https://gongfeng.example.com/mcp/sse",
            transportType: "sse",
            headers: { Authorization: "<your-gongfeng-token>" },
            timeout: 60,
          },
        },
      },
    }, null, 2),
    userParams: { "your-gongfeng-token": "<your-gongfeng-token>" },
  },
  {
    id: "u-2",
    serverName: "tapd",
    displayName: "TAPD 项目管理",
    description: "连接 TAPD 项目管理平台，支持需求查询、缺陷管理、迭代跟踪等功能",
    transportType: "sse",
    status: "failed",
    enabled: true,
    tools: [],
    errorMessage: "连接超时：无法在 90s 内建立 SSE 连接，请检查网络或 Token 是否正确",
    configJson: JSON.stringify({
      mcp: {
        servers: {
          tapd: {
            url: "https://tapd.example.com/mcp/sse",
            transportType: "sse",
            headers: { Authorization: "Bearer <your-tapd-token>" },
            timeout: 90,
          },
        },
      },
    }, null, 2),
    userParams: { "your-tapd-token": "<your-tapd-token>" },
  },
  {
    id: "u-3",
    serverName: "your-mcp",
    displayName: "",
    description: "用户自定义的本地 MCP 服务",
    transportType: "stdio",
    status: "connected",
    enabled: true,
    tools: ["custom_tool_a", "custom_tool_b"],
    configJson: JSON.stringify({
      mcp: {
        servers: {
          "your-mcp": {
            command: "npx",
            args: ["-y", "your-mcp-server"],
            transportType: "stdio",
            timeout: 30,
          },
        },
      },
    }, null, 2),
    userParams: {},
  },
];

// ── 辅助函数 ──────────────────────────────────────────

/** 整理缩进：移除所有行的最小公共前导空白，清理尾部空行 */
function trimCommonIndent(text: string): string {
  const lines = text.replace(/\t/g, '    ').split('\n');
  const nonEmptyLines = lines.filter(l => l.trim().length > 0);
  if (nonEmptyLines.length === 0) return text;
  const minIndent = Math.min(...nonEmptyLines.map(l => l.match(/^(\s*)/)?.[1].length ?? 0));
  if (minIndent === 0) return text;
  const trimmed = lines.map(l => (l.trim().length > 0 ? l.slice(minIndent) : '')).join('\n');
  return trimmed.replace(/\n+$/, '');
}

/** 从完整 JSON 中提取指定 server 的内部内容（不含 key 和外层花括号） */
function extractServerValue(fullJson: string, serverName: string): string {
  try {
    const parsed = JSON.parse(fullJson);
    const server = parsed?.mcp?.servers?.[serverName];
    if (!server || typeof server !== 'object') return '';
    const inner = JSON.stringify(server, null, 2);
    const lines = inner.split('\n');
    if (lines.length <= 2) return '';
    return lines.slice(1, -1).map(l => l.replace(/^  /, '')).join('\n');
  } catch {
    return '';
  }
}

/** 将用户编辑的 server 内部内容组装成完整 JSON 字符串 */
function assembleFullJson(serverName: string, serverValueContent: string): string {
  const indentedLines = serverValueContent
    .split('\n')
    .map(line => (line.trim() ? `        ${line}` : ''))
    .join('\n');
  const escapedName = JSON.stringify(serverName);
  return `{\n  "mcp": {\n    "servers": {\n      ${escapedName}: {\n${indentedLines}\n      }\n    }\n  }\n}`;
}

// ── 组件 ──────────────────────────────────────────────

export default function ToolsMcpPanel() {
  // ── 列表状态 ──
  const [mcpList, setMcpList] = useState<UserMCP[]>(INITIAL_USER_MCPS);
  const [searchQuery, setSearchQuery] = useState("");
  const [refreshing, setRefreshing] = useState(false);
  /** 展开工具列表的 MCP id */
  const [expandedToolsId, setExpandedToolsId] = useState<string | null>(null);
  /** 记录每个 MCP 工具列表是否溢出两行 */
  const [overflowMap, setOverflowMap] = useState<Record<string, boolean>>({});
  /** 工具列表容器 ref */
  const toolsRefMap = useRef<Record<string, HTMLDivElement | null>>({});

  // ── 添加 MCP 弹窗 ──
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [addSearchQuery, setAddSearchQuery] = useState("");
  /** 当前需填写参数的模板 */
  const [paramTemplate, setParamTemplate] = useState<EnterpriseMCPTemplate | null>(null);
  const [paramValues, setParamValues] = useState<Record<string, string>>({});
  /** 每个参数的注入位置：'header' | 'body'，默认 'header' */
  const [paramInjectTarget, setParamInjectTarget] = useState<Record<string, 'header' | 'body'>>({});

  // ── 查看源码弹窗 ──
  const [sourceDialogOpen, setSourceDialogOpen] = useState(false);
  /** 可编辑的 server 内部字段内容 */
  const [sourceEditorContent, setSourceEditorContent] = useState("");
  const [sourceServerName, setSourceServerName] = useState("");
  const [sourceDisplayName, setSourceDisplayName] = useState("");
  const [sourceMcpId, setSourceMcpId] = useState<string | null>(null);
  const [sourceJsonError, setSourceJsonError] = useState("");

  // ── 删除确认 ──
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteMcpId, setDeleteMcpId] = useState<string | null>(null);

  // ── 重启确认（删除/保存源码/切换开关后弹出） ──
  const [restartDialogOpen, setRestartDialogOpen] = useState(false);
  const [restartAction, setRestartAction] = useState<"delete" | "save" | "toggle">("save");
  /** 切换开关时记录的 MCP id，取消修改时需要回退 */
  const [toggleRevertId, setToggleRevertId] = useState<string | null>(null);

  // ── 检测工具列表是否溢出两行 ──
  const checkOverflow = useCallback(() => {
    const newMap: Record<string, boolean> = {};
    mcpList.forEach((mcp) => {
      const el = toolsRefMap.current[mcp.id];
      if (el) {
        // scrollHeight 是内容真实高度，clientHeight 是 maxHeight 限制后的可见高度
        newMap[mcp.id] = el.scrollHeight > el.clientHeight + 1;
      }
    });
    setOverflowMap(newMap);
  }, [mcpList]);

  useLayoutEffect(() => {
    const raf = requestAnimationFrame(() => checkOverflow());
    return () => cancelAnimationFrame(raf);
  }, [checkOverflow]);

  useEffect(() => {
    const observer = new ResizeObserver(() => checkOverflow());
    Object.values(toolsRefMap.current).forEach((el) => {
      if (el) observer.observe(el);
    });
    return () => observer.disconnect();
  }, [checkOverflow]);

  // ── 过滤 ──
  const filteredList = mcpList.filter(
    (m) =>
      m.displayName.toLowerCase().includes(searchQuery.toLowerCase()) ||
      m.serverName.toLowerCase().includes(searchQuery.toLowerCase()) ||
      m.description.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // ── 刷新状态 ──
  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    setTimeout(() => {
      // Mock：随机切换一些 MCP 的连接状态
      setMcpList((prev) =>
        prev.map((m) => {
          if (!m.enabled) return m;
          const rand = Math.random();
          if (rand > 0.7) {
            return {
              ...m,
              status: m.status === "connected" ? "failed" : "connected",
              tools: m.status === "connected" ? [] : ["tool_a", "tool_b"],
              errorMessage: m.status === "connected" ? "连接超时，请检查网络配置" : undefined,
            };
          }
          return m;
        })
      );
      setRefreshing(false);
      toast.success("技能列表已刷新");
    }, 1000);
  }, []);

  // ── 单条刷新连接状态 ──
  const [refreshingSingleId, setRefreshingSingleId] = useState<string | null>(null);
  const handleRefreshSingle = useCallback((mcpId: string) => {
    setRefreshingSingleId(mcpId);
    setTimeout(() => {
      setMcpList((prev) =>
        prev.map((m) => {
          if (m.id !== mcpId || !m.enabled) return m;
          // Mock：随机切换连接状态
          const newStatus = Math.random() > 0.3 ? "connected" : "failed";
          return {
            ...m,
            status: newStatus as "connected" | "failed",
            tools: newStatus === "connected" ? (m.tools.length > 0 ? m.tools : ["tool_a", "tool_b"]) : [],
            errorMessage: newStatus === "failed" ? "连接超时，请检查网络配置" : undefined,
          };
        })
      );
      setRefreshingSingleId(null);
      toast.success("连接状态已刷新");
    }, 800);
  }, []);

  // ── 添加 MCP ──
  const alreadyAddedNames = mcpList.map((m) => m.serverName);
  const availableTemplates = MOCK_ENTERPRISE_MCP_TEMPLATES.filter(
    (t) =>
      !alreadyAddedNames.includes(t.serverName) &&
      (t.displayName.toLowerCase().includes(addSearchQuery.toLowerCase()) ||
        t.description.toLowerCase().includes(addSearchQuery.toLowerCase()))
  );

  const handleSelectTemplate = (tpl: EnterpriseMCPTemplate) => {
    if (tpl.userRequiredParams.length > 0) {
      setParamTemplate(tpl);
      setParamValues({});
      // 默认所有参数注入到 header
      const defaultTargets: Record<string, 'header' | 'body'> = {};
      tpl.userRequiredParams.forEach((p) => { defaultTargets[p] = 'header'; });
      setParamInjectTarget(defaultTargets);
    } else {
      // 无需填参数，直接添加
      doAddMCP(tpl, {});
    }
  };

  const doAddMCP = (tpl: EnterpriseMCPTemplate, params: Record<string, string>, restart: boolean = true, injectTargets?: Record<string, 'header' | 'body'>) => {
    // 替换模板中的 <xxx> 占位符
    let configJson = tpl.configJsonTemplate;
    Object.entries(params).forEach(([key, value]) => {
      configJson = configJson.replace(`<${key}>`, value);
    });
    // 将选择注入到 body 的参数，从 headers 移入 body
    if (injectTargets) {
      try {
        const parsed = JSON.parse(configJson);
        const server = parsed?.mcp?.servers?.[tpl.serverName];
        if (server) {
          Object.entries(injectTargets).forEach(([key, target]) => {
            if (target === 'body') {
              // 从 headers 中移除该 key
              if (server.headers) {
                const headerKey = Object.keys(server.headers).find(
                  (k) => server.headers[k] === params[key] || k === key
                );
                if (headerKey) {
                  const val = server.headers[headerKey];
                  delete server.headers[headerKey];
                  // 写入 body
                  if (!server.body) server.body = {};
                  server.body[headerKey] = val;
                }
              }
            }
          });
          configJson = JSON.stringify(parsed, null, 2);
        }
      } catch { /* JSON 解析失败时保持原样 */ }
    }

    const newMCP: UserMCP = {
      id: `u-${Date.now()}`,
      serverName: tpl.serverName,
      displayName: tpl.displayName,
      description: tpl.description,
      transportType: (() => {
        try {
          const parsed = JSON.parse(configJson);
          const server = parsed?.mcp?.servers?.[tpl.serverName];
          return server?.transportType || (server?.command ? "stdio" : "sse");
        } catch { return "sse"; }
      })() as "stdio" | "sse" | "streamable-http",
      status: "connected",
      enabled: true,
      tools: ["tool_1", "tool_2", "tool_3"],
      configJson,
      userParams: params,
    };
    setMcpList((prev) => [newMCP, ...prev]);
    setParamTemplate(null);
    setAddDialogOpen(false);
    setAddSearchQuery("");
    toast.success(`MCP「${tpl.displayName}」已添加`);
  };

  // ── 查看源码 ──
  const handleOpenSource = (mcp: UserMCP) => {
    setSourceServerName(mcp.serverName);
    setSourceDisplayName(mcp.displayName || mcp.serverName);
    // 提取 server 内部字段作为可编辑内容
    const inner = extractServerValue(mcp.configJson, mcp.serverName);
    setSourceEditorContent(inner);
    setSourceMcpId(mcp.id);
    setSourceJsonError("");
    setSourceDialogOpen(true);
  };

  const handleSaveSource = (restart: boolean) => {
    // 组装完整 JSON 并校验
    const fullJson = assembleFullJson(sourceServerName, sourceEditorContent);
    try {
      JSON.parse(fullJson);
    } catch {
      setSourceJsonError("JSON 格式错误，无法保存");
      return;
    }
    // 保存配置
    setMcpList((prev) =>
      prev.map((m) => (m.id === sourceMcpId ? { ...m, configJson: fullJson } : m))
    );
    setSourceDialogOpen(false);
    if (restart) {
      toast.success("已保存，正在重启实例…");
      setTimeout(() => toast.success("重启完成"), 2000);
    } else {
      toast("已保存，可稍后手动重启生效");
    }
  };

  // ── 删除 ──
  const handleDelete = (mcpId: string) => {
    setDeleteMcpId(mcpId);
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = (restart: boolean) => {
    if (!deleteMcpId) return;
    const mcp = mcpList.find((m) => m.id === deleteMcpId);
    setMcpList((prev) => prev.filter((m) => m.id !== deleteMcpId));
    setDeleteDialogOpen(false);
    setDeleteMcpId(null);
    if (restart) {
      toast.success(`MCP「${mcp?.displayName || mcp?.serverName}」已删除，正在重启实例…`);
      setTimeout(() => toast.success("重启完成"), 2000);
    } else {
      toast.success(`MCP「${mcp?.displayName || mcp?.serverName}」已删除，可稍后手动重启生效`);
    }
  };

  // ── 开启/关闭 ──
  const handleToggle = (mcpId: string) => {
    setMcpList((prev) =>
      prev.map((m) => {
        if (m.id !== mcpId) return m;
        return { ...m, enabled: !m.enabled };
      })
    );
    setToggleRevertId(mcpId);
    setRestartAction("toggle");
    setRestartDialogOpen(true);
  };

  const handleRestartCancel = () => {
    // 「取消修改」仅在 toggle 时回退
    if (restartAction === "toggle" && toggleRevertId) {
      setMcpList((prev) =>
        prev.map((m) => {
          if (m.id !== toggleRevertId) return m;
          return { ...m, enabled: !m.enabled };
        })
      );
    }
    setRestartDialogOpen(false);
    setToggleRevertId(null);
  };

  const handleRestartLater = () => {
    setRestartDialogOpen(false);
    setToggleRevertId(null);
    toast("已保存，可稍后手动重启生效");
  };

  const handleRestartNow = () => {
    setRestartDialogOpen(false);
    setToggleRevertId(null);
    toast.success("正在重启实例…");
    setTimeout(() => toast.success("重启完成"), 2000);
  };

  const deleteMcp = mcpList.find((m) => m.id === deleteMcpId);

  // ── 渲染 ──
  return (
    <>
    <TenantSection
      title="MCP 配置"
      cardPadding="default"
      actions={
        <>
          <div className="relative w-[240px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" />
            <Input
              tenant
              placeholder="搜索 MCP..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9 h-9 text-sm"
            />
          </div>
          <Button
            variant="tenant-dialog-confirm"
            onClick={() => setAddDialogOpen(true)}
            className="h-9"
          >
            <Plus className="w-4 h-4" />
            添加 MCP
          </Button>
          <Button
            variant="tenant-outline"
            size="icon"
            onClick={handleRefresh}
            disabled={refreshing}
            className="h-9 w-9"
          >
            <RefreshCwBold className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
          </Button>
        </>
      }
    >
      {/* 提示（§8.8 规范） */}
      <Alert variant="info">
        <AlertInfoIcon />
        <AlertDescription>
          状态验证仅支持公网访问的 MCP。
          <br />
          本地命令或者内网访问的MCP，需登录实例校验状态。
        </AlertDescription>
      </Alert>

      {/* 列表区域 */}
      <div className="flex-1 overflow-y-auto">
          {filteredList.length === 0 ? (
            <TenantCard padding="none">
              <Empty className="border-0 py-16">
                <EmptyHeader>
                  <EmptyMedia />
                  <EmptyDescription>暂无 MCP 配置</EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  <Button variant="tenant-outline" size="sm" onClick={() => setAddDialogOpen(true)}>
                    <Plus className="w-4 h-4" />
                    添加 MCP
                  </Button>
                </EmptyContent>
              </Empty>
            </TenantCard>
          ) : (
            <div className="grid grid-cols-2 gap-4">
              {filteredList.map((mcp) => {
                const isExpanded = expandedToolsId === mcp.id;
                const isFailed = mcp.enabled && mcp.status === "failed";
                return (
                  <TenantCard
                    key={mcp.id}
                    state={mcp.enabled ? "normal" : "static"}
                    interactive={mcp.enabled && !isFailed}
                    padding="none"
                    className={
                      !mcp.enabled
                        ? "opacity-60 bg-gray-50/50"
                        : isFailed
                          ? "!border-red-100 bg-red-50/30 hover:!border-red-200"
                          : ""
                    }
                  >
                    <div className="p-3">
                      {/* 第一行：图标+名称+状态+开关 */}
                      <div className="flex items-center gap-2">
                        <TooltipProvider delayDuration={200}>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <div className="w-7 h-7 rounded-full bg-[var(--accent)] flex items-center justify-center shrink-0">
                                {getMcpIcon(mcp.serverName, "w-3.5 h-3.5 text-[var(--foreground)]")}
                              </div>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="text-xs">
                              {mcp.transportType === "stdio" ? "本地命令" : "远程服务"}
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                        <span className="font-medium text-sm text-[var(--text-title)] truncate flex-1">
                          {mcp.displayName || mcp.serverName}
                        </span>
                        {/* 状态指示 */}
                        {mcp.enabled ? (
                          mcp.status === "connected" ? (
                            <span className="flex items-center gap-1 text-[11px] text-[var(--text-success)] shrink-0">
                              <CheckCircle2 className="w-3 h-3" />
                              已连接
                            </span>
                          ) : (
                            <span className="flex items-center gap-1 text-[11px] text-[var(--text-danger)] shrink-0">
                              <XCircle className="w-3 h-3" />
                              连接失败
                            </span>
                          )
                        ) : (
                          <span className="text-[11px] text-[var(--text-weak)] shrink-0">已关闭</span>
                        )}
                        <Switch
                          checked={mcp.enabled}
                          onCheckedChange={() => handleToggle(mcp.id)}
                        />
                        {/* 操作按钮 */}
                        <TooltipProvider delayDuration={200}>
                          <div className="flex items-center gap-1 shrink-0">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <button
                                  onClick={() => handleRefreshSingle(mcp.id)}
                                  disabled={refreshingSingleId === mcp.id}
                                  className="w-6 h-6 rounded-md text-[var(--muted-foreground)] hover:text-[var(--text-brand)] hover:bg-[#EFF6FF] flex items-center justify-center transition-colors disabled:opacity-50"
                                >
                                  <RefreshCw className={`w-3.5 h-3.5 ${refreshingSingleId === mcp.id ? "animate-spin" : ""}`} />
                                </button>
                              </TooltipTrigger>
                              <TooltipContent>刷新连接</TooltipContent>
                            </Tooltip>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <button
                                  onClick={() => handleOpenSource(mcp)}
                                  className="w-6 h-6 rounded-md text-[var(--muted-foreground)] hover:text-[var(--text-brand)] hover:bg-[#EFF6FF] flex items-center justify-center transition-colors"
                                >
                                  <Code2 className="w-3.5 h-3.5" />
                                </button>
                              </TooltipTrigger>
                              <TooltipContent>查看源码</TooltipContent>
                            </Tooltip>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <button
                                  onClick={() => handleDelete(mcp.id)}
                                  className="w-6 h-6 rounded-md text-[var(--muted-foreground)] hover:text-[var(--text-danger)] hover:bg-red-50 flex items-center justify-center transition-colors"
                                >
                                  <Trash2 className="w-3.5 h-3.5" />
                                </button>
                              </TooltipTrigger>
                              <TooltipContent>删除</TooltipContent>
                            </Tooltip>
                          </div>
                        </TooltipProvider>
                      </div>

                      {/* 第二行：描述 */}
                      <div className="mt-1">
                        <TooltipProvider delayDuration={300}>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <p className="text-xs text-[var(--text-muted)] truncate min-w-0 leading-relaxed cursor-default">
                                {mcp.description || "用户自定义 MCP"}
                              </p>
                            </TooltipTrigger>
                            <TooltipContent side="bottom" align="start" className="text-xs max-w-[280px]">
                              {mcp.description || "用户自定义 MCP"}
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </div>

                      {/* 第三行：工具列表（已连接时显示，收起时限高两行+右下角展开按钮） */}
                      {mcp.enabled && mcp.status === "connected" && mcp.tools.length > 0 && (
                        <div className="mt-1.5 relative">
                          {/* 工具标签容器 */}
                          <div
                            ref={(el) => { toolsRefMap.current[mcp.id] = el; }}
                            className={`flex flex-wrap gap-1 ${!isExpanded ? "overflow-hidden" : ""}`}
                            style={!isExpanded ? { maxHeight: "46px" } : undefined}
                          >
                            {mcp.tools.map((tool) => (
                              <span
                                key={tool}
                                className="inline-block px-1.5 py-0.5 bg-gray-100 text-[var(--text-secondary)] text-[10px] font-mono rounded whitespace-nowrap"
                              >
                                {tool}
                              </span>
                            ))}
                            {/* 展开后：收起按钮混排在最后一个标签后面 */}
                            {isExpanded && (
                              <button
                                onClick={() => setExpandedToolsId(null)}
                                className="inline-block px-1.5 py-0.5 text-[var(--text-brand)] hover:text-[var(--text-brand)] text-[10px] font-medium whitespace-nowrap"
                              >
                                收起
                              </button>
                            )}
                          </div>
                          {/* 未展开 + 有溢出时：右下角绝对定位"展开全部"按钮，带白色渐变遮罩 */}
                          {overflowMap[mcp.id] && !isExpanded && (
                            <button
                              onClick={() => setExpandedToolsId(mcp.id)}
                              className="absolute right-0 bottom-[3px] flex items-center pl-6 pr-0.5 py-0.5 text-[var(--text-brand)] hover:text-[var(--text-brand)] text-[10px] font-medium whitespace-nowrap"
                              style={{ background: "linear-gradient(to right, transparent, white 30%)" }}
                            >
                              展开全部
                            </button>
                          )}
                        </div>
                      )}

                      {/* 报错信息（连接失败时展示，独立行+复制按钮） */}
                      {mcp.enabled && mcp.status === "failed" && mcp.errorMessage && (
                        <div className="mt-1.5 flex items-start gap-1.5 px-2.5 py-1.5 bg-red-50 rounded-[var(--radius-lg)]">
                          <p className="text-[11px] text-[var(--text-danger)] leading-relaxed flex-1 min-w-0 line-clamp-2">
                            {mcp.errorMessage}
                          </p>
                          <TooltipProvider delayDuration={200}>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <button
                                  onClick={() => {
                                    navigator.clipboard.writeText(mcp.errorMessage || "");
                                    toast.success("已复制错误信息");
                                  }}
                                  className="w-5 h-5 rounded text-red-400 hover:text-[var(--text-danger)] hover:bg-red-100 flex items-center justify-center transition-colors shrink-0 mt-0.5"
                                >
                                  <Copy className="w-3 h-3" />
                                </button>
                              </TooltipTrigger>
                              <TooltipContent side="top" className="text-xs">复制错误信息</TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </div>
                      )}
                    </div>
                  </TenantCard>
                );
              })}
            </div>
          )}
        </div>
    </TenantSection>

      {/* ===== 添加 MCP 弹窗 — 步骤1：选择 ===== */}
      <Dialog open={addDialogOpen && !paramTemplate} onOpenChange={setAddDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>添加 MCP</DialogTitle>
            <DialogDescription>从企业已配置的 MCP 中选配</DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" />
              <Input
                tenant
                placeholder="搜索名称或描述..."
                value={addSearchQuery}
                onChange={(e) => setAddSearchQuery(e.target.value)}
                className="pl-9 bg-white"
              />
            </div>
            <div className="max-h-[320px] overflow-y-auto space-y-2">
              {availableTemplates.length === 0 ? (
                <p className="text-sm text-[var(--text-weak)] text-center py-8">
                  {alreadyAddedNames.length === MOCK_ENTERPRISE_MCP_TEMPLATES.length
                    ? "所有企业 MCP 均已添加"
                    : "没有匹配的 MCP"}
                </p>
              ) : (
                availableTemplates.map((tpl) => (
                  <div
                    key={tpl.id}
                    className="flex items-center gap-3 p-3 rounded-[var(--radius-card)] border border-gray-100 hover:border-blue-200 hover:bg-[var(--cp-brand-blue-soft,#EFF6FF)]/30 transition-colors group"
                  >
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-[var(--text-title)] truncate">
                        {tpl.displayName}
                      </p>
                      <p className="text-xs text-[var(--text-muted)] line-clamp-1 mt-0.5">
                        {tpl.description}
                      </p>
                    </div>
                    <button
                      onClick={() => handleSelectTemplate(tpl)}
                      className="w-7 h-7 rounded-md bg-[var(--cp-brand-blue-soft,#EFF6FF)] text-[var(--text-brand)] hover:bg-blue-100 flex items-center justify-center transition-colors shrink-0 opacity-0 group-hover:opacity-100"
                    >
                      <Plus className="w-4 h-4" />
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* ===== 添加 MCP 弹窗 — 步骤2：填写参数 ===== */}
      <Dialog
        open={!!paramTemplate}
        onOpenChange={(open) => {
          if (!open) setParamTemplate(null);
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>填写参数</DialogTitle>
            <DialogDescription>
              「{paramTemplate?.displayName}」需要您填写以下参数
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {paramTemplate?.userRequiredParams.map((param) => (
              <div key={param} className="space-y-1.5">
                <label className="text-sm font-medium text-[var(--text-secondary)]">{param}</label>
                <Input
                  tenant
                  placeholder={`请输入 ${param}`}
                  type="password"
                  value={paramValues[param] || ""}
                  onChange={(e) =>
                    setParamValues((prev) => ({ ...prev, [param]: e.target.value }))
                  }
                />
              </div>
            ))}
          </div>
          <DialogFooter>
            <Button variant="tenant-outline" onClick={() => setParamTemplate(null)}>
              取消
            </Button>
            <Button
              variant="tenant-primary"
              onClick={() => {
                if (!paramTemplate) return;
                const allFilled = paramTemplate.userRequiredParams.every(
                  (p) => (paramValues[p] || "").trim().length > 0
                );
                if (!allFilled) {
                  toast.error("请填写所有必填参数");
                  return;
                }
                doAddMCP(paramTemplate, paramValues, false, paramInjectTarget);
              }}
            >
              确认但不重启
            </Button>
            <Button
              variant="tenant-primary"
              onClick={() => {
                if (!paramTemplate) return;
                const allFilled = paramTemplate.userRequiredParams.every(
                  (p) => (paramValues[p] || "").trim().length > 0
                );
                if (!allFilled) {
                  toast.error("请填写所有必填参数");
                  return;
                }
                doAddMCP(paramTemplate, paramValues, true, paramInjectTarget);
              }}
            >
              确认并重启实例
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 查看源码弹窗 ===== */}
      <Dialog open={sourceDialogOpen} onOpenChange={setSourceDialogOpen}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>查看源码</DialogTitle>
            <DialogDescription>
              名称：{sourceDisplayName}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            {/* 固化外层 + 可编辑 server 内部字段 的编辑器 */}
            <div className="border border-[var(--border)] rounded-[var(--radius-card)] overflow-hidden font-mono text-xs">
              {/* 固定前缀行（不可编辑）— 灰色背景，只显示 "server-name": { */}
              <div className="bg-gray-50 text-[var(--text-weak)] px-3 py-1.5 border-b border-gray-100 select-none leading-relaxed text-xs whitespace-pre">
                <div><span className="text-[var(--text-muted)]">{`"${sourceServerName}"`}</span>{': {'}</div>
              </div>
              {/* 可编辑区域 */}
              <div className="relative">
                {/* 整理缩进按钮 — 悬浮在编辑区右上角 */}
                <button
                        type="button"
                        onClick={() => setSourceEditorContent(trimCommonIndent(sourceEditorContent))}
                        className="absolute top-1.5 right-2 z-10 flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] text-[var(--text-weak)] hover:text-[var(--text-secondary)] hover:bg-[var(--accent)] transition-colors"
                      >
                        <AlignLeft className="w-3 h-3" />
                        整理缩进
                      </button>
                <Textarea
                  value={sourceEditorContent}
                  onChange={(e) => {
                    setSourceEditorContent(e.target.value);
                    setSourceJsonError("");
                  }}
                  placeholder="请输入 server 配置字段"
                  className="border-0 rounded-none font-mono text-xs focus-visible:ring-0 focus-visible:ring-offset-0 focus-visible:border-transparent focus:ring-0 focus:outline-none shadow-none resize-none overflow-auto leading-relaxed min-h-0"
                  style={{ paddingLeft: 'calc(0.75rem + 2ch)', fontSize: '12px', maxHeight: `${10 * 1.625}em` }}
                  spellCheck={false}
                />
              </div>
              {/* 固定后缀行（不可编辑）— 灰色背景 */}
              <div className="bg-gray-50 text-[var(--text-weak)] px-3 py-1.5 border-t border-gray-100 select-none leading-relaxed text-xs whitespace-pre">
                <div>{'}'}</div>
              </div>
            </div>
            {sourceJsonError && (
              <p className="text-xs text-[var(--text-danger)]">{sourceJsonError}</p>
            )}
          </div>
          <DialogFooter className="flex gap-2">
            <Button variant="tenant-outline" onClick={() => setSourceDialogOpen(false)} className="text-sm">
              取消
            </Button>
            <Button variant="tenant-outline" onClick={() => handleSaveSource(false)}>
              保存但不重启
            </Button>
            <Button
              variant="tenant-primary"
              onClick={() => handleSaveSource(true)}
            >
              保存并重启实例
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 删除确认弹窗 ===== */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent className="sm:max-w-sm">
          <AlertDialogHeader>
            <AlertDialogTitle>删除 MCP</AlertDialogTitle>
            <AlertDialogDescription>
              确认删除 <span className="font-medium text-[var(--text-title)]">{deleteMcp?.displayName || deleteMcp?.serverName}</span> MCP，删除后将不可使用。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter className="flex gap-2">
            <AlertDialogCancel className="text-sm">取消</AlertDialogCancel>
            <AlertDialogAction
              variant="tenant-outline"
              className="text-sm"
              onClick={() => handleConfirmDelete(false)}
            >
              删除但不重启
            </AlertDialogAction>
            <AlertDialogAction
              variant="tenant-destructive"
              className="text-sm"
              onClick={() => handleConfirmDelete(true)}
            >
              删除并重启实例
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ===== 重启确认弹窗（仅开关切换时弹出） ===== */}
      <Dialog open={restartDialogOpen} onOpenChange={(open) => { if (!open) handleRestartCancel(); }}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>是否重启以生效？</DialogTitle>
            <DialogDescription>
              修改已保存，需要重启实例后才能生效。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="flex gap-2">
            {restartAction === "toggle" && (
              <Button variant="tenant-outline" onClick={handleRestartCancel} className="text-sm">
                取消修改
              </Button>
            )}
            <Button variant="tenant-outline" onClick={handleRestartLater} className="text-sm">
              暂不重启
            </Button>
            <Button
              variant="tenant-primary"
              onClick={handleRestartNow}
              className="text-sm"
            >
              重启
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
