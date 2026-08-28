import { useMemo, useState } from "react";
import {
  ArrowRight,
  Check,
  ChevronRight,
  CircleAlert,
  Copy,
  FileText,
  Folder,
  FolderOpen,
  Info,
  Keyboard,
  MoreHorizontal,
  Plus,
  RefreshCw,
  Search,
  Settings,
  X,
} from "lucide-react";

import { Button, SmallIconStateButton } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
  InputGroupText,
} from "@/components/ui/input-group";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { Switch } from "@/components/ui/switch";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { RadioCard } from "@/components/ui/radio-card";
import { Label } from "@/components/ui/label";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSeparator,
  FieldSet,
  FieldTitle,
} from "@/components/ui/field";
import { Badge } from "@/components/ui/badge";
import { FilterChip, FilterChipGroup } from "@/components/ui/filter-chip";
import { Progress } from "@/components/ui/progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { DatePicker } from "@/components/ui/date-picker";
import { DateTimePicker } from "@/components/ui/date-time-picker";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
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
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  Drawer,
  DrawerBody,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer";
import {
  Alert,
  AlertDescription,
  AlertErrorIcon,
  AlertInfoIcon,
  AlertOperationInfoIcon,
  AlertProductNewsIcon,
  AlertSuccessIcon,
  AlertTitle,
} from "@/components/ui/alert";
import { AdminNoticeAlert } from "@/components/ui/admin-notice-alert";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { InfoPopover } from "@/components/ui/info-popover";
import { Kbd, KbdGroup } from "@/components/ui/kbd";
import { Spinner } from "@/components/ui/spinner";
import { Stepper } from "@/components/ui/stepper";
import {
  Table,
  TableActionCell,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Pagination } from "@/components/ui/pagination";
import {
  Segment,
  SegmentContent,
  SegmentItem,
  SegmentList,
} from "@/components/ui/segment";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { SegmentedTabs } from "@/components/ui/segmented-tabs";
import { StatusTag } from "@/components/ui/status-tag";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  SurfaceCard,
  SurfaceConfig,
  SurfaceInner,
  SurfaceOverlay,
  TenantCard,
} from "@/components/ui/Surface";
import { TenantSection } from "@/components/ui/TenantSection";
import {
  BodyMedium,
  BodyText,
  CardTitle,
  CodeText,
  CompactText,
  HelperText,
  InlineNumber,
  MetaMedium,
  MetaText,
  MiniBodyText,
  PanelTitle,
  SectionTitle,
  SmallBodyText,
  StatNumber,
  StepText,
  TenantDocTitle,
  TenantHeroTitle,
  TenantPageTitle,
  TinyText,
  UrlText,
} from "@/components/ui/Typography";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import {
  NumberCard,
  RequestsIcon,
  InputTokensIcon,
  OutputTokensIcon,
  TotalTokensIcon,
} from "@/components/ui/number-card";
import DarkVeil from "@/components/ui/DarkVeil";
import {
  AdminSidebarBadge,
  AdminSidebarBrand,
  AdminSidebarFooter,
  AdminSidebarFooterAction,
  AdminSidebarGroupLabel,
  AdminSidebarHeaderAction,
  AdminSidebarLogo,
  AdminSidebarMenu,
  AdminSidebarMenuButton,
  AdminSidebarMenuItem,
  AdminSidebarUser,
  SidebarCollapseIcon,
} from "@/components/ui/admin-sidebar";
import {
  TopNav,
  CenterTabs,
  HelpIcon,
  NavDivider,
  NavIconButton,
  NotificationPanel,
  SwitchAdminIcon,
  UserMenu,
} from "@/components/topnav";
import type { Notification } from "@/components/topnav";
import goTenantArrowIcon from "@/assets/icons/go-tenant-arrow.svg";
import { toast } from "sonner";
import { withClose } from "@/components/ui/sonner";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Transfer } from "@/components/ui/transfer";
import { FileBrowser, type VersionInfo } from "@/components/ui/file-browser";
import type { FileEntry } from "@/components/ui/tree";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle as CardTitleUI,
} from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { LineTabs } from "@/components/ui/line-tabs";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { Slider } from "@/components/ui/slider";
import { Separator } from "@/components/ui/separator";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  ToggleGroup,
  ToggleGroupItem,
} from "@/components/ui/toggle-group";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { AllUsersTag } from "@/components/ui/all-users-tag";
import { BackButton } from "@/components/ui/back-button";
import { FavoriteButton } from "@/components/ui/favorite-button";
import { MoreActionsDropdown } from "@/components/ui/more-actions-dropdown";
import { TreeSelect } from "@/components/ui/tree-select";
import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious,
} from "@/components/ui/carousel";
import { Calendar } from "@/components/ui/calendar";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSeparator,
  InputOTPSlot,
} from "@/components/ui/input-otp";
import { AspectRatio } from "@/components/ui/aspect-ratio";
import {
  NavigationMenu,
  NavigationMenuContent,
  NavigationMenuItem,
  NavigationMenuLink,
  NavigationMenuList,
  NavigationMenuTrigger,
} from "@/components/ui/navigation-menu";
import {
  Menubar,
  MenubarContent,
  MenubarItem,
  MenubarMenu,
  MenubarSeparator,
  MenubarShortcut,
  MenubarTrigger,
} from "@/components/ui/menubar";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { useForm } from "react-hook-form";
import { FilterTrigger } from "@/components/ui/filter-trigger";
import {
  SearchableSelect,
  InstantMultiSelect,
  FilterMultiSelect,
} from "@/components/ui/select";
import { ScopeSelect } from "@/components/ScopeSelect";
import { GroupSelect } from "@/components/GroupSelect";
import { TokenValueEditor } from "@/components/policy/TokenValueEditor";
import type { UserGroup } from "@/pages/admin/MemberManagement/types";
import componentUsage from "./design-system/component-usage.generated.json";

const ADMIN_ICON_BASE = "/assets/admin-sidebar";
const DOCUMENTED_COMPONENT_COUNT = "50+";

const GROUP_LABELS = {
  foundation: "基础视觉",
  action: "操作组件",
  form: "表单组件",
  feedback: "反馈组件",
  data: "数据展示",
  navigation: "导航与布局",
  admin: "管控端专属",
} as const;

type GroupKey = keyof typeof GROUP_LABELS;
type Platform = "Global 全局" | "Tenant 用户端" | "Admin 管控端";
type ComponentId =
  | "color"
  | "typography"
  | "surface-card"
  | "surface-inner"
  | "surface-config"
  | "surface-overlay"
  | "tenant-card"
  | "dark-veil"
  | "number-card"
  | "button"
  | "button-group"
  | "input"
  | "input-group"
  | "textarea"
  | "select"
  | "date-picker"
  | "date-time-picker"
  | "checkbox"
  | "field"
  | "radio-group"
  | "radio-card"
  | "switch"
  | "filter-chip"
  | "alert"
  | "dialog"
  | "alert-dialog"
  | "drawer"
  | "tooltip"
  | "popover"
  | "info-popover"
  | "progress"
  | "spinner"
  | "table"
  | "pagination"
  | "kbd"
  | "badge"
  | "status-tag"
  | "empty"
  | "stepper"
  | "segment"
  | "segmented-tabs"
  | "tabs"
  | "tenant-section"
  | "topnav"
  | "admin-page-header"
  | "admin-sidebar"
  | "toast"
  | "avatar"
  | "tree"
  | "breadcrumb"
  | "transfer"
  | "search-filter-bar"
  | "batch-actions-bar"
  | "chart-stat"
  | "upload"
  | "tag"
  | "accordion"
  | "card"
  | "dropdown-menu"
  | "line-tabs"
  | "sheet"
  | "skeleton"
  | "slider"
  | "separator"
  | "scroll-area"
  | "collapsible"
  | "toggle"
  | "toggle-group"
  | "hover-card"
  | "context-menu"
  | "all-users-tag"
  | "back-button"
  | "favorite-button"
  | "more-actions-dropdown"
  | "tree-select"
  | "carousel"
  | "form"
  | "calendar"
  | "input-otp"
  | "aspect-ratio"
  | "navigation-menu"
  | "menubar"
  | "resizable"
  | "file-browser"
  | "select-panel"
  | "filter-trigger"
  | "filter-panel-suite";

type ApplicationPage = {
  name: string;
  path: string;
  platform: Platform;
  priority: "高" | "中" | "补充";
  usage: string;
};

type ComponentMeta = {
  id: ComponentId;
  group: GroupKey;
  name: string;
  cnName: string;
  description: string;
  owner: string;
  maintainer?: string;
  source: string;
  doc: string;
  platform: Platform;
  adoption: "高频参考" | "核心参考" | "常用" | "专用" | "持续补充中";
  applicationSummary: string;
  applicationScope: string;
  moduleCount: number;
  instanceCount: number;
  tags: string[];
  usage: string[];
  notes: string[];
  migration: string[];
  applicationPages?: ApplicationPage[];
  /** 在展示台 Tab/组织/搜索/计数中隐藏；详情页仍可通过 URL ?id=xxx 直链访问。 */
  hidden?: boolean;
};

type ColorUsageState = "active" | "component" | "candidate" | "alias" | "reserved";

type ColorToken = {
  name: string;
  cssVar?: string;
  className?: string;
  value: string;
  swatch?: string;
  usage: string;
  badges?: string[];
  usageState?: ColorUsageState;
  usageSources?: string[];
};

type ColorGroup = {
  title: string;
  description: string;
  tokens: ColorToken[];
};

const neutralGrayTokens: ColorToken[] = [
  { name: "gray-50", cssVar: "--color-gray-50", className: "bg-gray-50 / text-gray-50", value: "#FAFAFA", usage: "极浅背景" },
  { name: "gray-100", cssVar: "--color-gray-100", className: "bg-gray-100 / text-gray-100", value: "#F5F5F5", usage: "浅背景 / Tab 底" },
  { name: "gray-200", cssVar: "--color-gray-200", className: "bg-gray-200 / border-gray-200", value: "#EAEEF4", usage: "描边 / 分割线" },
  { name: "gray-300", cssVar: "--color-gray-300", className: "bg-gray-300 / border-gray-300", value: "#D4D4D4", usage: "描边强调" },
  { name: "gray-400", cssVar: "--color-gray-400", className: "text-gray-400", value: "#A3A3A3", usage: "极弱文字" },
  { name: "gray-500", cssVar: "--color-gray-500", className: "text-gray-500", value: "#737373", usage: "辅助文字" },
  { name: "gray-600", cssVar: "--color-gray-600", className: "text-gray-600", value: "#475569", usage: "中等文字" },
  { name: "gray-700", cssVar: "--color-gray-700", className: "text-gray-700", value: "#404040", usage: "次级正文" },
  { name: "gray-900", cssVar: "--color-gray-900", className: "text-gray-900", value: "#171717", usage: "主文字 / 正文" },
  { name: "gray-950", cssVar: "--color-gray-950", className: "text-gray-950", value: "#0A0A0A", usage: "强调文字" },
];

const blueGrayTokens: ColorToken[] = [
  { name: "slate-50", className: "bg-slate-50 / text-slate-50", value: "#F8FAFC", usage: "蓝灰极浅背景" },
  { name: "slate-100", className: "bg-slate-100 / text-slate-100", value: "#F1F5F9", usage: "蓝灰浅背景" },
  { name: "slate-200", className: "bg-slate-200 / border-slate-200", value: "#E2E8F0", usage: "蓝灰分割线" },
  { name: "slate-300", className: "bg-slate-300 / border-slate-300", value: "#CBD5E1", usage: "蓝灰描边" },
  { name: "slate-400", className: "text-slate-400", value: "#94A3B8", usage: "蓝灰弱文字" },
  { name: "slate-500", className: "text-slate-500", value: "#64748B", usage: "蓝灰辅助文字" },
  { name: "slate-600", className: "text-slate-600", value: "#475569", usage: "蓝灰中等文字" },
  { name: "slate-700", className: "text-slate-700", value: "#334155", usage: "蓝灰次级正文" },
  { name: "slate-800", className: "text-slate-800", value: "#1E293B", usage: "蓝灰深正文" },
  { name: "slate-900", className: "text-slate-900", value: "#0F172A", usage: "蓝灰强调" },
  { name: "slate-950", cssVar: "--general-foreground", className: "text-slate-950", value: "#020617", usage: "强强调 / CTA 起点" },
];

const textSemanticTokens: ColorToken[] = [
  { name: "text-emphasis", cssVar: "--text-emphasis", className: "text-[var(--text-emphasis)]", value: "#020617", usage: "强强调、关键数字、强标题", badges: ["Typography 使用中"] },
  { name: "text-title", cssVar: "--text-title", className: "text-[var(--text-title)]", value: "#0F172A", usage: "页面标题、模块标题", badges: ["Typography 使用中"] },
  { name: "text-body", cssVar: "--text-body", className: "text-[var(--text-body)]", value: "#1E293B", usage: "普通正文", badges: ["Typography 使用中"] },
  { name: "text-secondary", cssVar: "--text-secondary", className: "text-[var(--text-secondary)]", value: "#334155", usage: "描述、补充说明、表格次要字段", badges: ["Typography 使用中"] },
  { name: "text-muted", cssVar: "--text-muted", className: "text-[var(--text-muted)]", value: "#64748B", usage: "时间、备注、辅助信息", badges: ["Typography 使用中"] },
  { name: "text-weak", cssVar: "--text-weak", className: "text-[var(--text-weak)]", value: "#94A3B8", usage: "占位、空状态、极弱提示", badges: ["Typography 使用中"] },
  { name: "text-brand", cssVar: "--text-brand", className: "text-[var(--text-brand)]", value: "#1447E6", usage: "链接、活跃态、品牌强调", badges: ["Typography 使用中"] },
  { name: "text-danger", cssVar: "--text-danger", className: "text-[var(--text-danger)]", value: "#DC2626", usage: "删除、错误、危险操作", badges: ["Typography 使用中"] },
];

const semanticTokens: ColorToken[] = [
  { name: "background", cssVar: "--background", value: "#FFFFFF", usage: "页面底色" },
  { name: "card", cssVar: "--card", value: "#FFFFFF", usage: "卡片底" },
  { name: "popover", cssVar: "--popover", value: "#FFFFFF", usage: "浮层底" },
  { name: "primary-foreground", cssVar: "--primary-foreground", value: "oklch(0.985 0 0)", swatch: "oklch(0.985 0 0)", usage: "主按钮前景" },
  { name: "destructive-foreground", cssVar: "--destructive-foreground", value: "oklch(0.985 0 0)", swatch: "oklch(0.985 0 0)", usage: "危险按钮前景" },
  { name: "secondary", cssVar: "--secondary", value: "#F5F5F5", usage: "次级背景" },
  { name: "muted", cssVar: "--muted", value: "#F5F5F5", usage: "静默背景" },
  { name: "accent", cssVar: "--accent", value: "#F5F5F5", usage: "Hover 浅背景" },
  { name: "border", cssVar: "--border", value: "#EAEEF4", usage: "描边 / 分割线" },
  { name: "input", cssVar: "--input", value: "#EAEEF4", usage: "输入框描边" },
  { name: "muted-foreground", cssVar: "--muted-foreground", value: "#737373", usage: "辅助文字" },
  { name: "admin-description", cssVar: "--admin-page-description-foreground", value: "#596980", usage: "管控端页头描述" },
  { name: "secondary-foreground", cssVar: "--secondary-foreground", value: "#404040", usage: "次级前景" },
  { name: "foreground", cssVar: "--foreground", value: "#0A0A0A", usage: "主文字" },
  { name: "card-foreground", cssVar: "--card-foreground", value: "#0A0A0A", usage: "卡片文字" },
  { name: "popover-foreground", cssVar: "--popover-foreground", value: "#0A0A0A", usage: "浮层文字" },
  { name: "accent-foreground", cssVar: "--accent-foreground", value: "#0A0A0A", usage: "Hover 前景" },
  { name: "general-foreground", cssVar: "--general-foreground", value: "#020617", usage: "强强调文字" },
];

const brandTokens: ColorToken[] = [
  { name: "blue-500", cssVar: "--color-blue-500", className: "text-blue-500", value: "#355EF1", usage: "Tailwind 蓝色覆盖" },
  { name: "brand-blue", cssVar: "--brand-blue", value: "#1447E6", usage: "品牌主蓝" },
  { name: "brand-purple", cssVar: "--brand-purple", value: "#1447E6", usage: "品牌紫别名" },
  { name: "ring", cssVar: "--ring", value: "#1447E6", usage: "Focus 外环" },
  { name: "primary", cssVar: "--primary", value: "oklch(0.546 0.245 262.881)", swatch: "oklch(0.546 0.245 262.881)", usage: "主色语义" },
  { name: "destructive", cssVar: "--destructive", value: "oklch(0.577 0.245 27.325)", swatch: "oklch(0.577 0.245 27.325)", usage: "危险操作" },
];

const alertTokens: ColorToken[] = [
  { name: "operation-info-bg", cssVar: "--alert-operation-info-bg", value: "#FFFFFF", usage: "操作说明底" },
  { name: "info-bg", cssVar: "--alert-info-bg", value: "#F0F3FC", usage: "Info 底" },
  { name: "product-news-bg", cssVar: "--alert-product-news-bg", value: "var(--alert-info-bg)", swatch: "#F0F3FC", usage: "产品动态底" },
  { name: "operation-info-border", cssVar: "--alert-operation-info-border", value: "#EAEEF4", usage: "操作说明描边" },
  { name: "info-border", cssVar: "--alert-info-border", value: "#BFCFFE", usage: "Info 描边" },
  { name: "product-news-border", cssVar: "--alert-product-news-border", value: "var(--alert-info-border)", swatch: "#BFCFFE", usage: "产品动态描边" },
  { name: "operation-info-icon", cssVar: "--alert-operation-info-icon", value: "var(--text-muted)", swatch: "#64748B", usage: "操作说明图标" },
  { name: "warning-bg", cssVar: "--alert-warning-bg", value: "#FFF7ED", usage: "Warning 底" },
  { name: "warning-border", cssVar: "--alert-warning-border", value: "#FED7AA", usage: "Warning 描边" },
  { name: "warning-icon", cssVar: "--alert-warning-icon", value: "#FF6900", usage: "Warning 图标" },
  { name: "info-icon", cssVar: "--alert-info-icon", value: "#1447E6", usage: "Info 图标" },
  { name: "product-news-icon", cssVar: "--alert-product-news-icon", value: "var(--alert-info-icon)", swatch: "#1447E6", usage: "产品动态图标" },
  { name: "info-foreground", cssVar: "--alert-info-foreground", value: "#0A0A0A", usage: "Alert 文字" },
  { name: "warning-foreground", cssVar: "--alert-warning-foreground", value: "#0A0A0A", usage: "Warning 文字" },
  { name: "operation-info-fg", cssVar: "--alert-operation-info-foreground", value: "var(--alert-info-foreground)", swatch: "#0A0A0A", usage: "操作说明文字" },
  { name: "product-news-fg", cssVar: "--alert-product-news-foreground", value: "var(--alert-info-foreground)", swatch: "#0A0A0A", usage: "产品动态文字" },
];

const chartTokens: ColorToken[] = [
  { name: "chart-5", cssVar: "--chart-5", value: "oklch(0.78 0.12 120)", swatch: "oklch(0.78 0.12 120)", usage: "图表色 5" },
  { name: "chart-4", cssVar: "--chart-4", value: "oklch(0.72 0.15 160)", swatch: "oklch(0.72 0.15 160)", usage: "图表色 4" },
  { name: "chart-3", cssVar: "--chart-3", value: "oklch(0.65 0.18 200)", swatch: "oklch(0.65 0.18 200)", usage: "图表色 3" },
  { name: "chart-1", cssVar: "--chart-1", value: "oklch(0.546 0.245 262.881)", swatch: "oklch(0.546 0.245 262.881)", usage: "图表色 1" },
  { name: "chart-2", cssVar: "--chart-2", value: "oklch(0.48 0.22 280)", swatch: "oklch(0.48 0.22 280)", usage: "图表色 2" },
];

const sidebarTokens: ColorToken[] = [
  { name: "sidebar", cssVar: "--sidebar", value: "#FFFFFF", usage: "侧栏底" },
  { name: "sidebar-primary-fg", cssVar: "--sidebar-primary-foreground", value: "#FFFFFF", usage: "侧栏主色前景" },
  { name: "sidebar-accent", cssVar: "--sidebar-accent", value: "#F5F5F5", usage: "侧栏 Hover 底" },
  { name: "sidebar-border", cssVar: "--sidebar-border", value: "#EAEEF4", usage: "侧栏描边" },
  { name: "sidebar-primary", cssVar: "--sidebar-primary", value: "#1447E6", usage: "侧栏活跃主色" },
  { name: "sidebar-ring", cssVar: "--sidebar-ring", value: "#1447E6", usage: "侧栏 Focus" },
  { name: "sidebar-foreground", cssVar: "--sidebar-foreground", value: "#0A0A0A", usage: "侧栏文字" },
  { name: "sidebar-accent-fg", cssVar: "--sidebar-accent-foreground", value: "#0A0A0A", usage: "侧栏 Hover 文字" },
];

const adminSidebarTokens: ColorToken[] = [
  { name: "admin-sidebar-bg", cssVar: "--admin-sidebar-bg", value: "#FFFFFF", usage: "管控侧栏底" },
  { name: "action-bg", cssVar: "--admin-sidebar-action-bg", value: "#FFFFFF", usage: "侧栏操作按钮底" },
  { name: "action-hover-bg", cssVar: "--admin-sidebar-action-hover-bg", value: "#F5F5F5", usage: "操作按钮 Hover" },
  { name: "badge-bg", cssVar: "--admin-sidebar-badge-bg", value: "#F5F5F5", usage: "侧栏 Badge 底" },
  { name: "item-hover-bg", cssVar: "--admin-sidebar-item-hover-bg", value: "rgba(180, 191, 225, 0.14)", usage: "菜单 Hover 底" },
  { name: "badge-brand-bg", cssVar: "--admin-sidebar-badge-brand-bg", value: "color-mix(in srgb, var(--brand-blue) 10%, var(--admin-sidebar-bg))", usage: "品牌 Badge 底" },
  { name: "avatar-bg", cssVar: "--admin-sidebar-avatar-bg", value: "color-mix(in srgb, var(--brand-blue) 32%, var(--admin-sidebar-bg))", usage: "头像底" },
  { name: "item-active-bg", cssVar: "--admin-sidebar-item-active-bg", value: "linear-gradient(90deg, #E9F3FF 0%, #E3EAFF 100%)", usage: "菜单选中底" },
  { name: "action-border", cssVar: "--admin-sidebar-action-border", value: "#E3E3E3", usage: "操作按钮描边" },
  { name: "action-hover-border", cssVar: "--admin-sidebar-action-hover-border", value: "#E3E3E3", usage: "操作按钮 Hover 描边" },
  { name: "admin-sidebar-border", cssVar: "--admin-sidebar-border", value: "#EAEEF4", usage: "管控侧栏描边" },
  { name: "badge-brand-border", cssVar: "--admin-sidebar-badge-brand-border", value: "color-mix(in srgb, var(--brand-blue) 24%, var(--admin-sidebar-bg))", usage: "品牌 Badge 描边" },
  { name: "admin-sidebar-muted", cssVar: "--admin-sidebar-muted", value: "#737373", usage: "侧栏辅助文字" },
  { name: "avatar-fg", cssVar: "--admin-sidebar-avatar-foreground", value: "#020617", usage: "头像文字" },
  { name: "admin-sidebar-fg", cssVar: "--admin-sidebar-foreground", value: "#0A0A0A", usage: "侧栏正文" },
];

const colorGroups: ColorGroup[] = [
  { title: "Text 文字语义色（--text-*）", description: "Typography 当前使用的文字语义 token，后续页面文字迁移优先收口到这一组。", tokens: textSemanticTokens },
  { title: "中灰色 Gray（当前全局 --color-gray-*）", description: "项目已覆盖的 gray 色阶，保留给历史样式、背景与描边参考。", tokens: neutralGrayTokens },
  { title: "蓝灰色 Slate（替换候选）", description: "Tailwind slate 蓝灰色阶，包含 #334155、#475569、#020617 等候选。", tokens: blueGrayTokens },
  { title: "语义色 Semantic（:root）", description: "shadcn 与全局页面、卡片、浮层、描边、前景等语义 token。", tokens: semanticTokens },
  { title: "品牌 / 交互 Brand", description: "品牌蓝、主色、Focus Ring 与危险操作色。", tokens: brandTokens },
  { title: "Alert 提示", description: "Info / Warning / Operation Info 等提示组件色值。", tokens: alertTokens },
  { title: "Chart 图表", description: "Recharts 等图表使用的全局 chart token。", tokens: chartTokens },
  { title: "Sidebar 侧栏", description: "shadcn sidebar 语义 token。", tokens: sidebarTokens },
  { title: "Admin Sidebar 管控侧栏", description: "管控端侧栏专属色彩 token。", tokens: adminSidebarTokens },
];

type ComponentMetaDraft = Omit<ComponentMeta, "owner" | "doc" | "platform" | "adoption" | "applicationSummary" | "applicationScope" | "moduleCount" | "instanceCount" | "tags" | "usage" | "notes" | "migration"> &
  Partial<Pick<ComponentMeta, "owner" | "doc" | "platform" | "adoption" | "applicationSummary" | "applicationScope" | "moduleCount" | "instanceCount" | "tags" | "usage" | "notes" | "migration">>;

function componentMeta(meta: ComponentMetaDraft): ComponentMeta {
  return {
    owner: "addietang / miekoyychen",
    doc: ".codebuddy/skills/clawpro-portable-design-skill",
    platform: "Global 全局",
    adoption: "持续补充中",
    applicationSummary: `${meta.cnName} 最新真实状态展示，供页面复用和校准。`,
    applicationScope: meta.description,
    moduleCount: 4,
    instanceCount: 8,
    tags: ["新增接入", "最新组件"],
    usage: ["优先复用组件源码导出", "在相同业务形态中保持真实组件状态一致"],
    notes: ["本页直接渲染组件源码，避免用手写样式替代。", "如需扩展状态，优先沉淀到组件源码而不是业务页面。"],
    migration: ["手写临时结构 → 全局组件", "散落样式 → 组件内置 variant / props"],
    ...meta,
  };
}

const COMPONENTS: ComponentMeta[] = [
  {
    id: "color",
    group: "foundation",
    name: "Color",
    cnName: "颜色 Token 色卡",
    description: "集中展示全局颜色 token 色卡，并突出 --text-* 文字语义色。",
    owner: "miekoyychen / addietang",
    source: "client/src/index.css",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · 色彩系统",
    platform: "Global 全局",
    adoption: "核心参考",
    applicationSummary: "全局文字色替换和页面效果校准核心参考。",
    applicationScope: "文字、描边、分割线、浅背景、品牌色、提示色、图表色和侧栏色",
    moduleCount: 97,
    instanceCount: 97,
    tags: ["已接入预览", "核心参考", "Color Token"],
    usage: ["文字语义 token 对照", "中灰色与蓝灰色替换校准", "品牌 / 提示 / 图表 / 侧栏色检查"],
    notes: ["全局 token 主要来自 index.css 的 @theme 与 :root。", "Typography 已切换到 --text-* 文字语义 token。", "后续页面迁移应优先使用 Typography，避免逐页写死文字色。"],
    migration: ["散落文字色 → Typography 或 text-[var(--text-*)]", "中灰描边 / 背景 → 暂不迁移到文字 token", "强强调文字 → --text-emphasis"],
  },
  {
    id: "typography",
    group: "foundation",
    name: "Typography",
    cnName: "字体语义组件",
    description: "统一标题、正文、Meta、数字、代码与步骤文字的语义入口。",
    owner: "miekoyychen / addietang",
    source: "client/src/components/ui/Typography.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Typography 字体组件",
    platform: "Global 全局",
    adoption: "核心参考",
    applicationSummary: "核心文字体系，新增用户端页面优先参考。",
    applicationScope: "页面标题、卡片标题、正文说明、数字、路径与步骤标识",
    moduleCount: 75,
    instanceCount: 893,
    tags: ["已接入预览", "核心参考"],
    usage: ["页面标题、模块标题、卡片标题", "正文说明、辅助信息、时间与 ID", "统计数字、表格数字、代码路径"],
    notes: ["新增页面优先使用 Typography 语义组件。", "Typography tone 已绑定 --text-* 语义 token。", "数字、代码、步骤标识优先使用专用组件。"],
    migration: ["页面标题 → TenantPageTitle / TenantHeroTitle", "卡片标题 → CardTitle", "ID / Token / 路径 → CodeText", "统计大数字 → StatNumber"],
  },
  {
    id: "surface-card",
    group: "foundation",
    name: "SurfaceCard",
    cnName: "表层卡片容器",
    description: "用于页面主区块、列表卡、统计卡和可点击信息卡。",
    owner: "miekoyychen / addietang",
    source: "client/src/components/ui/Surface.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Surface 卡片规范",
    platform: "Global 全局",
    adoption: "高频参考",
    applicationSummary: "卡片容器高频参考，已用于多个主流程页面。",
    applicationScope: "页面主区块、列表卡、统计卡、Agent 卡",
    moduleCount: 42,
    instanceCount: 112,
    tags: ["已接入预览", "高频参考"],
    usage: ["页面主区块", "可点击信息卡", "统计卡片"],
    notes: ["主卡片使用 SurfaceCard，不建议手写卡片阴影。", "需要可点击微动效时使用 hover 属性。"],
    migration: ["手写主卡片 → SurfaceCard", "可点击卡片 → SurfaceCard hover"],
  },
  {
    id: "surface-inner",
    group: "foundation",
    name: "SurfaceInner",
    cnName: "内嵌卡片容器",
    description: "用于卡片内部的子面板、表格容器和组织面板。",
    owner: "miekoyychen / addietang",
    source: "client/src/components/ui/Surface.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Surface 卡片规范",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "常用于卡片内的二级信息组织。",
    applicationScope: "子面板、内嵌表格、组织容器",
    moduleCount: 15,
    instanceCount: 31,
    tags: ["已接入预览", "常用"],
    usage: ["卡片内组织面板", "表格外壳", "二级信息区"],
    notes: ["内嵌卡片强调低层级，不要使用强阴影。", "与 SurfaceCard 嵌套使用时保持信息层级清晰。"],
    migration: ["卡片内子容器 → SurfaceInner", "表格外壳 → SurfaceInner"],
  },
  {
    id: "surface-config",
    group: "foundation",
    name: "SurfaceConfig",
    cnName: "高亮配置卡",
    description: "用于管理端操作要点、引导卡和需要略强存在感的配置卡。",
    owner: "miekoyychen / addietang",
    source: "client/src/components/ui/Surface.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Surface 卡片规范",
    platform: "Admin 管控端",
    adoption: "常用",
    applicationSummary: "管控端配置说明与重点操作区域常用。",
    applicationScope: "操作要点、引导卡、Pro 推荐卡、配置说明区",
    moduleCount: 1,
    instanceCount: 1,
    tags: ["已接入预览", "Admin 管控端"],
    usage: ["管理端配置说明", "重点操作引导", "推荐配置卡"],
    notes: ["用于需要强调的配置卡，不要替代所有普通卡片。", "管控端场景优先参考。"],
    migration: ["强调型配置卡 → SurfaceConfig", "管理端引导卡 → SurfaceConfig"],
  },
  {
    id: "surface-overlay",
    group: "foundation",
    name: "SurfaceOverlay",
    cnName: "浮层容器",
    // 展示台软隐藏：业务侧零引用，作为「自定义浮层」逃生通道保留源码与 lint hint，
    // Tab / 分类 / 计数全部不参与，URL 直链 ?id=surface-overlay 仍能到详情页。
    hidden: true,
    description: "用于自定义浮层容器，Dialog、Popover 等通常已经内置浮层样式。",
    owner: "miekoyychen / addietang",
    source: "client/src/components/ui/Surface.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Surface 卡片规范",
    platform: "Tenant 用户端",
    adoption: "专用",
    applicationSummary: "专用于少量自定义浮层场景。",
    applicationScope: "自定义下拉、浮动说明、自定义菜单外壳",
    moduleCount: 0,
    instanceCount: 0,
    tags: ["已接入预览", "Tenant 用户端", "专用"],
    usage: ["自定义浮层", "自定义菜单", "临时信息面板"],
    notes: ["Dialog / Popover / DropdownMenu 通常不需要再包 SurfaceOverlay。", "仅在自定义浮层外壳时使用。"],
    migration: ["自定义浮层外壳 → SurfaceOverlay"],
  },
  componentMeta({
    id: "tenant-card",
    group: "foundation",
    name: "TenantCard",
    cnName: "用户端业务卡片",
    description: "用户端 Agent 卡、技能卡等 12px 圆角业务列表卡，支持 normal / hover / static 三态。",
    source: "client/src/components/ui/Surface.tsx",
    doc: "SKILL-TENANT.md · TenantCard / Surface.tsx L6",
    platform: "Tenant 用户端",
    adoption: "高频参考",
    applicationScope: "用户端 Agent 卡片、技能卡、入口卡和业务列表卡",
    moduleCount: 7,
    instanceCount: 21,
    tags: ["新增接入", "Tenant 用户端", "卡片三态"],
    usage: ["用户端业务列表卡", "需要 12px 圆角与 Tenant 卡片阴影的场景", "可点击卡片使用 interactive"],
    notes: ["管控端不要使用 TenantCard。", "普通管理端主卡片继续使用 SurfaceCard。", "padding 默认对齐 AgentCard：20px / gap 24px。"],
    migration: ["用户端手写业务卡 → TenantCard", "用户端可点击卡片 → TenantCard interactive", "管理端卡片保持 SurfaceCard"],
  }),
  componentMeta({
    id: "dark-veil",
    group: "foundation",
    name: "DarkVeil",
    cnName: "动态背景",
    description: "管控端开通页 / 能力介绍 hero 的装饰性动态背景（WebGL CPPN 流动纹理）。纯背景组件，不承载信息 / 交互。",
    owner: "miekoyychen",
    source: "client/src/components/ui/DarkVeil.tsx",
    doc: "SKILL-GLOBAL-COMPONENTS.md · DarkVeil 动态背景 / component-specs/dark-veil.md",
    platform: "Admin 管控端",
    adoption: "专用",
    applicationSummary: "开通页 / 能力 hero 顶部的统一动态背景入口，命中 Auto-Trigger 才用，禁止扩散到普通功能页。",
    applicationScope: "管控端功能开通页 / 能力介绍 hero / 首次引导空态的顶部 hero 区局部背景",
    moduleCount: 1,
    instanceCount: 1,
    tags: ["新增接入", "Admin 管控端", "纯背景", "WebGL"],
    usage: [
      "命中 §0 Auto-Trigger（开通页 / 能力 hero + 设计师拍板）时的 hero 动态背景",
      "始终铺「基底 #E0EBFE + DarkVeil + 收束叠层」三件套，内容层 relative z-10",
      "宿主仓无 ogl / WebGL 时按 L0 / L1 / L2 分档兜底，至少做到 L1",
    ],
    notes: [
      "纯装饰：背景层永远 pointer-events-none，无可点击 / 可聚焦元素落在 canvas 上。",
      "参数配方抄 CloudDevActivation.tsx：speed 1.1 / warpAmount 1.1 / noiseIntensity 0.05 / tintColor #B2C3FF。",
      "唯一新依赖 ogl；单页最多 1 个实例；不要改 shader / 清理逻辑。",
      "禁止用于列表 / 表单 / 详情 / 整页背景，禁止用于 Tenant / Landing。",
    ],
    migration: [
      "每个开通页各写一套 canvas / 渐变背景 → 统一 DarkVeil（L0）或 L1 / L2 兜底",
      "无 ogl / WebGL 环境 → 纯 CSS 静态渐变兜底（portable/css/dark-veil.css）",
    ],
    applicationPages: [
      { name: "开通云开发能力", path: "/admin/cloud-dev", platform: "Admin 管控端", priority: "高", usage: "开通页 hero 顶部「基底 + DarkVeil + 收束叠层」动态背景 · 1 处" },
    ],
  }),
  componentMeta({
    id: "number-card",
    group: "data",
    name: "NumberCard",
    cnName: "数字卡片",
    description: "用于展示统计指标（KPI）的标准数字卡片：图标 + 标题 + StatNumber 大数字，可附加进度条/徽标。",
    source: "client/src/components/ui/number-card.tsx",
    doc: "SKILL-GLOBAL-COMPONENTS.md · NumberCard 数字卡片",
    platform: "Global 全局",
    adoption: "高频参考",
    applicationSummary: "概览区数字卡的统一入口，禁止业务再手写 SurfaceCard + StatNumber 拼装。",
    applicationScope: "Tokens 监控、运营 Dashboard、配额消耗、概览统计区",
    moduleCount: 6,
    instanceCount: 12,
    tags: ["新增接入", "概览卡", "数字 KPI"],
    usage: ["Dashboard 顶部概览卡", "Tokens / 请求数 / 成本统计", "配额消耗百分比 + 进度条", "运营指标对比"],
    notes: [
      "图标默认 18×18，使用 GradientIcon（黑→蓝渐变）或内置 4 枚 NumberCardIcon。",
      "数字使用 StatNumber（24px / semibold / tabular-nums），不要再手写字号。",
      "extra 用于数字旁附加进度条/徽标（垂直居中、左右对齐）。",
    ],
    migration: [
      "手写概览卡 SurfaceCard + StatNumber → NumberCard",
      "概览图标的 SVG + radialGradient → GradientIcon 包装器或内置图标",
    ],
  }),
  {
    id: "button",
    group: "action",
    name: "Button",
    cnName: "按钮",
    description: "覆盖主操作、次级操作、弹窗确认、危险操作与表格文字操作。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/button.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Button 组件",
    platform: "Global 全局",
    adoption: "高频参考",
    applicationSummary: "高频参考，已在多个核心页面中使用。",
    applicationScope: "页面主操作、弹窗确认、卡片操作、表格操作",
    moduleCount: 133,
    instanceCount: 960,
    tags: ["已接入预览", "高频参考"],
    usage: ["页面主操作和创建入口", "卡片底部详情、刷新、重试", "弹窗确认、取消、危险操作", "表格操作列文本按钮"],
    notes: ["主操作使用 claw-primary，次级操作使用 claw-outline。", "表格操作列优先使用 link 或 TableActionCell。", "不建议覆盖组件内置颜色、圆角和 disabled 态。"],
    migration: ["手写主按钮 → Button variant=\"claw-primary\"", "手写次级按钮 → Button variant=\"claw-outline\"", "弹窗确认 → Button variant=\"dialog-confirm\"", "表格操作 → TableActionCell + Button"],
  },
  {
    id: "input",
    group: "form",
    name: "Input",
    cnName: "输入框",
    description: "用于单行文本输入、搜索输入和弹窗字段输入。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/input.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Input 组件",
    platform: "Global 全局",
    adoption: "高频参考",
    applicationSummary: "表单场景高频参考，已在配置页和弹窗中广泛使用。",
    applicationScope: "表单输入、搜索筛选、弹窗字段",
    moduleCount: 74,
    instanceCount: 194,
    tags: ["已接入预览", "高频参考"],
    usage: ["配置表单", "搜索输入", "弹窗字段"],
    notes: ["默认白底，不建议额外加灰色底。", "错误态使用 aria-invalid 并配合错误说明。", "弹窗内也必须复用 Input。"],
    migration: ["原生 input → Input", "手写搜索框 → Input + Search icon"],
  },
  componentMeta({
    id: "input-group",
    group: "form",
    name: "InputGroup",
    cnName: "输入组合",
    description: "带前后缀、快捷操作和多行输入的组合输入容器。",
    source: "client/src/components/ui/input-group.tsx",
    adoption: "常用",
    applicationScope: "搜索框前后缀、URL 输入、输入框内快捷按钮",
    moduleCount: 0,
    instanceCount: 0,
    tags: ["新增接入", "表单组件"],
    usage: ["搜索输入前置图标", "URL / Token 输入后置复制操作", "多行文本前置说明"],
    notes: ["内部输入使用 InputGroupInput / InputGroupTextarea。", "不要手写 input 外壳和 focus ring。"],
    migration: ["手写带图标输入框 → InputGroup", "输入框内按钮 → InputGroupButton"],
  }),
  {
    id: "textarea",
    group: "form",
    name: "Textarea",
    cnName: "多行文本域",
    description: "用于说明、备注、配置描述等多行文本输入。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/textarea.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · 其他组件速查",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "常用于配置说明和备注填写。",
    applicationScope: "备注、描述、说明、长文本输入",
    moduleCount: 16,
    instanceCount: 25,
    tags: ["已接入预览", "常用"],
    usage: ["备注填写", "描述输入", "长文本配置"],
    notes: ["视觉状态与 Input 保持一致。", "不要手写不同边框和 focus 态。"],
    migration: ["原生 textarea → Textarea"],
  },

  {
    id: "date-picker",
    group: "form",
    name: "DatePicker",
    cnName: "日期选择",
    description: "用于日期字段选择，基于 Popover 与 Calendar 组合。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/date-picker.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · DatePicker 组件",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "常用于时间筛选和日期配置。",
    applicationScope: "日期筛选、有效期设置、配置时间",
    moduleCount: 6,
    instanceCount: 13,
    tags: ["已接入预览", "常用"],
    usage: ["日期筛选", "有效期配置", "时间范围表单"],
    notes: ["触发器样式与 Input 对齐。", "禁用、hover、focus 状态参考本页真实示例。"],
    migration: ["手写日期触发器 → DatePicker"],
  },
  {
    id: "date-time-picker",
    group: "form",
    name: "DateTimePicker",
    cnName: "日期时间选择",
    description:
      "在 DatePicker 基础上扩展时分（秒）多列选择，复用 react-day-picker 与 Calendar / Popover，底部带预览 + 确定。",
    owner: "addietang",
    source: "client/src/components/ui/date-time-picker.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · DatePicker 组件（DateTimePicker 变体）",
    platform: "Global 全局",
    adoption: "专用",
    applicationSummary: "需要精确到时分或秒的时间点选择场景。",
    applicationScope: "定时任务、生效时间、日志时间点、到秒级配置",
    moduleCount: 1,
    instanceCount: 1,
    tags: ["已接入预览", "新增"],
    usage: ["定时任务时间点", "精确生效时间", "到秒级时间配置"],
    notes: [
      "值格式：YYYY-MM-DD HH:mm；开启 showSeconds 后为 YYYY-MM-DD HH:mm:ss。",
      "右侧时 / 分 /（秒）列选中态与日期同走品牌蓝 #1447E6。",
      "草稿态交互：选完点「确定」才提交 onChange。",
    ],
    migration: ["手写时间选择 → DateTimePicker", "DatePicker + 单独时间输入 → DateTimePicker"],
  },
  {
    id: "checkbox",
    group: "form",
    name: "Checkbox",
    cnName: "复选框",
    description: "用于确认项、多选项和表格选择。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/checkbox.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Checkbox 组件",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "常用于确认、选择和批量操作。",
    applicationScope: "确认勾选、多选、表格选择",
    moduleCount: 42,
    instanceCount: 90,
    tags: ["已接入预览", "常用"],
    usage: ["确认项", "多选配置", "表格选择"],
    notes: ["checked 状态使用品牌蓝。", "Label 与 Checkbox 组合保持可点击区域清晰。"],
    migration: ["手写复选框 → Checkbox"],
  },
  componentMeta({
    id: "field",
    group: "form",
    name: "Field",
    cnName: "表单字段结构",
    description: "统一字段 Label、说明、错误、组织和分隔结构。",
    source: "client/src/components/ui/field.tsx",
    adoption: "常用",
    applicationScope: "表单字段、设置项说明、错误信息和字段组织",
    moduleCount: 1,
    instanceCount: 14,
    tags: ["新增接入", "表单结构"],
    usage: ["表单字段标准结构", "字段错误和说明", "字段集和分隔说明"],
    notes: ["FieldError 用于错误内容，不要只靠红色 placeholder。", "横向布局使用 orientation=\"horizontal\"。"],
    migration: ["散落 Label + Input + p → Field 系列", "手写错误文案 → FieldError"],
  }),
  {
    id: "radio-group",
    group: "form",
    name: "RadioGroup",
    cnName: "单选组",
    description: "用于互斥选项选择。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/radio-group.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · 其他组件速查",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "常用于配置项中的单选模式。",
    applicationScope: "模式选择、类型选择、单选配置",
    moduleCount: 16,
    instanceCount: 24,
    tags: ["已接入预览", "常用"],
    usage: ["互斥选项", "模式选择", "类型选择"],
    notes: ["选项文字使用 Label，确保可读和可点。", "不要用多个 Checkbox 伪装单选。"],
    migration: ["手写单选 → RadioGroup"],
  },
  componentMeta({
    id: "radio-card",
    group: "form",
    name: "RadioCard",
    cnName: "单选卡片",
    description: "配合 RadioGroup 使用的卡片式选项，支持描述和附加内容。",
    source: "client/src/components/ui/radio-card.tsx",
    adoption: "常用",
    applicationScope: "配置模式、套餐类型、安装方式等大块单选项",
    moduleCount: 3,
    instanceCount: 3,
    tags: ["新增接入", "表单组件"],
    usage: ["卡片式单选", "带描述的互斥选项", "配置向导选项"],
    notes: ["必须放在 RadioGroup 内使用。", "checked 状态由业务值传入。"],
    migration: ["手写可选卡片 → RadioCard", "Checkbox 伪装单选 → RadioGroup + RadioCard"],
  }),
  {
    id: "switch",
    group: "form",
    name: "Switch",
    cnName: "开关",
    description: "用于功能开关、配置启停和状态切换。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/switch.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Switch 组件",
    platform: "Global 全局",
    adoption: "高频参考",
    applicationSummary: "开关类配置高频参考。",
    applicationScope: "功能启停、配置开关、状态切换",
    moduleCount: 27,
    instanceCount: 48,
    tags: ["已接入预览", "高频参考"],
    usage: ["功能启停", "配置开关", "状态切换"],
    notes: ["开启色使用品牌蓝。", "不要手写轨道和 thumb 样式。"],
    migration: ["手写开关 → Switch"],
  },
  {
    id: "alert",
    group: "feedback",
    name: "Alert",
    cnName: "提示条",
    description: "用于信息提示、操作说明、警告提示、产品动态和管控端彩色公告条。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/alert.tsx / admin-notice-alert.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Alert 提示组件",
    platform: "Global 全局",
    adoption: "高频参考",
    applicationSummary: "页面提示、操作说明与管控端顶部公告高频参考。",
    applicationScope: "信息提示、操作说明、警告、产品动态、待配置、资源告警",
    moduleCount: 32,
    instanceCount: 72,
    tags: ["已接入预览", "高频参考"],
    usage: ["页面常驻说明", "操作上下文提示", "警告提示", "产品动态通知", "管控端顶部公告"],
    notes: ["普通说明用 info，操作说明用 operation-info。", "warning 标准图标使用 CircleAlert。", "管控端顶部彩色公告用 AdminNoticeAlert，不要替换页面内普通 Alert。"],
    migration: ["手写提示条 → Alert", "管控端顶部公告 → AdminNoticeAlert"],
  },
  {
    id: "dialog",
    group: "feedback",
    name: "Dialog",
    cnName: "弹窗",
    description: "用于普通表单、信息确认和详情查看。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/dialog.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Dialog 组件",
    platform: "Global 全局",
    adoption: "高频参考",
    applicationSummary: "弹窗场景高频参考，常用于表单与确认。",
    applicationScope: "表单弹窗、详情弹窗、普通确认",
    moduleCount: 79,
    instanceCount: 181,
    tags: ["已接入预览", "高频参考"],
    usage: ["表单弹窗", "详情查看", "普通确认"],
    notes: ["危险确认使用 AlertDialog。", "弹窗内 Input / Select / Table 也复用全局组件。"],
    migration: ["手写弹窗 → Dialog", "危险确认 Dialog → AlertDialog"],
  },
  {
    id: "alert-dialog",
    group: "feedback",
    name: "AlertDialog",
    cnName: "危险确认弹窗",
    description: "用于删除、停用等需要确认的危险操作。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/alert-dialog.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · AlertDialog",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "危险操作确认的标准承载。",
    applicationScope: "删除、停用、不可逆操作确认",
    moduleCount: 29,
    instanceCount: 45,
    tags: ["已接入预览", "常用"],
    usage: ["删除确认", "停用确认", "危险操作二次确认"],
    notes: ["危险确认不要使用普通 Dialog。", "确认按钮使用 destructive 语义。"],
    migration: ["危险操作确认 → AlertDialog"],
  },
  {
    id: "drawer",
    group: "admin",
    name: "Drawer",
    cnName: "右侧详情抽屉",
    description: "用于管控端详情查看和局部配置编辑，默认从右侧滑出。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/drawer.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Drawer / 右侧抽屉（管控端详情类）",
    platform: "Admin 管控端",
    adoption: "高频参考",
    applicationSummary: "管控端详情面板高频参考，已用于 Agent 列表详情与监控信息承载。",
    applicationScope: "详情查看、局部配置编辑、高密度信息组织、抽屉内紧凑表格",
    moduleCount: 9,
    instanceCount: 9,
    tags: ["已接入预览", "Admin 管控端", "高频参考"],
    usage: ["Agent 详情查看", "局部配置编辑", "抽屉内紧凑表格", "右侧刷新 / 关闭操作"],
    notes: ["右侧详情抽屉使用 Drawer direction=\"right\"。", "标题使用 DrawerTitle asChild + PanelTitle。", "Body 使用 DrawerBody，隐藏滚动条但保留滚动能力。"],
    migration: ["手写 fixed 右侧面板 → Drawer", "手写抽屉内容滚动区 → DrawerBody", "抽屉内散落文字 class → Typography 语义组件"],
    applicationPages: [
      { name: "Agent 列表", path: "/admin/openclaw-monitor", platform: "Admin 管控端", priority: "高", usage: "Agent 详情、刷新、模型/通道/技能组织信息" },
      { name: "资源管理", path: "/admin/agent-template", platform: "Admin 管控端", priority: "中", usage: "资源模板详情和局部配置编辑" },
      { name: "安全组", path: "/admin/security-group", platform: "Admin 管控端", priority: "补充", usage: "规则详情、操作记录和紧凑信息查看" },
    ],
  },
  {
    id: "tooltip",
    group: "feedback",
    name: "Tooltip",
    cnName: "气泡提示",
    description: "用于短说明、术语解释和禁用原因提示。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/tooltip.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · 其他组件速查",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "轻量说明常用组件。",
    applicationScope: "短说明、禁用原因、术语解释",
    moduleCount: 76,
    instanceCount: 297,
    tags: ["已接入预览", "常用"],
    usage: ["icon 说明", "禁用原因", "字段提示"],
    notes: ["Tooltip 只承载短文案。", "复杂内容使用 Popover。"],
    migration: ["手写 hover 提示 → Tooltip"],
  },
  {
    id: "popover",
    group: "feedback",
    name: "Popover",
    cnName: "气泡浮层",
    description: "用于轻量操作、说明和临时筛选。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/popover.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · 其他组件速查",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "轻量浮层常用组件。",
    applicationScope: "临时筛选、轻量说明、快捷操作",
    moduleCount: 29,
    instanceCount: 47,
    tags: ["已接入预览", "常用"],
    usage: ["临时筛选", "快捷操作", "轻量说明"],
    notes: ["复杂流程不要放在 Popover 中。", "对齐、圆角、阴影以真实组件为准。"],
    migration: ["手写浮层 → Popover"],
  },
  componentMeta({
    id: "info-popover",
    group: "feedback",
    name: "InfoPopover",
    cnName: "说明气泡",
    description: "基于 Popover 的 hover 信息气泡，适合弹窗内字段说明和多行说明。",
    source: "client/src/components/ui/info-popover.tsx",
    adoption: "常用",
    applicationScope: "字段说明、复杂提示、可阅读的 hover 气泡",
    moduleCount: 1,
    instanceCount: 7,
    tags: ["新增接入", "反馈组件"],
    usage: ["弹窗内 Info 说明", "字段规则解释", "需要可停留阅读的短浮层"],
    notes: ["复杂说明优先用 InfoPopover / Popover，不要塞进 Tooltip。", "默认 hover 开合并带关闭延迟。"],
    migration: ["弹窗内 Info Tooltip → InfoPopover", "手写 hover 气泡 → InfoPopover"],
  }),
  {
    id: "progress",
    group: "feedback",
    name: "Progress",
    cnName: "进度条",
    description: "用于任务进度、资源使用比例和加载进度展示。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/progress.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · 进度条",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "任务和资源进度展示常用。",
    applicationScope: "任务进度、资源额度、加载进度",
    moduleCount: 4,
    instanceCount: 7,
    tags: ["已接入预览", "常用"],
    usage: ["资源额度", "任务进度", "加载进度"],
    notes: ["颜色需根据业务语义选择。", "不要手写轨道和填充样式。"],
    migration: ["手写进度条 → Progress"],
  },
  {
    id: "table",
    group: "data",
    name: "Table",
    cnName: "表格",
    description: "统一表头、行高、hover、选中行和数据单元格。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/table.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Table 表格组件规范",
    platform: "Global 全局",
    adoption: "高频参考",
    applicationSummary: "数据列表高频参考，管控端尤其常用。",
    applicationScope: "配置列表、资源列表、弹窗内列表",
    moduleCount: 65,
    instanceCount: 158,
    tags: ["已接入预览", "高频参考"],
    usage: ["管理端配置列表", "资源选择列表", "状态与数量数据展示"],
    notes: ["表格结构统一使用 Table 系列组件。", "分页放在表格容器内部、Table 外部，页面级标准表格通常搭配默认尺寸 Pagination。", "紧凑版使用 density=\"compact\"，仅改变密度，不改变圆角、边框和分割线。"],
    migration: ["原生 table + 自定义 class → Table 系列组件", "高密度表格 → Table density=\"compact\""],
  },
  {
    id: "pagination",
    group: "data",
    name: "Pagination",
    cnName: "分页器",
    description: "统一页面级列表和弹窗内列表的分页展示与交互。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/pagination.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Pagination 分页器规范",
    platform: "Global 全局",
    adoption: "高频参考",
    applicationSummary: "列表分页高频参考。",
    applicationScope: "页面级表格、弹窗列表、简洁翻页",
    moduleCount: 29,
    instanceCount: 38,
    tags: ["已接入预览", "高频参考"],
    usage: ["页面级表格底部分页", "弹窗内资源列表", "简洁翻页"],
    notes: ["页面级标准表格通常使用默认尺寸，弹窗内可按空间使用 small。", "不建议页面内自行实现分页按钮。"],
    migration: ["手写分页按钮 → Pagination"],
  },
  componentMeta({
    id: "kbd",
    group: "data",
    name: "Kbd",
    cnName: "快捷键标签",
    description: "用于键盘快捷键和组合键提示的轻量标签。",
    source: "client/src/components/ui/kbd.tsx",
    adoption: "常用",
    applicationScope: "快捷键提示、命令面板入口、Tooltip 内键位说明",
    moduleCount: 0,
    instanceCount: 0,
    tags: ["新增接入", "数据展示"],
    usage: ["快捷键提示", "输入框后缀", "Tooltip / Popover 中的键位说明"],
    notes: ["组合键使用 KbdGroup。", "不要用普通 Badge 伪装键盘按键。"],
    migration: ["手写 kbd 样式 → Kbd", "多个键位 → KbdGroup"],
  }),
  {
    id: "badge",
    group: "data",
    name: "Badge",
    cnName: "徽章",
    description: "用于轻量标签、状态分类和辅助信息标识。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/badge.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Badge",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "轻量标签常用组件。",
    applicationScope: "分类标签、角标、辅助信息",
    moduleCount: 33,
    instanceCount: 85,
    tags: ["已接入预览", "常用"],
    usage: ["分类标签", "角标", "辅助状态"],
    notes: ["复杂业务状态优先使用 StatusTag。", "不要随意新增 Badge 色值。"],
    migration: ["手写标签 → Badge"],
  },
  {
    id: "status-tag",
    group: "data",
    name: "StatusTag",
    cnName: "状态标签",
    description: "用于运行中、待完成、进行中、异常等状态表达。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/status-tag.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · StatusTag / client/public/research/admin-status-tag-usage-audit.md",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "业务状态表达常用组件；当前管控端 19 个模块、84 处使用。",
    applicationScope: "运行状态、配置状态、任务状态、范围/版本等轻量信息",
    moduleCount: 52,
    instanceCount: 180,
    tags: ["已接入预览", "常用", "Admin 管控端已拉取"],
    usage: ["green：成功 / 开启 / 生效", "blue：进行中 / 全部用户 / 推荐信息", "gray：待处理 / 关闭 / 版本 / 范围", "red：失败 / 异常"],
    notes: ["优先使用已有 green / blue / gray / red 语义色，不要新增随意色。", "mode=\"dot\" 只用于真实状态，不用于版本号、范围、价格等信息标签。", "详细使用清单见 admin-status-tag-usage-audit.md。"],
    migration: ["手写状态胶囊 → StatusTag", "状态类标签使用 mode=\"dot\"", "信息类标签使用 mode=\"fill\""],
    applicationPages: [
      { name: "OpenClaw 监控", path: "/admin/openclaw-monitor", platform: "Admin 管控端", priority: "高", usage: "实例生命周期、异常/处理中状态、模型主备状态 · 12 处" },
      { name: "成员管理", path: "/admin/members", platform: "Admin 管控端", priority: "高", usage: "成员角色、账号状态、组织/配置摘要 · 12 处" },
      { name: "技能配置", path: "/admin/skill-config", platform: "Admin 管控端", priority: "高", usage: "初始技能包、角色设定、版本、已添加、应用范围 · 20 处" },
      { name: "镜像管理", path: "/admin/image-management", platform: "Admin 管控端", priority: "高", usage: "Agent 类型、首选/自定义内核、应用范围 · 7 处" },
      { name: "模型配置", path: "/admin/model-config", platform: "Admin 管控端", priority: "高", usage: "模型应用范围与更多组织数量 · 4 处" },
      { name: "平台策略", path: "/admin/platform-policy", platform: "Admin 管控端", priority: "高", usage: "策略当前值、开关状态和组织规则 · 4 处" },
      { name: "基础信息", path: "/admin/basic-info", platform: "Admin 管控端", priority: "中", usage: "初始化步骤完成态、功能/配置类型标识 · 6 处" },
      { name: "安全组", path: "/admin/security-group", platform: "Admin 管控端", priority: "中", usage: "云端/本地规则启停状态 · 4 处" },
      { name: "审计日志", path: "/admin/audit-log", platform: "Admin 管控端", priority: "中", usage: "请求成功 / 失败结果 · 2 处" },
      { name: "Agent 工具库", path: "/admin/agent-tool-library", platform: "Admin 管控端", priority: "中", usage: "试用中 / 未开通状态 · 2 处" },
      { name: "文件管理", path: "/admin/file-management", platform: "Admin 管控端", priority: "中", usage: "免费、未启用状态 · 2 处" },
      { name: "技能详情", path: "/admin/skill-detail/1", platform: "Admin 管控端", priority: "中", usage: "安全检测状态 · 2 处" },
      { name: "Tokens 监控", path: "/admin/tokens-monitor", platform: "Admin 管控端", priority: "补充", usage: "当前版本标识 · 1 处" },
    ],
  },
  {
    id: "empty",
    group: "data",
    name: "Empty",
    cnName: "空状态",
    description: "用于列表无数据、筛选无结果和初始空场景。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/empty.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · 空状态",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "空场景表达常用组件。",
    applicationScope: "无数据、无结果、初始引导",
    moduleCount: 19,
    instanceCount: 26,
    tags: ["已接入预览", "常用"],
    usage: ["无数据", "搜索无结果", "初始引导"],
    notes: ["空状态应包含说明和下一步行动。", "图标、标题、描述和操作要成组出现。"],
    migration: ["散落空状态 → Empty 系列组件"],
  },
  {
    id: "segment",
    group: "data",
    name: "Segment",
    cnName: "分段选择器",
    description: "用于内容区分类切换和详情页子导航。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/segment.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Segment 分段选择器规范",
    platform: "Global 全局",
    adoption: "核心参考",
    applicationSummary: "内容区分类切换核心参考。",
    applicationScope: "详情页分区、内容分类、局部状态切换",
    moduleCount: 2,
    instanceCount: 3,
    tags: ["已接入预览", "核心参考"],
    usage: ["详情页配置分区", "内容分类", "局部状态切换"],
    notes: ["内容区子分类优先使用 Segment。", "不要散落手写 button 组合表达切换状态。"],
    migration: ["手写分类按钮组 → Segment"],
  },
  {
    id: "tabs",
    group: "data",
    name: "Tabs",
    cnName: "标签页",
    description: "用于页面内轻量标签切换。",
    owner: "addietang / miekoyychen",
    source: "client/src/components/ui/tabs.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · Tabs",
    platform: "Global 全局",
    adoption: "常用",
    applicationSummary: "轻量标签切换常用。",
    applicationScope: "标签页切换、内容面板切换",
    moduleCount: 11,
    instanceCount: 23,
    tags: ["已接入预览", "常用"],
    usage: ["内容面板", "轻量标签", "弹窗内分区"],
    notes: ["Segment 更适合内容区子分类。", "Tabs 用于轻量标签面板。"],
    migration: ["手写 tab → Tabs / Segment"],
  },
  {
    id: "topnav",
    group: "navigation",
    name: "TopNav",
    cnName: "用户端顶部导航",
    description: "用户端顶部导航壳，承载左侧 Logo、中间 Tabs 和右侧功能区。",
    owner: "miekoyychen / addietang",
    maintainer: "jingsujiang / brennali",
    source: "client/src/components/topnav/TopNav.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · 用户端导航栏完整规范",
    platform: "Tenant 用户端",
    adoption: "核心参考",
    applicationSummary: "用户端导航核心参考。",
    applicationScope: "用户端主导航、功能入口、用户菜单承载",
    moduleCount: 2,
    instanceCount: 2,
    tags: ["已接入预览", "Tenant 用户端"],
    usage: ["用户端主导航", "页面 Tab 切换", "右侧功能入口"],
    notes: ["用户端导航采用 1200px 最小宽度策略。", "不要重新拼装顶部导航结构。"],
    migration: ["手写用户端顶部导航 → TopNav"],
  },
  componentMeta({
    id: "admin-page-header",
    group: "admin",
    name: "AdminPageHeader",
    cnName: "管控页头",
    description: "管控端页面标题、描述、标题附件和右侧操作区的统一页头。",
    source: "client/src/components/ui/admin-page-header.tsx",
    platform: "Admin 管控端",
    adoption: "常用",
    applicationScope: "管控端页面标题区、右侧操作按钮和标题辅助信息",
    moduleCount: 24,
    instanceCount: 24,
    tags: ["新增接入", "Admin 管控端"],
    usage: ["管控端页面标题", "标题旁状态或标签", "右侧主次操作按钮"],
    notes: ["不要在每个页面重复手写标题布局。", "标题文字基于 Typography 语义组件。"],
    migration: ["手写管控页头 → AdminPageHeader", "散落标题操作区 → actions 插槽"],
  }),
  {
    id: "admin-sidebar",
    group: "admin",
    name: "AdminSidebar",
    cnName: "管控端侧边栏",
    description: "管控端左侧导航结构，包含品牌区、组织、菜单项和底部用户区。",
    owner: "miekoyychen",
    source: "client/src/components/ui/admin-sidebar.tsx",
    doc: ".codebuddy/skills/clawpro-portable-design-skill · AdminSidebar",
    platform: "Admin 管控端",
    adoption: "核心参考",
    applicationSummary: "管控端导航核心参考。",
    applicationScope: "管控端主导航、组织菜单、收起展开",
    moduleCount: 1,
    instanceCount: 1,
    tags: ["已接入预览", "Admin 管控端"],
    usage: ["管控端主导航", "组织菜单", "收起展开侧栏"],
    notes: ["AdminSidebar 仅用于管控端。", "侧栏 token 由 miekoyychen 维护。", "不要覆盖侧栏内部样式。"],
    migration: ["手写管控端菜单 → AdminSidebar"],
  },
  {
    id: "toast", group: "feedback", name: "Toast", cnName: "消息提示",
    description: "操作反馈顶部弹出通知，基于 sonner。白底统一风格，关闭按钮右侧。",
    owner: "addietang", source: "client/src/components/ui/sonner.tsx", doc: ".codebuddy/skills/clawpro-portable-design-skill §27",
    platform: "Global 全局", adoption: "高频参考", applicationSummary: "全站操作反馈。",
    applicationScope: "表单提交/操作确认/异步任务完成", moduleCount: 1, instanceCount: 2,
    tags: ["已接入预览", "Global 全局"],
    usage: ["操作成功/失败反馈", "异步任务完成通知"],
    notes: ["所有类型统一白底", "关闭按钮必须在右侧", "z-index 99999"],
    migration: ["自行拼装通知 UI → toast API"],
  },
  {
    id: "avatar", group: "data", name: "Avatar", cnName: "头像",
    description: "用户/Agent 头像，4 档标准尺寸，圆形裁切，首字母 Fallback。",
    owner: "addietang", source: "client/src/components/ui/avatar.tsx", doc: ".codebuddy/skills/clawpro-portable-design-skill §22",
    platform: "Global 全局", adoption: "常用", applicationSummary: "用户列表/侧栏/卡片头部。",
    applicationScope: "用户管理/Agent 卡片/会话/侧边栏", moduleCount: 0, instanceCount: 0,
    tags: ["已接入预览", "Global 全局"],
    usage: ["用户头像", "Agent 头像", "侧边栏底部用户区"],
    notes: ["4 档标准尺寸：24/32/40/48px", "不用方形", "无图片时显示首字母"],
    migration: ["自定义尺寸头像 → Avatar 标准 4 档"],
  },
  {
    id: "tree", group: "navigation", name: "Tree", cnName: "树结构",
    description: "层级结构导航，用于组织管理、文件树、目录导航。",
    owner: "addietang", source: "client/src/pages/admin/MemberManagement/GroupList.tsx", doc: ".codebuddy/skills/clawpro-portable-design-skill §13",
    platform: "Admin 管控端", adoption: "常用", applicationSummary: "组织管理/技能目录。",
    applicationScope: "用户组织/技能目录/文件树/网络管理树", moduleCount: 2, instanceCount: 4,
    tags: ["已接入预览", "Admin 管控端"],
    usage: ["组织管理", "文件树", "目录导航"],
    notes: ["行高 32px", "图标统一 #71717a", "缩进 8 + depth × 16"],
    migration: ["自定义树结构 → Tree 标准规范"],
  },
  {
    id: "breadcrumb", group: "navigation", name: "Breadcrumb", cnName: "面包屑",
    description: "页面层级导航，祖先页灰色可点击，当前页深色不可点。",
    owner: "addietang", source: "—", doc: "component-specs/breadcrumb.md",
    platform: "Global 全局", adoption: "常用", applicationSummary: "详情页/多级嵌套页。",
    applicationScope: "详情页/子页面/多级嵌套", moduleCount: 0, instanceCount: 0,
    tags: ["已接入预览", "Global 全局"],
    usage: ["详情页返回导航", "多级嵌套页层级指示"],
    notes: ["仅一级时不显示", "分隔符用 ChevronRight", "当前页不可点击"],
    migration: ["手写面包屑 → Breadcrumb 组件"],
  },
  {
    id: "transfer", group: "form", name: "Transfer", cnName: "穿梭框",
    description: "双面板穿梭选择，替代旧 CvmSelectComponent。instant/batch 两种模式。",
    owner: "addietang", source: "client/src/components/ui/transfer.tsx", doc: ".codebuddy/skills/clawpro-portable-design-skill §31",
    platform: "Admin 管控端", adoption: "常用", applicationSummary: "弹窗内批量选择资产。",
    applicationScope: "批量加策略/生效范围编辑/网络管控选 Agent", moduleCount: 0, instanceCount: 0,
    tags: ["已接入预览", "Admin 管控端"],
    usage: ["批量选择资产", "穿梭框", "生效范围编辑"],
    notes: ["优先 instant 模式", "禁止手搓双 Table + Checkbox", "弹窗内 simple 分页"],
    migration: ["CvmSelectComponent / 双 Table 手搓 → Transfer"],
  },
  {
    id: "search-filter-bar", group: "data", name: "SearchFilterBar", cnName: "搜索筛选条",
    description: "列表页顶部搜索/筛选/刷新工具条，gap-3 排列。",
    owner: "addietang", source: "—", doc: "component-specs/search-filter-bar.md",
    platform: "Global 全局", adoption: "高频参考", applicationSummary: "所有列表页筛选区。",
    applicationScope: "管理端列表页/用户端列表页", moduleCount: 1, instanceCount: 30,
    tags: ["已接入预览", "Global 全局"],
    usage: ["列表页搜索", "状态筛选", "日期筛选", "刷新"],
    notes: ["搜索框左侧 Search 图标", "控件间距 gap-3", "不要每个控件单独写 margin"],
    migration: ["散落的筛选控件 → SearchFilterBar 统一布局"],
  },
  {
    id: "batch-actions-bar", group: "data", name: "BatchActionsBar", cnName: "批量操作条",
    description: "表格多选时底部浮出的批量操作工具条。",
    owner: "addietang", source: "—", doc: "component-specs/batch-actions-bar.md",
    platform: "Admin 管控端", adoption: "常用", applicationSummary: "表格批量操作。",
    applicationScope: "管理端所有表格页/批量删除/批量导出", moduleCount: 1, instanceCount: 10,
    tags: ["已接入预览", "Admin 管控端"],
    usage: ["表格批量操作", "全选/跨页选择", "批量删除/导出"],
    notes: ["sticky bottom", "显示已选数量", "取消选择按钮"],
    migration: ["自写批量操作区 → BatchActionsBar"],
  },
  {
    id: "chart-stat", group: "data", name: "ChartStat", cnName: "图表统计",
    description: "统计数字卡片 + 折线图/环形图展示，DIN 数字字体。",
    owner: "addietang", source: "—", doc: "component-specs/chart-stat.md",
    platform: "Global 全局", adoption: "常用", applicationSummary: "Dashboard/监控页。",
    applicationScope: "TokensMonitor/OpenClawMonitor/Overview", moduleCount: 2, instanceCount: 8,
    tags: ["已接入预览", "Global 全局"],
    usage: ["统计数字展示", "折线图", "环形图"],
    notes: ["数字用 DIN 字体", "StatNumber 组件", "图表用 recharts"],
    migration: ["手写统计卡片 → StatCard + StatNumber"],
  },
  {
    id: "upload", group: "form", name: "Upload", cnName: "上传/文件浏览",
    description: "拖拽上传区 + 文件列表 + 进度展示。",
    owner: "addietang", source: "—", doc: "component-specs/upload-file-browser.md",
    platform: "Global 全局", adoption: "常用", applicationSummary: "文件管理/导入。",
    applicationScope: "文件管理/CSV导入/配置导入", moduleCount: 1, instanceCount: 5,
    tags: ["已接入预览", "Global 全局"],
    usage: ["文件上传", "批量导入", "配置文件上传"],
    notes: ["拖拽区使用 dashed 虚线边框", "文件列表显示名称+大小+进度", "禁止使用默认 Upload 图标"],
    migration: ["自写上传区 → Upload 组件"],
  },
  {
    id: "tag", group: "data", name: "Tag", cnName: "标签",
    description: "用户自定义标签，区别于 StatusBadge（状态标签）。支持基础/可关闭/彩色分类。",
    owner: "addietang", source: "—", doc: "component-specs/tag-label.md",
    platform: "Global 全局", adoption: "常用", applicationSummary: "用户标签/分类标签。",
    applicationScope: "资源标记/分类筛选/用户自建标签", moduleCount: 1, instanceCount: 15,
    tags: ["已接入预览", "Global 全局"],
    usage: ["用户自建标签", "分类标签", "筛选 chip"],
    notes: ["圆角 4px（不是 full）", "区别于 StatusBadge", "彩色从预定义色板选"],
    migration: ["自拼 bg-*-50 text-*-700 标签 → Tag 组件"],
  },
  {
    id: "accordion", group: "navigation", name: "Accordion", cnName: "手风琴",
    description: "可折叠的信息面板，逐项展开/收起，常用于 FAQ、设置组织。",
    owner: "addietang", source: "client/src/components/ui/accordion.tsx", doc: "Radix Accordion",
    platform: "Global 全局", adoption: "常用", applicationSummary: "FAQ/设置组织/折叠详情。",
    applicationScope: "常见问题/配置面板/设置组织", moduleCount: 2, instanceCount: 6,
    tags: ["已接入预览", "Global 全局"],
    usage: ["FAQ 问答列表", "设置项组织折叠", "详情信息收纳"],
    notes: ["同时只展开一项用 type='single'", "多项可展开用 type='multiple'", "动画高度过渡"],
    migration: ["自写折叠面板 → Accordion"],
  },
  {
    id: "card", group: "foundation", name: "Card", cnName: "通用卡片",
    description: "通用卡片容器，含 Header/Content/Footer 三段式结构。",
    owner: "addietang", source: "client/src/components/ui/card.tsx", doc: "Radix Card",
    platform: "Global 全局", adoption: "常用", applicationSummary: "内容区块/表单容器/信息展示。",
    applicationScope: "信息展示/表单容器/内容区块/统计卡片", moduleCount: 5, instanceCount: 20,
    tags: ["已接入预览", "Global 全局"],
    usage: ["内容区块容器", "表单容器", "信息展示卡片"],
    notes: ["与 SurfaceCard 区别：Card 带标题/描述/Footer 语义结构", "阴影和圆角跟随全局 token"],
    migration: ["手写 div 卡片 → Card 标准结构"],
  },
  {
    id: "dropdown-menu", group: "action", name: "DropdownMenu", cnName: "下拉菜单",
    description: "触发式下拉菜单，用于操作列、更多按钮、用户菜单。",
    owner: "addietang", source: "client/src/components/ui/dropdown-menu.tsx", doc: "SKILL-GLOBAL-COMPONENTS.md §17",
    platform: "Global 全局", adoption: "高频参考", applicationSummary: "操作列/更多菜单/用户下拉。",
    applicationScope: "表格操作列/卡片更多/顶栏用户菜单/右键菜单", moduleCount: 15, instanceCount: 45,
    tags: ["已接入预览", "Global 全局"],
    usage: ["表格操作列", "更多操作按钮", "用户菜单"],
    notes: ["菜单项高度 32px", "hover 背景 #F5F5F5", "危险操作项用红色"],
    migration: ["自写下拉 div → DropdownMenu"],
  },
  {
    id: "line-tabs", group: "navigation", name: "LineTabs", cnName: "线性标签页",
    description: "页面标题下方一级 Tab 切换器（下划线式），区别于实心 Tabs。",
    owner: "addietang", source: "client/src/components/ui/line-tabs.tsx", doc: "SKILL-GLOBAL-COMPONENTS.md §11.5",
    platform: "Global 全局", adoption: "高频参考", applicationSummary: "页面一级导航 Tab。",
    applicationScope: "详情页/概览页/设置页 Tab 切换", moduleCount: 8, instanceCount: 16,
    tags: ["已接入预览", "Global 全局"],
    usage: ["页面一级 Tab", "详情页子视图切换"],
    notes: ["下划线式，非实心底色", "容器 border-b border-[#dbe6ff]", "选中项 border-b-2 border-[#1447E6]"],
    migration: ["自写下划线 Tab → LineTabs"],
  },
  {
    id: "sheet", group: "feedback", name: "Sheet", cnName: "侧拉面板",
    description: "从屏幕边缘滑出的面板，用于详情、编辑表单、预览。",
    owner: "addietang", source: "client/src/components/ui/sheet.tsx", doc: "Radix Dialog Sheet",
    platform: "Global 全局", adoption: "常用", applicationSummary: "详情面板/编辑表单/快速预览。",
    applicationScope: "详情侧拉/编辑表单/预览面板", moduleCount: 3, instanceCount: 8,
    tags: ["已接入预览", "Global 全局"],
    usage: ["详情侧拉面板", "编辑表单侧拉", "快速预览"],
    notes: ["默认从右侧滑出", "宽度适配内容（常用 400-600px）", "含遮罩背景"],
    migration: ["自写侧拉 div → Sheet"],
  },
  {
    id: "skeleton", group: "feedback", name: "Skeleton", cnName: "骨架屏",
    description: "内容加载占位，提供页面骨架渲染反馈，避免空白闪烁。",
    owner: "addietang", source: "client/src/components/ui/skeleton.tsx", doc: "UI Skeleton",
    platform: "Global 全局", adoption: "常用", applicationSummary: "加载态占位。",
    applicationScope: "卡片加载/列表加载/图片加载/表格加载", moduleCount: 4, instanceCount: 12,
    tags: ["已接入预览", "Global 全局"],
    usage: ["内容加载占位", "图片加载占位", "列表骨架屏"],
    notes: ["使用 animate-pulse 动画", "尺寸跟随实际内容区块", "颜色 #F5F5F5"],
    migration: ["loading 文字/spinner 替换 → Skeleton 占位"],
  },
  {
    id: "slider", group: "form", name: "Slider", cnName: "滑块",
    description: "拖拽选择数值范围，用于音量、透明度、温度等连续值。",
    owner: "addietang", source: "client/src/components/ui/slider.tsx", doc: "Radix Slider",
    platform: "Global 全局", adoption: "常用", applicationSummary: "数值范围选择。",
    applicationScope: "参数调整/音量设置/温度设置/比例设置", moduleCount: 1, instanceCount: 3,
    tags: ["已接入预览", "Global 全局"],
    usage: ["参数范围调整", "音量/温度控制", "比例/阈值设置"],
    notes: ["轨道高度 4px", "滑块圆形 16px", "支持 range 双滑块"],
    migration: ["input[type=range] → Slider"],
  },
  {
    id: "separator", group: "foundation", name: "Separator", cnName: "分割线",
    description: "水平或垂直分割线，用于内容区域间的视觉分隔。",
    owner: "addietang", source: "client/src/components/ui/separator.tsx", doc: "Radix Separator",
    platform: "Global 全局", adoption: "常用", applicationSummary: "区域分割/视觉断层。",
    applicationScope: "菜单分隔/工具栏分隔/区域分割", moduleCount: 10, instanceCount: 40,
    tags: ["已接入预览", "Global 全局"],
    usage: ["菜单项分隔", "工具栏区域分割", "内容区域断层"],
    notes: ["默认水平，orientation='vertical' 切换", "颜色 border", "间距由父容器控制"],
    migration: ["hr / border-b → Separator"],
  },
  {
    id: "scroll-area", group: "navigation", name: "ScrollArea", cnName: "滚动区域",
    description: "自定义样式的滚动容器，统一滚动条样式，避免浏览器默认滚动条。",
    owner: "addietang", source: "client/src/components/ui/scroll-area.tsx", doc: "Radix ScrollArea",
    platform: "Global 全局", adoption: "常用", applicationSummary: "内容溢出区域。",
    applicationScope: "长列表/侧栏/下拉面板/弹窗内容", moduleCount: 6, instanceCount: 18,
    tags: ["已接入预览", "Global 全局"],
    usage: ["长列表滚动", "侧栏内容滚动", "弹窗内容溢出"],
    notes: ["滚动条 6px 宽", "hover 时显示", "支持水平+垂直滚动"],
    migration: ["overflow-auto → ScrollArea 统一样式"],
  },
  {
    id: "collapsible", group: "navigation", name: "Collapsible", cnName: "折叠面板",
    description: "简单的展开/收起容器，轻量替代 Accordion（单项）。",
    owner: "addietang", source: "client/src/components/ui/collapsible.tsx", doc: "Radix Collapsible",
    platform: "Global 全局", adoption: "常用", applicationSummary: "单项折叠/展开更多。",
    applicationScope: "侧栏组织/高级设置/展开更多", moduleCount: 3, instanceCount: 8,
    tags: ["已接入预览", "Global 全局"],
    usage: ["侧栏组织收起", "高级设置折叠", "查看更多/收起"],
    notes: ["单项折叠用 Collapsible", "多项用 Accordion", "含动画过渡"],
    migration: ["自写 toggle div → Collapsible"],
  },
  {
    id: "toggle-group", group: "action", name: "ToggleGroup", cnName: "切换按钮组",
    description: "一组互斥或多选切换按钮，视图切换/对齐/格式工具栏。",
    owner: "addietang", source: "client/src/components/ui/toggle-group.tsx", doc: "Radix ToggleGroup",
    platform: "Global 全局", adoption: "常用", applicationSummary: "工具栏多选/互斥切换。",
    applicationScope: "工具栏/对齐方式/文字格式/视图选择", moduleCount: 2, instanceCount: 5,
    tags: ["已接入预览", "Global 全局"],
    usage: ["视图模式切换", "对齐方式选择", "文字格式工具栏"],
    notes: ["type='single' 互斥", "type='multiple' 多选", "间距和 Segment 有别"],
    migration: ["自写 radio-like 按钮组 → ToggleGroup"],
  },
  {
    id: "hover-card", group: "feedback", name: "HoverCard", cnName: "悬停卡片",
    description: "鼠标悬停时浮出的信息卡片，用于预览用户/资源详情。",
    owner: "addietang", source: "client/src/components/ui/hover-card.tsx", doc: "Radix HoverCard",
    platform: "Global 全局", adoption: "常用", applicationSummary: "悬停预览信息。",
    applicationScope: "用户头像悬停/链接预览/资源信息卡", moduleCount: 2, instanceCount: 5,
    tags: ["已接入预览", "Global 全局"],
    usage: ["用户信息悬停预览", "链接预览卡", "资源信息预览"],
    notes: ["延迟 200ms 显示", "离开后延迟消失", "用 Portal 避免溢出裁切"],
    migration: ["自写 hover 浮层 → HoverCard"],
  },
  {
    id: "context-menu", group: "action", name: "ContextMenu", cnName: "右键菜单",
    description: "右键触发的上下文菜单，用于表格行/文件/区域的快捷操作。",
    owner: "addietang", source: "client/src/components/ui/context-menu.tsx", doc: "Radix ContextMenu",
    platform: "Global 全局", adoption: "常用", applicationSummary: "右键快捷操作。",
    applicationScope: "文件管理/表格行/画布/编辑器", moduleCount: 1, instanceCount: 3,
    tags: ["已接入预览", "Global 全局"],
    usage: ["文件右键操作", "表格行右键", "画布元素操作"],
    notes: ["菜单样式复用 DropdownMenu", "支持子菜单", "支持快捷键提示"],
    migration: ["自写 onContextMenu → ContextMenu"],
  },
  {
    id: "all-users-tag", group: "data", name: "AllUsersTag", cnName: "全部用户标签",
    description: "管控端'全部用户'展示标签，白底描边统一样式。",
    owner: "addietang", source: "client/src/components/ui/all-users-tag.tsx", doc: "内部规范",
    platform: "Admin 管控端", adoption: "专用", applicationSummary: "用户管理列表标签。",
    applicationScope: "成员管理/安全策略/组织展示", moduleCount: 3, instanceCount: 8,
    tags: ["已接入预览", "Admin 管控端"],
    usage: ["成员列表全部用户标记", "安全策略全局范围标识"],
    notes: ["基于 Badge variant='outline'", "统一白底描边样式"],
    migration: ["手写'全部用户'标签 → AllUsersTag"],
  },
  {
    id: "back-button", group: "navigation", name: "BackButton", cnName: "返回按钮",
    description: "标准返回按钮，用于详情页/子页面回上级。",
    owner: "addietang", source: "client/src/components/ui/back-button.tsx", doc: "内部规范",
    platform: "Global 全局", adoption: "常用", applicationSummary: "详情页返回操作。",
    applicationScope: "所有详情页/子页面/编辑页", moduleCount: 6, instanceCount: 12,
    tags: ["已接入预览", "Global 全局"],
    usage: ["详情页返回上级", "编辑页取消返回"],
    notes: ["左箭头 + 文字", "统一交互行为", "不要自写返回逻辑"],
    migration: ["自写返回链接 → BackButton"],
  },
  {
    id: "favorite-button", group: "action", name: "FavoriteButton", cnName: "收藏按钮",
    description: "收藏/取消收藏切换按钮，心形图标。",
    owner: "addietang", source: "client/src/components/ui/favorite-button.tsx", doc: "内部规范",
    platform: "Global 全局", adoption: "常用", applicationSummary: "列表/卡片收藏。",
    applicationScope: "Agent列表/技能卡片/资源卡片", moduleCount: 2, instanceCount: 6,
    tags: ["已接入预览", "Global 全局"],
    usage: ["Agent 收藏", "技能收藏", "资源收藏"],
    notes: ["选中态填充红色", "hover 心跳动画", "统一交互反馈"],
    migration: ["自写收藏逻辑 → FavoriteButton"],
  },
  {
    id: "more-actions-dropdown", group: "action", name: "MoreActionsDropdown", cnName: "更多操作下拉",
    description: "表格操作列/卡片右上角三点菜单统一封装。",
    owner: "addietang", source: "client/src/components/ui/more-actions-dropdown.tsx", doc: "内部规范",
    platform: "Global 全局", adoption: "高频参考", applicationSummary: "操作列更多菜单。",
    applicationScope: "表格操作列/卡片右上角/列表尾部", moduleCount: 12, instanceCount: 35,
    tags: ["已接入预览", "Global 全局"],
    usage: ["表格操作列三点菜单", "卡片右上角更多", "列表项快捷操作"],
    notes: ["MoreHorizontal 图标", "基于 DropdownMenu 封装", "危险操作用红色"],
    migration: ["自写三点下拉 → MoreActionsDropdown"],
  },
  {
    id: "tree-select", group: "form", name: "TreeSelect", cnName: "树选择器",
    description: "树形单选下拉组件，用于层级数据选择。",
    owner: "addietang", source: "client/src/components/ui/tree-select.tsx", doc: "内部规范",
    platform: "Admin 管控端", adoption: "常用", applicationSummary: "组织选择/目录选择。",
    applicationScope: "组织选择/目录导航/层级关系选择", moduleCount: 2, instanceCount: 5,
    tags: ["已接入预览", "Admin 管控端"],
    usage: ["用户组织选择", "目录选择", "层级关系选择"],
    notes: ["下拉面板内展示树结构", "选中后显示路径", "支持搜索过滤"],
    migration: ["自写树形下拉 → TreeSelect"],
  },
  {
    id: "carousel", group: "data", name: "Carousel", cnName: "轮播",
    description: "内容轮播组件，用于图片展示/功能介绍/引导页。",
    owner: "addietang", source: "client/src/components/ui/carousel.tsx", doc: "embla-carousel",
    platform: "Global 全局", adoption: "常用", applicationSummary: "图片/功能展示。",
    applicationScope: "Landing 页/功能介绍/图片预览", moduleCount: 1, instanceCount: 2,
    tags: ["已接入预览", "Global 全局"],
    usage: ["图片轮播", "功能介绍", "引导页滑动"],
    notes: ["基于 embla-carousel", "支持左右箭头", "支持 dot 指示器"],
    migration: ["自写轮播 → Carousel"],
  },
  {
    id: "form", group: "form", name: "Form", cnName: "表单",
    description: "基于 react-hook-form 的表单容器，统一校验/错误展示。",
    owner: "addietang", source: "client/src/components/ui/form.tsx", doc: "react-hook-form + Radix",
    platform: "Global 全局", adoption: "常用", applicationSummary: "复杂表单容器。",
    applicationScope: "配置表单/新建/编辑弹窗", moduleCount: 5, instanceCount: 15,
    tags: ["已接入预览", "Global 全局"],
    usage: ["复杂配置表单", "新建/编辑弹窗表单", "多步骤表单"],
    notes: ["与 Field 组件配合使用", "校验错误统一红色提示", "支持异步校验"],
    migration: ["散乱 state 表单 → Form + Field 标准结构"],
  },
  {
    id: "calendar", group: "form", name: "Calendar", cnName: "日历",
    description: "日期选择面板，作为 DatePicker 的底层面板组件。",
    owner: "addietang", source: "client/src/components/ui/calendar.tsx", doc: "react-day-picker",
    platform: "Global 全局", adoption: "常用", applicationSummary: "日期选择面板。",
    applicationScope: "DatePicker 内置面板/独立日历展示", moduleCount: 2, instanceCount: 4,
    tags: ["已接入预览", "Global 全局"],
    usage: ["日期选择面板", "日历展示", "时间范围选择"],
    notes: ["基于 react-day-picker", "搭配 DatePicker 使用", "单独使用较少"],
    migration: ["自写日历 → Calendar"],
  },
  {
    id: "input-otp", group: "form", name: "InputOTP", cnName: "验证码输入",
    description: "分格验证码/PIN 输入框，每格一个字符。",
    owner: "addietang", source: "client/src/components/ui/input-otp.tsx", doc: "input-otp",
    platform: "Global 全局", adoption: "常用", applicationSummary: "验证码/PIN 输入。",
    applicationScope: "登录验证/安全验证/邀请码输入", moduleCount: 1, instanceCount: 2,
    tags: ["已接入预览", "Global 全局"],
    usage: ["短信验证码", "PIN 码输入", "邀请码"],
    notes: ["每格 1 字符", "自动聚焦下一格", "支持粘贴填充"],
    migration: ["多个 Input 拼装 → InputOTP"],
  },
  {
    id: "aspect-ratio", group: "foundation", name: "AspectRatio", cnName: "宽高比容器",
    description: "保持内容宽高比的容器，用于图片/视频/嵌入内容。",
    owner: "addietang", source: "client/src/components/ui/aspect-ratio.tsx", doc: "Radix AspectRatio",
    platform: "Global 全局", adoption: "常用", applicationSummary: "媒体内容宽高比。",
    applicationScope: "图片容器/视频容器/卡片封面", moduleCount: 2, instanceCount: 4,
    tags: ["已接入预览", "Global 全局"],
    usage: ["图片 16:9 容器", "视频 4:3 容器", "卡片封面 2:1"],
    notes: ["ratio prop 控制比例", "内容 absolute 填充", "响应式适配"],
    migration: ["padding-top hack → AspectRatio"],
  },
  {
    id: "navigation-menu", group: "navigation", name: "NavigationMenu", cnName: "导航菜单",
    description: "水平导航菜单，适用于顶栏多级导航。",
    owner: "addietang", source: "client/src/components/ui/navigation-menu.tsx", doc: "Radix NavigationMenu",
    platform: "Global 全局", adoption: "常用", applicationSummary: "顶栏多级导航。",
    applicationScope: "顶栏导航/产品切换/多级菜单", moduleCount: 1, instanceCount: 2,
    tags: ["已接入预览", "Global 全局"],
    usage: ["顶栏多级导航", "产品/模块切换"],
    notes: ["支持下拉子菜单", "hover 或 click 触发", "含箭头指示"],
    migration: ["自写导航 → NavigationMenu"],
  },
  {
    id: "menubar", group: "navigation", name: "Menubar", cnName: "菜单栏",
    description: "桌面应用风格的顶部菜单栏（文件/编辑/视图…）。",
    owner: "addietang", source: "client/src/components/ui/menubar.tsx", doc: "Radix Menubar",
    platform: "Global 全局", adoption: "常用", applicationSummary: "桌面风格菜单栏。",
    applicationScope: "编辑器/IDE 风格菜单/桌面 App", moduleCount: 0, instanceCount: 1,
    tags: ["已接入预览", "Global 全局"],
    usage: ["编辑器菜单栏", "桌面应用菜单", "工具栏菜单"],
    notes: ["水平排列菜单项", "每项含子下拉", "支持快捷键提示"],
    migration: ["自写菜单栏 → Menubar"],
  },
  {
    id: "resizable", group: "navigation", name: "Resizable", cnName: "可调节面板",
    description: "可拖拽调节大小的面板布局，用于侧栏/编辑器/预览。",
    owner: "addietang", source: "client/src/components/ui/resizable.tsx", doc: "react-resizable-panels",
    platform: "Global 全局", adoption: "常用", applicationSummary: "可调节布局面板。",
    applicationScope: "编辑器布局/侧栏调节/预览面板", moduleCount: 1, instanceCount: 2,
    tags: ["已接入预览", "Global 全局"],
    usage: ["侧栏宽度拖拽调节", "编辑器面板分割", "预览区域调节"],
    notes: ["基于 react-resizable-panels", "拖拽手柄垂直居中", "min/max 约束"],
    migration: ["CSS resize → Resizable"],
  },
  {
    id: "file-browser", group: "data", name: "FileBrowser", cnName: "文件浏览器",
    description: "多版本资产包只读浏览器：版本列表 + 文件树 + 内容查看（Preview/Source）。",
    owner: "addietang", source: "client/src/components/ui/file-browser.tsx", doc: "component-specs/file-browser.md",
    platform: "Admin 管控端", adoption: "专用", applicationSummary: "Skill / Plugin / MCP 详情页文件浏览 Tab。",
    applicationScope: "资产包多版本浏览（Skill/Plugin/MCP）", moduleCount: 3, instanceCount: 3,
    tags: ["已接入预览", "Admin 管控端"],
    usage: ["Skill 详情", "Plugin 详情", "MCP 详情"],
    notes: ["三栏布局 14% / 22% / flex-1", "默认优先选 SKILL.md", ".md/.mdx 自动 Preview", "可选 showDownload"],
    migration: ["自写版本+文件树+内容三栏 → FileBrowser"],
  },

  {
    id: "filter-trigger", group: "form", name: "FilterTrigger", cnName: "筛选触发器",
    description: "筛选触发器统一组件，三种变体：button/chip/text。",
    owner: "addietang", source: "client/src/components/ui/filter-trigger.tsx", doc: "SKILL-GLOBAL-COMPONENTS.md §6",
    platform: "Global 全局", adoption: "高频参考", applicationSummary: "筛选入口。",
    applicationScope: "列表页筛选区/工具栏/搜索条", moduleCount: 8, instanceCount: 25,
    tags: ["已接入预览", "Global 全局"],
    usage: ["列表筛选按钮", "条件筛选 chip", "文字链筛选"],
    notes: ["button 变体仿 Select 样式", "chip 变体 Filter Chip", "text 变体纯文字链"],
    migration: ["自写筛选按钮 → FilterTrigger"],
  },
  {
    id: "filter-panel-suite", group: "form", name: "Select", cnName: "筛选面板组件套件",
    description: "筛选面板组件全景：含 SelectPanel（三段式骨架）、TreeSelect（树形选择）及可合并 ×3 组 / 独立 ×3 组件，集中预览于同一页。",
    owner: "addietang", source: "client/src/components/ui/select.tsx · select-panel.tsx · tree-select.tsx · ScopeSelect/GroupSelect", doc: "SKILL-GLOBAL-COMPONENTS.md §6",
    platform: "Global 全局", adoption: "高频参考", applicationSummary: "筛选/选择面板组件总览。",
    applicationScope: "所有列表页筛选区/选择面板/范围编辑/树形选择", moduleCount: 27, instanceCount: 80,
    tags: ["已接入预览", "Global 全局"],
    usage: ["列表页筛选条件", "范围/分组选择", "配额值编辑", "表格列头筛选"],
    notes: [
      "TreeSelect（树形·单选·确认）：triggerVariant=button 用于页面，filter-icon 用于表头列筛选",
      "ScopeSelect（范围选择）：mode=instant 即时含触发器，mode=confirm 确认树",
      "独立组件：GroupSelect（多选·即时·树）/ TokenValueEditor（单选·确认·扁平），数据结构异构不参与合并",
    ],
    migration: ["散乱自写筛选下拉 → 按「单选/多选 × 即时/确认 × 扁平/树」数据结构选用对应组件"],
  },
];

/* ───────────── Admin 管控端 · 36 个组件规范组织 ─────────────
 * 与 .codebuddy/skills/clawpro-portable-design-skill/component-specs/*.md
 * 严格 1:1 对齐（共 36 份 spec）。Admin 管控端筛选 tab 下的左侧侧栏将
 * 按本数组的顺序与组织渲染，每个 spec 关联若干已注册的 ComponentId，
 * 点击卡片仍打开对应组件的真实预览与全状态展示。
 */
type AdminSpecGroup = {
  /** spec 文件名（不带 .md，等于 component-specs 子文件名） */
  id: string;
  /** 英文/PascalCase 名称（侧栏组织标题） */
  name: string;
  /** 中文名 */
  cnName: string;
  /** spec 文档相对路径 */
  doc: string;
  /** 关联的现有 ComponentId 列表（按推荐展示顺序） */
  components: ComponentId[];
};

const ADMIN_SPEC_GROUPS: AdminSpecGroup[] = [
  { id: "admin-sidebar", name: "AdminSidebar", cnName: "管控侧栏", doc: "component-specs/admin-sidebar.md", components: ["admin-sidebar"] },
  { id: "alert", name: "Alert", cnName: "提示条", doc: "component-specs/alert.md", components: ["alert"] },
  { id: "avatar", name: "Avatar", cnName: "头像", doc: "component-specs/avatar.md", components: ["avatar"] },
  { id: "badge", name: "Badge", cnName: "徽标", doc: "component-specs/badge.md", components: ["badge"] },
  { id: "batch-actions-bar", name: "BatchActionsBar", cnName: "批量操作条", doc: "component-specs/batch-actions-bar.md", components: ["batch-actions-bar"] },
  { id: "breadcrumb", name: "Breadcrumb", cnName: "面包屑", doc: "component-specs/breadcrumb.md", components: ["breadcrumb"] },
  { id: "button", name: "Button", cnName: "按钮", doc: "component-specs/button.md", components: ["button", "button-group"] },
  { id: "card-surface", name: "Card / Surface", cnName: "卡片与表层", doc: "component-specs/card-surface.md", components: ["surface-card", "surface-inner", "surface-config", "surface-overlay", "tenant-card", "card"] },
  { id: "chart-stat", name: "ChartStat", cnName: "图表统计", doc: "component-specs/chart-stat.md", components: ["chart-stat"] },
  { id: "data-table", name: "DataTable", cnName: "数据表格", doc: "component-specs/data-table.md", components: ["table"] },
  { id: "date-picker", name: "DatePicker", cnName: "日期选择", doc: "component-specs/date-picker.md", components: ["date-picker", "date-time-picker", "calendar"] },
  { id: "dialog-drawer", name: "Dialog & Drawer", cnName: "弹窗与抽屉", doc: "component-specs/dialog-drawer.md", components: ["dialog", "alert-dialog", "drawer", "sheet"] },
  { id: "empty-state", name: "EmptyState", cnName: "空状态", doc: "component-specs/empty-state.md", components: ["empty"] },
  { id: "file-browser", name: "FileBrowser", cnName: "文件浏览器", doc: "component-specs/file-browser.md", components: ["file-browser"] },
  { id: "form-controls", name: "FormControls", cnName: "表单容器", doc: "component-specs/form-controls.md", components: ["field", "form"] },
  { id: "input-select", name: "Input & Select", cnName: "输入与选择", doc: "component-specs/input-select.md", components: ["input", "input-group", "textarea", "select", "select-panel", "input-otp"] },
  { id: "loading-progress", name: "Loading & Progress", cnName: "加载与进度", doc: "component-specs/loading-progress.md", components: ["progress", "spinner", "skeleton"] },
  { id: "number-card", name: "NumberCard", cnName: "数字卡片", doc: "component-specs/number-card.md", components: ["number-card"] },
  { id: "page-header", name: "PageHeader", cnName: "页头", doc: "component-specs/page-header.md", components: ["admin-page-header"] },
  { id: "pagination", name: "Pagination", cnName: "分页器", doc: "component-specs/pagination.md", components: ["pagination"] },
  { id: "popover-dropdown-menu", name: "Popover & Menu", cnName: "浮层与菜单", doc: "component-specs/popover-dropdown-menu.md", components: ["popover", "info-popover", "dropdown-menu", "hover-card", "context-menu", "menubar", "navigation-menu"] },
  { id: "search-filter-bar", name: "SearchFilterBar", cnName: "搜索筛选条", doc: "component-specs/search-filter-bar.md", components: ["search-filter-bar", "filter-trigger", "filter-chip", "filter-panel-suite"] },
  { id: "segment", name: "Segment", cnName: "分段控件", doc: "component-specs/segment.md", components: ["segment", "segmented-tabs"] },
  { id: "selection-controls", name: "SelectionControls", cnName: "选择控件", doc: "component-specs/selection-controls.md", components: ["checkbox", "radio-group", "radio-card", "switch", "toggle", "toggle-group"] },
  { id: "status-tag", name: "StatusTag", cnName: "状态标签", doc: "component-specs/status-tag.md", components: ["status-tag"] },
  { id: "table", name: "Table", cnName: "表格", doc: "component-specs/table.md", components: ["table"] },
  { id: "tabs", name: "Tabs", cnName: "标签页", doc: "component-specs/tabs.md", components: ["tabs", "line-tabs"] },
  { id: "tag-label", name: "Tag & Label", cnName: "标签", doc: "component-specs/tag-label.md", components: ["tag", "all-users-tag", "kbd"] },
  { id: "tenant-topnav", name: "TenantTopnav", cnName: "顶部导航", doc: "component-specs/tenant-topnav.md", components: ["topnav", "tenant-section"] },
  { id: "toast", name: "Toast", cnName: "消息提示", doc: "component-specs/toast.md", components: ["toast"] },
  { id: "tooltip", name: "Tooltip", cnName: "文字提示", doc: "component-specs/tooltip.md", components: ["tooltip"] },
  { id: "transfer", name: "Transfer", cnName: "穿梭框", doc: "component-specs/transfer.md", components: ["transfer", "tree-select"] },
  { id: "tree", name: "Tree", cnName: "树结构", doc: "component-specs/tree.md", components: ["tree"] },
  { id: "typography", name: "Typography", cnName: "字体", doc: "component-specs/typography.md", components: ["typography"] },
  { id: "upload-file-browser", name: "Upload & FileBrowser", cnName: "上传与文件", doc: "component-specs/upload-file-browser.md", components: ["upload", "file-browser"] },
];

/**
 * Page Reference · 典型页面样例
 * 数据源：.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/*.md
 * 截图副本：client/public/page-references/*.png（由 .codebuddy 同步而来）
 */
type PageReference = {
  id: string;
  name: string;
  cnName: string;
  category: string;
  route: string;
  source: string;
  spec: string;
  screenshot: string;
  description: string;
  whyTypical: string;
  keyComponents: string[];
};

const PAGE_REFERENCES: PageReference[] = [
  {
    id: "admin-channel-config",
    name: "ChannelConfig",
    cnName: "通道配置",
    category: "配置页",
    route: "/admin/channel-config",
    source: "client/src/pages/admin/ChannelConfig.tsx",
    spec: "assets/page-references/admin-channel-config.md",
    screenshot: "/page-references/admin-channel-config.png",
    description: "Tabs + 描述 + 行内 Switch 表格的最小配置页骨架，复用度最高。",
    whyTypical: "配置页里最常被复用的骨架：组织(Tabs) → 描述 → 表格。",
    keyComponents: ["Alert", "AdminPageHeader", "LineTabs", "Table", "Switch", "Dialog"],
  },
  {
    id: "admin-members",
    name: "MemberManagement",
    cnName: "用户管理",
    category: "列表页（基础）",
    route: "/admin/members",
    source: "client/src/pages/admin/MemberManagement.tsx",
    spec: "assets/page-references/admin-members.md",
    screenshot: "/page-references/admin-members.png",
    description: "PageHeader + Segment + Search + 工具栏 + Table + Pagination 标准管理列表。",
    whyTypical: "标准管理列表页的完整版，覆盖筛选 / 搜索 / 表格 / 行内操作 / 分页。",
    keyComponents: ["AdminPageHeader", "Segment", "Input", "Button", "Table", "Badge", "StatusTag", "Pagination"],
  },
  {
    id: "admin-openclaw-monitor",
    name: "OpenClawMonitor",
    cnName: "Agent 列表",
    category: "列表页（富功能）",
    route: "/admin/openclaw-monitor",
    source: "client/src/pages/admin/OpenClawMonitor.tsx",
    spec: "assets/page-references/admin-openclaw-monitor.md",
    screenshot: "/page-references/admin-openclaw-monitor.png",
    description: "AlertBanner + 4 NumberCard 状态筛选 + 列头筛选表格 + 行操作 + 抽屉的运维型列表。",
    whyTypical: "ClawPro 最完整的列表页全家桶模板：状态统计 / 搜索 / 批量 / 命令面板 / 列头筛选 / 行操作 / 抽屉详情七大模式。",
    keyComponents: ["Alert", "AdminPageHeader", "NumberCard", "Input", "DropdownMenu", "Popover", "StatusTag", "Table", "Drawer"],
  },
  {
    id: "admin-agent-template",
    name: "ResourceManagement",
    cnName: "资源管理（即将开放）",
    category: "空页面",
    route: "/admin/agent-template",
    source: "client/src/pages/admin/ResourceManagement.tsx",
    spec: "assets/page-references/admin-agent-template.md",
    screenshot: "/page-references/admin-agent-template.png",
    description: "整页 EmptyState：占位插画 + 标题 + 描述，不引入 Empty 卡片描边。",
    whyTypical: "「即将开放 / 暂无入口」类页面的最小集，35 行源码即可上线。",
    keyComponents: ["TenantPageTitle", "BodyText", "占位插画"],
  },
  {
    id: "admin-security-management",
    name: "SecurityManagement",
    cnName: "AI Agent 安全",
    category: "复杂列表页",
    route: "/admin/security-management",
    source: "client/src/pages/admin/Security/index.tsx",
    spec: "assets/page-references/admin-security-management.md",
    screenshot: "/page-references/admin-security-management.png",
    description: "PageHeader + 3 NumberCard + LineTabs + 子表区(工具栏 + Empty/Table) 的多层级骨架。",
    whyTypical: "唯一一个统计 + 多 Tab + 子列表 + 工具栏 + 空态全套组合的页面。",
    keyComponents: ["AdminPageHeader", "NumberCard", "LineTabs", "Tooltip", "Button", "Empty", "Table"],
  },
  {
    id: "admin-tokens-monitor",
    name: "TokensMonitor",
    cnName: "Tokens 监控",
    category: "数据看板",
    route: "/admin/tokens-monitor",
    source: "client/src/pages/admin/TokensMonitor.tsx",
    spec: "assets/page-references/admin-tokens-monitor.md",
    screenshot: "/page-references/admin-tokens-monitor.png",
    description: "5 NumberCard 大盘 + LineChart 趋势 + LineTabs 多维度切换明细的数据看板。",
    whyTypical: "ClawPro 管控端最复杂的数据看板模板：大盘卡 + 时序图 + 多维度切换明细三段式齐全。",
    keyComponents: ["Alert", "AdminPageHeader", "DatePicker", "NumberCard", "SurfaceCard", "LineChart", "LineTabs", "Table"],
  },
  {
    id: "admin-ops-observation",
    name: "OpsObservation",
    cnName: "运维观测（未开通态）",
    category: "服务开通引导",
    route: "/admin/ops-observation",
    source: "client/src/pages/admin/OpsObservation.tsx",
    spec: "assets/page-references/admin-ops-observation.md",
    screenshot: "/page-references/admin-ops-observation.png",
    description: "AlertBanner + PageHeader + 横长 CTA banner + 章节化 Feature 双列预览的服务开通引导页。",
    whyTypical: "服务未开通 / 引导开通态的标准模板，与传统空状态页互补：让用户提前看到价值再决策。",
    keyComponents: ["Alert", "AdminPageHeader", "DatePicker", "SurfaceCard", "Button", "FeatureCell"],
  },
];

function StatCard({ label, value, hint }: { label: string; value: string; hint: string }) {
  return (
    <div className="rounded-[4px] border border-[#DDE7F2] bg-white px-4 py-3">
      <span className="text-xs text-[#737373]">{label}</span>
      <div className="mt-2 flex items-end gap-2">
        <span className="font-din text-2xl font-bold leading-none tabular-nums text-[#020617]">{value}</span>
        <span className="pb-0.5 text-xs text-[#737373]">{hint}</span>
      </div>
    </div>
  );
}

function DetailSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="space-y-3">
      <PanelTitle>{title}</PanelTitle>
      {children}
    </section>
  );
}

function GuidanceBlock({
  title,
  items,
  variant,
}: {
  title: string;
  items: string[];
  variant: "usage" | "notice" | "migration";
}) {
  const config = {
    usage: { text: "text-[#1447E6]" },
    notice: { text: "text-[#B8640A]" },
    migration: { text: "text-[#334155]" },
  }[variant];

  return (
    <div className="min-w-0 border-t border-[#EAF1F8] pt-3">
      <div className="mb-2 flex items-center">
        <BodyMedium>{title}</BodyMedium>
      </div>
      <ul className="grid gap-1.5">
        {items.map((item, index) => (
          <li key={item} className="grid grid-cols-[14px_minmax(0,1fr)] items-start gap-1.5">
            <span className={`block h-5 text-[11px] font-medium leading-5 tabular-nums ${config.text}`}>{String(index + 1).padStart(2, "0")}</span>
            <MetaText as="span" tone="secondary" className="leading-5">{item}</MetaText>
          </li>
        ))}
      </ul>
    </div>
  );
}

function PreviewPanel({
  title,
  children,
  layout = "center",
}: {
  title: string;
  children: React.ReactNode;
  layout?: "center" | "wide";
}) {
  return (
    <div className="relative mt-3 rounded-[4px] border border-[#DDE7F2] bg-white">
      <div className="absolute -top-3 left-6 bg-white px-2">
        <BodyMedium>{title}</BodyMedium>
      </div>
      <div className="flex min-h-[340px] items-center justify-center p-10 pt-12">
        <div className={layout === "wide" ? "mx-auto w-full max-w-[960px]" : "mx-auto w-fit max-w-full"}>{children}</div>
      </div>
    </div>
  );
}

function isDarkColor(value: string) {
  const hex = value.trim().match(/^#([0-9a-f]{6})$/i)?.[1];
  if (!hex) return false;
  const r = Number.parseInt(hex.slice(0, 2), 16);
  const g = Number.parseInt(hex.slice(2, 4), 16);
  const b = Number.parseInt(hex.slice(4, 6), 16);
  return (r * 299 + g * 587 + b * 114) / 1000 < 118;
}

const activeSlateValues = new Set(["#F8FAFC", "#E2E8F0", "#94A3B8", "#64748B", "#334155", "#1E293B", "#0F172A", "#020617"]);

const colorUsageToneClasses: Record<ColorUsageState, string> = {
  active: "border-[#BFCFFE] bg-[#F0F3FC] text-[#1447E6]",
  component: "border-[#DDE7F2] bg-[#F8FAFC] text-[#334155]",
  candidate: "border-[#EAEEF4] bg-white text-[#737373]",
  alias: "border-[#FED7AA] bg-[#FFF7ED] text-[#B8640A]",
  reserved: "border-[#EAEEF4] bg-[#FAFAFA] text-[#737373]",
};

function getColorTokenUsageMeta(token: ColorToken, groupTitle: string) {
  const rawValue = (token.swatch ?? token.value).toUpperCase();
  if (token.usageState) {
    return {
      state: token.usageState,
      label: token.badges?.[0] ?? "自定义使用状态",
      badges: token.badges ?? ["自定义"],
      sources: token.usageSources ?? [],
    };
  }
  if (groupTitle.includes("Text")) {
    return {
      state: "active" as const,
      label: "Typography 与已迁移页面正在使用",
      badges: token.badges ?? ["Typography 使用中"],
      sources: ["Typography.tsx", "text-[var(--text-*)]"],
    };
  }
  if (groupTitle.includes("Gray")) {
    return {
      state: "component" as const,
      label: "全局 Tailwind 覆盖与存量页面使用",
      badges: ["基础色阶"],
      sources: ["index.css @theme", "gray-* utilities"],
    };
  }
  if (groupTitle.includes("Slate")) {
    const isActive = activeSlateValues.has(rawValue);
    return {
      state: isActive ? "active" as const : "candidate" as const,
      label: isActive ? "已映射到文字语义或组件色值" : "替换候选，暂未作为主入口",
      badges: [isActive ? "已映射" : "替换候选"],
      sources: isActive ? ["--text-*", "组件校准色"] : ["候选色阶"],
    };
  }
  if (groupTitle.includes("Semantic")) {
    return {
      state: "active" as const,
      label: "shadcn / 全局语义变量使用中",
      badges: ["语义使用"],
      sources: ["@theme inline", "ui/* components"],
    };
  }
  if (groupTitle.includes("Brand")) {
    const isAlias = token.name === "brand-purple";
    return {
      state: isAlias ? "alias" as const : "active" as const,
      label: isAlias ? "兼容旧命名，实际同 brand-blue" : "品牌主色 / 交互状态使用中",
      badges: [isAlias ? "兼容别名" : "品牌使用"],
      sources: isAlias ? ["legacy alias"] : ["Button", "Focus Ring", "Active 状态"],
    };
  }
  if (groupTitle.includes("Alert")) {
    return {
      state: "active" as const,
      label: "Alert / AdminNoticeAlert 使用中",
      badges: ["Alert 使用中"],
      sources: ["alert.tsx", "admin-notice-alert.tsx"],
    };
  }
  if (groupTitle.includes("Chart")) {
    return {
      state: "reserved" as const,
      label: "图表语义 token，按 Recharts 场景预留",
      badges: ["图表预留"],
      sources: ["chart-* token"],
    };
  }
  if (groupTitle.includes("Admin Sidebar")) {
    return {
      state: "active" as const,
      label: "AdminSidebar 组件专属 token 使用中",
      badges: ["AdminSidebar 使用中"],
      sources: ["admin-sidebar.tsx"],
    };
  }
  if (groupTitle.includes("Sidebar")) {
    return {
      state: "component" as const,
      label: "shadcn Sidebar 语义 token 使用中",
      badges: ["Sidebar 使用中"],
      sources: ["sidebar.tsx"],
    };
  }
  return {
    state: "component" as const,
    label: "组件使用中",
    badges: ["使用中"],
    sources: [],
  };
}

function ColorRamp({
  title,
  description,
  tokens,
}: ColorGroup) {
  return (
    <div className="space-y-3">
      <div>
        <BodyMedium>{title}</BodyMedium>
        <MetaText className="mt-1 block" tone="secondary">{description}</MetaText>
      </div>
      <div className="grid grid-cols-[repeat(auto-fill,minmax(184px,1fr))] gap-3">
        {tokens.map((token) => {
          const swatchColor = token.swatch ?? token.value;
          const dark = isDarkColor(swatchColor);
          const usageMeta = getColorTokenUsageMeta(token, title);
          return (
            <div key={`${title}-${token.name}`} className="overflow-hidden rounded-[4px] border border-[#DDE7F2] bg-white">
              <div className="relative h-[72px] border-b border-[#EAEEF4]" style={{ background: swatchColor }}>
                <div className="absolute right-2 top-2 flex flex-wrap justify-end gap-1">
                  {usageMeta.badges.map((badge) => (
                    <span
                      key={badge}
                      className="rounded-[2px] px-1.5 py-0.5 text-[10px] font-medium leading-none backdrop-blur-sm"
                      style={{ color: dark ? "#FFFFFF" : "#020617", backgroundColor: dark ? "rgba(2, 6, 23, 0.42)" : "rgba(255, 255, 255, 0.82)" }}
                    >
                      {badge}
                    </span>
                  ))}
                </div>
              </div>
              <div className="space-y-1 px-3 py-3">
                <BodyMedium className="block truncate text-[13px]">{token.name}</BodyMedium>
                <CodeText className="block truncate text-[11px]">{token.value}</CodeText>
                {token.cssVar ? <MetaText className="block truncate">{token.cssVar}</MetaText> : null}
                {token.className ? <MetaText className="block truncate">{token.className}</MetaText> : null}
                <MetaText className="block truncate" tone="secondary">用途：{token.usage}</MetaText>
                <div className={`mt-2 rounded-[4px] border px-2 py-1.5 ${colorUsageToneClasses[usageMeta.state]}`}>
                  <span className="block text-[11px] font-medium leading-[1.45]">使用情况：{usageMeta.label}</span>
                  {usageMeta.sources.length ? <span className="mt-0.5 block truncate text-[10px] leading-[1.4] opacity-75">来源：{usageMeta.sources.join(" / ")}</span> : null}
                </div>
                {dark ? <span className="sr-only">深色色卡</span> : null}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function getColorUsageSummary() {
  const metas = colorGroups.flatMap((group) => group.tokens.map((token) => getColorTokenUsageMeta(token, group.title)));
  return [
    { label: "直接使用", value: metas.filter((meta) => meta.state === "active").length, hint: "Typography / 语义 / 品牌 / Alert" },
    { label: "组件基础", value: metas.filter((meta) => meta.state === "component").length, hint: "Gray / Sidebar 等基础色" },
    { label: "替换候选", value: metas.filter((meta) => meta.state === "candidate").length, hint: "Slate 候选色阶" },
    { label: "兼容 / 预留", value: metas.filter((meta) => meta.state === "alias" || meta.state === "reserved").length, hint: "Legacy alias / Chart token" },
  ];
}

function ColorPreview() {
  const total = colorGroups.reduce((sum, group) => sum + group.tokens.length, 0);
  const usageSummary = getColorUsageSummary();

  return (
    <div className="space-y-4">
      <PreviewPanel title="全局颜色 token 色卡" layout="wide">
        <div className="space-y-7">
          <div className="rounded-[4px] border border-[#DDE7F2] bg-[#F8FAFF] px-4 py-3">
            <BodyMedium>当前展示 token 共 {total} 个</BodyMedium>
            <BodyText className="mt-1" tone="secondary">
              包含 index.css 中的全局颜色 token，并在每张色卡上标注“使用情况”和来源；Text 文字语义色已接入 Typography。
            </BodyText>
            <div className="mt-4 grid grid-cols-4 gap-2.5">
              {usageSummary.map((item) => (
                <div key={item.label} className="rounded-[4px] border border-[#DDE7F2] bg-white px-3 py-2">
                  <MetaText className="block">{item.label}</MetaText>
                  <div className="mt-1 flex items-end gap-1.5">
                    <InlineNumber className="text-lg font-semibold leading-none">{item.value}</InlineNumber>
                    <MetaText>个</MetaText>
                  </div>
                  <MetaText className="mt-1 block truncate" tone="secondary">{item.hint}</MetaText>
                </div>
              ))}
            </div>
          </div>
          {colorGroups.map((group) => (
            <ColorRamp key={group.title} {...group} />
          ))}
        </div>
      </PreviewPanel>
    </div>
  );
}

function TypographyPreview() {
  const rows = [
    ["TenantHeroTitle", <TenantHeroTitle key="hero">模型额度与用量总览</TenantHeroTitle>, "用户端 Hero 标题"],
    ["TenantPageTitle", <TenantPageTitle key="page">Agent 详情</TenantPageTitle>, "页面标题"],
    ["TenantDocTitle", <TenantDocTitle key="doc">帮助文档标题</TenantDocTitle>, "文章 / 文档标题"],
    ["SectionTitle", <SectionTitle key="section">组件分类展示</SectionTitle>, "大模块标题"],
    ["PanelTitle", <PanelTitle key="panel">全状态真实示例</PanelTitle>, "面板标题"],
    ["CardTitle", <CardTitle key="card">Alice 的技术助手</CardTitle>, "卡片标题"],
    ["BodyText", <BodyText key="body">这里展示组件使用说明和推荐参考方式。</BodyText>, "正文主内容"],
    ["BodyText secondary", <BodyText key="body-secondary" tone="secondary">用于描述行、补充说明等同字号浅色正文。</BodyText>, "描述性正文"],
    ["BodyMedium", <BodyMedium key="body-medium">按钮、Tab、Label 主文字</BodyMedium>, "中等强调正文"],
    ["CompactText", <CompactText key="compact">空间不足的紧凑描述文字。</CompactText>, "13px 紧凑文字"],
    ["MiniBodyText", <MiniBodyText key="mini-body">紧凑表格正文使用 12px 深色正文。</MiniBodyText>, "紧凑正文"],
    ["HelperText", <HelperText key="helper">仅支持英文字母、数字和下划线。</HelperText>, "输入说明 / 弱提示"],
    ["MetaText", <MetaText key="meta">更新于 2026-06-03 16:25</MetaText>, "辅助信息"],
    ["MetaMedium", <MetaMedium key="meta-medium">表头 / 次级强调</MetaMedium>, "辅助强调"],
    ["SmallBodyText", <SmallBodyText key="small-body">用户</SmallBodyText>, "小型标签文字"],
    ["TinyText", <TinyText key="tiny">Beta</TinyText>, "英文角标"],
    ["StatNumber", <StatNumber key="stat">128,000</StatNumber>, "统计数字"],
    ["InlineNumber", <InlineNumber key="inline-number">98.6%</InlineNumber>, "行内数字"],
    ["CodeText", <CodeText key="code">client/src/components/ui/button.tsx</CodeText>, "路径 / ID"],
    ["UrlText", <UrlText key="url">https://api.example.com/v1/chat/completions</UrlText>, "URL / 回调地址"],
    ["StepText", <StepText key="step">Step 1</StepText>, "步骤标识"],
  ] as const;
  const toneCards = [
    { token: "primary", name: "标题色", value: "--text-title / #0F172A", color: "var(--text-title)" },
    { token: "emphasis", name: "强调", value: "--text-emphasis / #020617", color: "var(--text-emphasis)" },
    { token: "body", name: "正文", value: "--text-body / #1E293B", color: "var(--text-body)" },
    { token: "secondary", name: "描述正文", value: "--text-secondary / #334155", color: "var(--text-secondary)" },
    { token: "muted", name: "辅助", value: "--text-muted / #64748B", color: "var(--text-muted)" },
    { token: "weak", name: "极弱", value: "--text-weak / #94A3B8", color: "var(--text-weak)" },
    { token: "brand", name: "活跃", value: "--text-brand / #1447E6", color: "var(--text-brand)" },
    { token: "danger", name: "危险", value: "--text-danger / #DC2626", color: "var(--text-danger)" },
  ] as const;

  return (
    <div className="space-y-4">
      <PreviewPanel title="Typography token 一览" layout="wide">
        <div className="divide-y divide-[#EAF1F8]">
          {rows.map(([name, example, usage]) => (
            <div key={name} className="grid grid-cols-[180px_minmax(0,1fr)_160px] items-center gap-4 py-3">
              <CodeText>{name}</CodeText>
              <div className="min-w-0">{example}</div>
              <MetaText>{usage}</MetaText>
            </div>
          ))}
        </div>
      </PreviewPanel>
      <PreviewPanel title="Tone 色阶示例" layout="wide">
        <div className="grid grid-cols-4 gap-2.5">
          {toneCards.map((tone) => (
            <div key={tone.token} className="overflow-hidden rounded-[4px] border border-[#EAF1F8] bg-white">
              <div className="h-11" style={{ backgroundColor: tone.color }} />
              <div className="px-2.5 py-2">
                <BodyMedium className="block truncate text-xs">{tone.name}</BodyMedium>
                <CodeText className="mt-0.5 block text-[11px]">{tone.value}</CodeText>
                <MetaText className="mt-1 block truncate">{tone.token}</MetaText>
              </div>
            </div>
          ))}
        </div>
      </PreviewPanel>
    </div>
  );
}

function SurfacePreview({ id }: { id: ComponentId }) {
  const panels = {
    "surface-card": <SurfaceCard hover className="rounded-[4px] p-5"><CardTitle>SurfaceCard</CardTitle><MetaText className="mt-2 block">页面主区块、列表卡、统计卡。开启 hover 可微抬。</MetaText></SurfaceCard>,
    "surface-inner": <SurfaceInner className="rounded-[4px] p-5"><CardTitle>SurfaceInner</CardTitle><MetaText className="mt-2 block">卡片内子面板或表格容器。</MetaText></SurfaceInner>,
    "surface-config": <SurfaceConfig className="rounded-[4px] p-5"><CardTitle>SurfaceConfig</CardTitle><MetaText className="mt-2 block">管理端高亮配置卡、引导卡。</MetaText></SurfaceConfig>,
    "surface-overlay": <SurfaceOverlay className="rounded-[4px] p-5"><CardTitle>SurfaceOverlay</CardTitle><MetaText className="mt-2 block">自定义浮层容器；Dialog / Popover 通常已内置浮层样式。</MetaText></SurfaceOverlay>,
    "tenant-card": (
      <TenantCard interactive className="w-[320px]">
        <div className="flex items-start justify-between gap-3">
          <CardTitle>Alice 的技术助手</CardTitle>
          <StatusTag mode="fill" variant="green">运行中</StatusTag>
        </div>
        <div className="space-y-1">
          <CodeText className="block">agent-prod-20260603-01</CodeText>
          <MetaText className="block">用户端业务卡片：12px 圆角 / 20px padding / 24px gap。</MetaText>
        </div>
        <div className="flex gap-2">
          <Button variant="tenant-outline-r20" size="claw-sm">设置</Button>
          <Button variant="tenant-primary" size="claw-sm">对话</Button>
        </div>
      </TenantCard>
    ),
  };
  return <PreviewPanel title="真实卡片层级示例">{panels[id as keyof typeof panels]}</PreviewPanel>;
}

function DarkVeilPreview() {
  return (
    <div className="space-y-4">
      <PreviewPanel title="hero 三层配方（基底 + DarkVeil + 收束叠层）" layout="wide">
        {/* 受控盒子：模拟 SurfaceCard(overflow-hidden) 内的 hero 区 */}
        <div className="relative h-[260px] w-full overflow-hidden rounded-[4px] border border-[#DDE7F2]">
          {/* 第 0 层：统一基底 */}
          <div className="pointer-events-none absolute inset-0 bg-[#E0EBFE]" />
          {/* 第 1 层：DarkVeil 动态背景 */}
          <DarkVeil
            speed={1.1}
            warpAmount={1.1}
            noiseIntensity={0.05}
            tintColor="#B2C3FF"
            className="pointer-events-none absolute inset-0 h-full w-full"
            style={{
              transform: "translateY(72px)",
              maskImage: "linear-gradient(to bottom, transparent 0%, #000 22%)",
              WebkitMaskImage: "linear-gradient(to bottom, transparent 0%, #000 22%)",
            }}
          />
          {/* 第 2 层：柔化收束叠层 */}
          <div className="pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-white/10 to-[#E0EBFE]" />
          {/* 内容层：永远 relative z-10 */}
          <div className="relative z-10 flex h-full flex-col justify-center px-10">
            <StatNumber>开通云开发能力</StatNumber>
            <BodyText className="mt-2 max-w-[420px]">
              示例 hero 区文案。DarkVeil 仅作装饰背景，文字 / 按钮始终落在 z-10 内容层之上。
            </BodyText>
          </div>
        </div>
      </PreviewPanel>
      <PreviewPanel title="使用说明" layout="wide">
        <div className="w-full space-y-1 rounded-[4px] border border-[#e5e5e5] bg-[#fafafa] p-3 text-xs text-[#64748B]">
          <p>• 纯背景组件：背景三层全部 pointer-events-none，不承载任何信息 / 交互 / 可点击元素。</p>
          <p>• 仅用于命中 Auto-Trigger 的管控端开通页 / 能力 hero，禁止扩散到列表 / 表单 / 整页背景。</p>
          <p>• 参数配方：speed 1.1 / warpAmount 1.1 / noiseIntensity 0.05 / tintColor #B2C3FF，顶部 mask 22% 淡出。</p>
          <p>• 唯一新依赖 ogl；无 ogl / WebGL 时按 L0 / L1 / L2 分档兜底，至少做到 L1 静态 CSS。</p>
        </div>
      </PreviewPanel>
    </div>
  );
}

function NumberCardPreview() {
  return (
    <div className="space-y-4">
      <PreviewPanel title="标准 KPI 概览（4 张内置渐变图标卡）" layout="wide">
        <div className="grid w-full grid-cols-4 gap-5">
          <NumberCard icon={<RequestsIcon />} label="总请求数" value="1,841" />
          <NumberCard icon={<InputTokensIcon />} label="输入 Tokens" value="533,112" />
          <NumberCard icon={<OutputTokensIcon />} label="输出 Tokens" value="419,040" />
          <NumberCard icon={<TotalTokensIcon />} label="总 Tokens" value="952,152" />
        </div>
      </PreviewPanel>
      <PreviewPanel title="extra 扩展：百分比 + 徽标 / 百分比 + 进度条">
        <div className="grid w-full grid-cols-2 gap-5">
          <NumberCard
            icon={<TotalTokensIcon />}
            label="今日全局配额消耗"
            value="0%"
            extra={
              <span className="text-xs font-semibold text-[#355EF1] bg-[#e0e9ff] px-2.5 py-1.5 rounded-[4px]">
                按组织
              </span>
            }
          />
          <NumberCard
            icon={<TotalTokensIcon />}
            label="本月全局配额消耗"
            value="68%"
            extra={
              <div className="flex w-full max-w-[200px] items-center">
                <div className="h-2 w-full overflow-hidden rounded-full bg-[#F1F5F9]">
                  <div className="h-full rounded-full bg-[#1447E6]" style={{ width: "68%" }} />
                </div>
              </div>
            }
          />
        </div>
      </PreviewPanel>
    </div>
  );
}

function ButtonPreview() {
  const variants = [
    ["claw-primary", <Button key="primary" variant="claw-primary" size="claw">创建 Agent</Button>],
    ["claw-outline", <Button key="outline" variant="claw-outline" size="claw">详细配置</Button>],
    ["dialog-confirm", <Button key="dialog" variant="dialog-confirm" size="claw-sm">确认</Button>],
    ["tenant-primary", <Button key="tenant-primary" variant="tenant-primary" size="claw">创建 Agent</Button>],
    ["tenant-outline", <Button key="tenant-outline" variant="tenant-outline" size="claw">详细配置</Button>],
    ["tenant-outline-r20", <Button key="tenant-outline-r20" variant="tenant-outline-r20" size="claw-sm">卡片操作</Button>],
    ["tenant-plain", <Button key="tenant-plain" variant="tenant-plain" size="sm" data-state="active">分类筛选</Button>],
    ["tenant-dialog-confirm", <Button key="tenant-dialog-confirm" variant="tenant-dialog-confirm" size="claw-sm">确认</Button>],
    ["destructive", <Button key="destructive" variant="destructive" size="claw">删除</Button>],
    ["tenant-destructive", <Button key="tenant-destructive" variant="tenant-destructive" size="claw">删除</Button>],
    ["ghost", <Button key="ghost" variant="ghost" size="claw">Ghost</Button>],
    ["tenant-ghost", <Button key="tenant-ghost" variant="tenant-ghost" size="claw">Ghost</Button>],
    ["plain", <Button key="plain" variant="plain" size="sm" data-state="active">筛选项</Button>],
    ["link", <Button key="link" variant="link" size="sm">查看文档</Button>],
    ["link-dark", <Button key="link-dark" variant="link-dark" size="sm">编辑</Button>],
  ] as const;

  return (
    <div className="space-y-4">
      <PreviewPanel title="Variant 状态" layout="wide">
        <div className="mx-auto grid max-w-[760px] grid-cols-4 gap-x-8 gap-y-7">
          {variants.map(([name, node]) => (
            <div key={name} className="flex min-h-[72px] flex-col items-center justify-end gap-2 text-center">
              <MetaText>{name}</MetaText>
              <div className="flex h-10 items-center justify-center">{node}</div>
            </div>
          ))}
        </div>
      </PreviewPanel>
      <PreviewPanel title="尺寸 / 图标 / Disabled / SmallIconStateButton" layout="wide">
        <div className="mx-auto flex max-w-[760px] flex-wrap items-center justify-center gap-3">
          <Button variant="claw-primary" size="claw-lg"><Plus />Large 40</Button>
          <Button variant="claw-primary" size="claw"><Plus />Default 36</Button>
          <Button variant="claw-primary" size="claw-sm"><Plus />Small 32</Button>
          <Button variant="claw-outline" size="claw-square" aria-label="设置"><Settings /></Button>
          <Button variant="claw-outline" size="claw" disabled>Disabled</Button>
          <SmallIconStateButton icon={Plus} label="添加" />
          <SmallIconStateButton icon={Settings} label="配置" />
          <SmallIconStateButton icon={Plus} label="添加" state="disabled" />
        </div>
      </PreviewPanel>
    </div>
  );
}

function FormPreview({ id }: { id: ComponentId }) {
  const [switchOn, setSwitchOn] = useState(true);
  const [checked, setChecked] = useState(true);
  const [date, setDate] = useState("2026-05-24");
  const [dateTime, setDateTime] = useState("2026-05-24 09:30");
  const [dateTimeSec, setDateTimeSec] = useState("2026-05-24 09:30:00");
  const [radioCardValue, setRadioCardValue] = useState("standard");
  const [chipValue, setChipValue] = useState("all");

  const map: Partial<Record<ComponentId, React.ReactNode>> = {
    input: (
      <PreviewPanel title="Input 状态">
        <div className="grid grid-cols-2 gap-4">
          <Input placeholder="请输入企业邮箱" />
          <Input defaultValue="miekoyychen@tencent.com" />
          <div className="space-y-1"><Input aria-invalid defaultValue="miekoyychen" /><MetaText tone="danger">请输入正确企业邮箱</MetaText></div>
          <Input defaultValue="addietang@tencent.com" disabled />
          <div className="relative col-span-2"><Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#A3A3A3]" /><Input className="pl-9" placeholder="搜索组件名、使用场景、源码路径" /></div>
        </div>
      </PreviewPanel>
    ),
    "input-group": (
      <PreviewPanel title="InputGroup 前后缀与内联操作" layout="wide">
        <div className="mx-auto max-w-[520px] space-y-3">
          <InputGroup>
            <InputGroupAddon><Search className="size-4" /></InputGroupAddon>
            <InputGroupInput placeholder="搜索 Agent、模型或组件" />
            <InputGroupAddon align="inline-end"><Kbd>⌘K</Kbd></InputGroupAddon>
          </InputGroup>
          <InputGroup>
            <InputGroupAddon><InputGroupText>https://</InputGroupText></InputGroupAddon>
            <InputGroupInput defaultValue="api.example.com/v1/chat/completions" />
            <InputGroupAddon align="inline-end"><InputGroupButton size="xs">复制</InputGroupButton></InputGroupAddon>
          </InputGroup>
        </div>
      </PreviewPanel>
    ),
    textarea: <PreviewPanel title="Textarea 状态"><Textarea placeholder="请输入页面效果校准说明" /><Textarea className="mt-3" defaultValue="已沉淀组件需优先复用真实组件样式。" /></PreviewPanel>,
    select: (
      <PreviewPanel title="Select 可交互示例">
        <Select defaultValue="admin">
          <SelectTrigger className="w-[320px]"><SelectValue placeholder="请选择范围" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="global">Global 全局</SelectItem>
            <SelectItem value="tenant">Tenant 用户端</SelectItem>
            <SelectItem value="admin">Admin 管控端</SelectItem>
          </SelectContent>
        </Select>
      </PreviewPanel>
    ),
    "date-picker": <PreviewPanel title="DatePicker 可交互示例"><DatePicker value={date} onChange={setDate} className="w-[320px]" /></PreviewPanel>,
    "date-time-picker": (
      <PreviewPanel title="DateTimePicker 可交互示例" layout="wide">
        <div className="mx-auto flex max-w-[640px] flex-col gap-6">
          <div className="space-y-2">
            <MetaText>日期 + 时分（默认）· 值 {dateTime}</MetaText>
            <DateTimePicker value={dateTime} onChange={setDateTime} className="w-[280px]" />
          </div>
          <div className="space-y-2">
            <MetaText>日期 + 时分秒（showSeconds）· 值 {dateTimeSec}</MetaText>
            <DateTimePicker showSeconds value={dateTimeSec} onChange={setDateTimeSec} className="w-[280px]" />
          </div>
          <div className="rounded-[4px] bg-[#fafafa] border border-[#e5e5e5] p-3 text-xs text-[#64748B] space-y-1">
            <p>• 右侧时 / 分 /（秒）列与日期同走品牌蓝 #1447E6 选中态</p>
            <p>• 值格式：默认 YYYY-MM-DD HH:mm；showSeconds → YYYY-MM-DD HH:mm:ss</p>
            <p>• 草稿态：选完点「确定」才提交 onChange</p>
          </div>
        </div>
      </PreviewPanel>
    ),
    checkbox: <PreviewPanel title="Checkbox 全状态"><div className="space-y-4"><div className="flex items-center gap-4"><div className="flex items-center gap-2"><Checkbox id="cb-default" /><Label htmlFor="cb-default">默认</Label></div><div className="flex items-center gap-2"><Checkbox id="cb-checked" checked={checked} onCheckedChange={(next) => setChecked(next === true)} /><Label htmlFor="cb-checked">选中</Label></div><div className="flex items-center gap-2"><Checkbox id="cb-indeterminate" checked="indeterminate" /><Label htmlFor="cb-indeterminate">半选</Label></div></div><div className="flex items-center gap-4"><div className="flex items-center gap-2"><Checkbox id="cb-disabled" disabled /><Label htmlFor="cb-disabled">禁用</Label></div><div className="flex items-center gap-2"><Checkbox id="cb-disabled-checked" disabled checked /><Label htmlFor="cb-disabled-checked">禁用+选中</Label></div></div><div className="rounded-[4px] bg-[#fafafa] border border-[#e5e5e5] p-3 text-xs text-[#64748B] space-y-1"><p>• 描边 --border-control (#C8CFDA)，不用 border-gray-200</p><p>• 选中色 #355EF1 · hover 边框变 #1447E6</p><p>• disabled 用 bg-[#f3f3f4]，不用 opacity-50</p></div></div></PreviewPanel>,
    field: (
      <PreviewPanel title="Field 字段结构" layout="wide">
        <FieldSet className="mx-auto max-w-[520px]">
          <FieldLegend>基础配置</FieldLegend>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="field-name">组件名称</FieldLabel>
              <Input id="field-name" defaultValue="InputGroup" />
              <FieldDescription>用于展示台搜索、过滤和输入组合场景。</FieldDescription>
            </Field>
            <Field orientation="horizontal" data-invalid="true">
              <Checkbox id="field-invalid" aria-invalid />
              <FieldContent>
                <FieldTitle>确认同步规范</FieldTitle>
                <FieldDescription>勾选后会在页面中优先使用全局组件。</FieldDescription>
                <FieldError>请确认后再提交。</FieldError>
              </FieldContent>
            </Field>
            <FieldSeparator>或</FieldSeparator>
            <Field orientation="horizontal">
              <Switch checked={switchOn} onCheckedChange={setSwitchOn} />
              <FieldContent>
                <FieldTitle>启用自动校准</FieldTitle>
                <FieldDescription>组件状态变化时同步更新预览。</FieldDescription>
              </FieldContent>
            </Field>
          </FieldGroup>
        </FieldSet>
      </PreviewPanel>
    ),
    "radio-group": (
      <PreviewPanel title="RadioGroup 可交互示例">
        <RadioGroup defaultValue="preview" className="flex gap-5">
          <div className="flex items-center gap-2"><RadioGroupItem value="preview" id="r-preview" /><Label htmlFor="r-preview">真实预览</Label></div>
          <div className="flex items-center gap-2"><RadioGroupItem value="guide" id="r-guide" /><Label htmlFor="r-guide">使用指引</Label></div>
          <div className="flex items-center gap-2"><RadioGroupItem value="migration" id="r-migration" /><Label htmlFor="r-migration">迁移建议</Label></div>
        </RadioGroup>
      </PreviewPanel>
    ),
    "radio-card": (
      <PreviewPanel title="RadioCard 卡片式单选" layout="wide">
        <RadioGroup value={radioCardValue} onValueChange={setRadioCardValue} className="mx-auto grid max-w-[720px] grid-cols-2 gap-3">
          <RadioCard id="radio-card-standard" value="standard" checked={radioCardValue === "standard"} title="标准配置" description="适合大多数 Agent 使用，推荐默认选择。" />
          <RadioCard id="radio-card-pro" value="pro" checked={radioCardValue === "pro"} title="高级配置" description="开放更多模型、通道和工具配置项。" />
        </RadioGroup>
      </PreviewPanel>
    ),
    switch: <PreviewPanel title="Switch 全状态"><div className="space-y-4"><div className="flex items-center gap-4"><div className="flex items-center gap-2"><Switch checked={switchOn} onCheckedChange={setSwitchOn} /><BodyMedium>{switchOn ? "开启" : "关闭"}</BodyMedium></div><div className="flex items-center gap-2"><Switch checked disabled /><MetaText>Disabled 开</MetaText></div><div className="flex items-center gap-2"><Switch checked={false} disabled /><MetaText>Disabled 关</MetaText></div></div><div className="rounded-[4px] bg-[#fafafa] border border-[#e5e5e5] p-3 text-xs text-[#64748B] space-y-1"><p>• 开启：轨道 #355EF1 · 关闭：轨道 #d3d6db</p><p>• 尺寸 h-5 w-9 (20×36) · 滑块白色圆形 4px 内缩</p></div></div></PreviewPanel>,
    "filter-chip": (
      <PreviewPanel title="FilterChip / FilterChipGroup">
        <div className="space-y-4">
          <FilterChipGroup
            value={chipValue}
            onChange={setChipValue}
            items={[{ id: "all", label: "全部" }, { id: "agent", label: "Agent" }, { id: "model", label: "模型" }, { id: "skill", label: "技能" }]}
          />
          <div className="flex gap-2"><FilterChip active>单个激活</FilterChip><FilterChip>普通标签</FilterChip></div>
        </div>
      </PreviewPanel>
    ),
  };

  return <>{map[id]}</>;
}

function AlertPreview() {
  const demoControls = <span className="text-xs tabular-nums text-[var(--text-secondary)]">4/5</span>;

  return (
    <PreviewPanel title="Alert 类型" layout="wide">
      <div className="space-y-8">
        <div className="space-y-4">
          <BodyText tone="secondary" className="text-[15px]">常规提示条</BodyText>
          <MetaText tone="secondary" className="!text-[12px] block">页面内嵌提示。按「信息层级」选择 variant：中性说明 → info / operation-info；非阻断风险 → warning；版本动态 → product-news。</MetaText>

          <div className="space-y-1.5">
            <div className="flex items-baseline gap-2 flex-wrap">
              <MetaMedium className="!text-[12px]">普通信息</MetaMedium>
              <CodeText className="!text-[11px]">variant=&quot;info&quot;</CodeText>
              <MetaText tone="secondary" className="!text-[11px]">页面常驻说明、功能告知。中性蓝底，描述类信息首选。</MetaText>
            </div>
            <Alert variant="info"><AlertInfoIcon /><AlertDescription>普通信息提示，适合页面常驻说明和功能告知。</AlertDescription></Alert>
          </div>

          <div className="space-y-1.5">
            <div className="flex items-baseline gap-2 flex-wrap">
              <MetaMedium className="!text-[12px]">操作说明</MetaMedium>
              <CodeText className="!text-[11px]">variant=&quot;operation-info&quot;</CodeText>
              <MetaText tone="secondary" className="!text-[11px]">表单 / 批量操作上下文的辅助说明，白底灰边更克制，强调步骤而非风险。</MetaText>
            </div>
            <Alert variant="operation-info"><AlertOperationInfoIcon /><AlertTitle>操作说明</AlertTitle><AlertDescription>用于批量操作前后的辅助说明。</AlertDescription></Alert>
          </div>

          <div className="space-y-1.5">
            <div className="flex items-baseline gap-2 flex-wrap">
              <MetaMedium className="!text-[12px]">注意事项</MetaMedium>
              <CodeText className="!text-[11px]">variant=&quot;warning&quot;</CodeText>
              <MetaText tone="secondary" className="!text-[11px]">配置缺失、配额不足等「非阻断」风险提醒；图标必须传 CircleAlert，禁止当 error 用。</MetaText>
            </div>
            <Alert variant="warning"><CircleAlert /><AlertTitle>注意事项</AlertTitle><AlertDescription>用于配置缺失、配额不足等非阻断提醒。</AlertDescription></Alert>
          </div>

          <div className="space-y-1.5">
            <div className="flex items-baseline gap-2 flex-wrap">
              <MetaMedium className="!text-[12px]">产品动态</MetaMedium>
              <CodeText className="!text-[11px]">variant=&quot;product-news&quot;</CodeText>
              <MetaText tone="secondary" className="!text-[11px]">版本发布、新功能上线、活动通告；图标固定 sparkle，文案前缀建议带【产品动态】。</MetaText>
            </div>
            <Alert variant="product-news"><AlertProductNewsIcon /><AlertDescription>【产品动态】组件展示台已接入新的全状态示例。</AlertDescription></Alert>
          </div>

          <div className="space-y-1.5">
            <div className="flex items-baseline gap-2 flex-wrap">
              <MetaMedium className="!text-[12px]">成功提示</MetaMedium>
              <CodeText className="!text-[11px]">variant=&quot;success&quot;</CodeText>
              <MetaText tone="secondary" className="!text-[11px]">操作成功、状态正常等正向反馈；淡翠绿底，克制不抢眼。</MetaText>
            </div>
            <Alert variant="success"><AlertSuccessIcon /><AlertDescription>配置保存成功，已生效。</AlertDescription></Alert>
          </div>

          <div className="space-y-1.5">
            <div className="flex items-baseline gap-2 flex-wrap">
              <MetaMedium className="!text-[12px]">错误提示</MetaMedium>
              <CodeText className="!text-[11px]">variant=&quot;error&quot;</CodeText>
              <MetaText tone="secondary" className="!text-[11px]">操作失败、请求异常等错误反馈；淡红底，比 warning 更强调问题已发生。</MetaText>
            </div>
            <Alert variant="error"><AlertErrorIcon /><AlertTitle>保存失败</AlertTitle><AlertDescription>网络异常，请稍后重试。</AlertDescription></Alert>
          </div>
        </div>

        <div className="space-y-4">
          <BodyText tone="secondary" className="text-[15px]">管理端彩色背景公告条</BodyText>
          <MetaText tone="secondary" className="!text-[12px] block">仅用于管控端顶部常驻位。带左侧彩签 + 右侧翻页 / 关闭控件；同时只展示 1 条，多条排队轮播。</MetaText>

          <div className="rounded-[4px] bg-[linear-gradient(180deg,#F7FAFF_0%,#EEF4FB_100%)] px-5 py-4 space-y-4">
            <div className="space-y-1.5">
              <div className="flex items-baseline gap-2 flex-wrap">
                <MetaMedium className="!text-[12px]">产品动态</MetaMedium>
                <CodeText className="!text-[11px]">type=&quot;product-news&quot;</CodeText>
                <MetaText tone="secondary" className="!text-[11px]">版本 / 新功能 / 活动公告；蓝色彩签 + sparkle 图标，可不带跳转链接。</MetaText>
              </div>
              <AdminNoticeAlert type="product-news" controls={demoControls}>
                <span>OpenClaw v2.4.0 已发布：记忆管理功能上线。</span>
              </AdminNoticeAlert>
            </div>

            <div className="space-y-1.5">
              <div className="flex items-baseline gap-2 flex-wrap">
                <MetaMedium className="!text-[12px]">待配置</MetaMedium>
                <CodeText className="!text-[11px]">type=&quot;pending-config&quot;</CodeText>
                <MetaText tone="secondary" className="!text-[11px]">基础信息 / 模型 / 安全组等首日未完成项的引导；橙色彩签，必须带跳转链接，处理完应自动消失。</MetaText>
              </div>
              <AdminNoticeAlert type="pending-config" controls={demoControls}>
                <span>有 3 项基础配置未完成（导入企业用户、配置至少一个通道、配置安全组），未完成配置将影响用户端的正常使用，</span>
                <span className="font-medium text-[var(--text-emphasis)] underline underline-offset-2">前往基础信息配置处理</span>
              </AdminNoticeAlert>
            </div>

            <div className="space-y-1.5">
              <div className="flex items-baseline gap-2 flex-wrap">
                <MetaMedium className="!text-[12px]">资源告警</MetaMedium>
                <CodeText className="!text-[11px]">type=&quot;resource-alert&quot;</CodeText>
                <MetaText tone="secondary" className="!text-[11px]">配额耗尽、VPC 故障等需「立刻处理」的运行态告警；橙色彩签，必须带工单 / 处理入口跳转。</MetaText>
              </div>
              <AdminNoticeAlert type="resource-alert" controls={demoControls}>
                <span>私有网络（VPC）配额已耗尽，将影响用户端云设备的正常创建与使用，</span>
                <span className="text-[var(--text-emphasis)] underline underline-offset-2">前往腾讯云控制台提交工单</span>
              </AdminNoticeAlert>
            </div>
          </div>
        </div>
      </div>
    </PreviewPanel>
  );
}

function DialogPreview() {
  return (
    <PreviewPanel title="Dialog 可交互示例">
      <Dialog>
        <DialogTrigger asChild><Button variant="claw-outline" size="claw-sm">打开表单弹窗</Button></DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新增组件说明</DialogTitle>
            <DialogDescription>弹窗内表单控件继续复用全局 Input 与 Select。</DialogDescription>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <Input placeholder="组件名称" />
            <Select defaultValue="feedback">
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="feedback">反馈组件</SelectItem><SelectItem value="data">数据展示</SelectItem></SelectContent>
            </Select>
          </div>
          <DialogFooter><Button variant="claw-outline" size="claw-sm">取消</Button><Button variant="dialog-confirm" size="claw-sm">确认</Button></DialogFooter>
        </DialogContent>
      </Dialog>
    </PreviewPanel>
  );
}

function AlertDialogPreview() {
  return (
    <PreviewPanel title="AlertDialog 危险确认">
      <AlertDialog>
        <AlertDialogTrigger asChild><Button variant="destructive" size="claw-sm">危险确认</Button></AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除该示例？</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription>危险操作请使用 AlertDialog 承载。</AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction>确认删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </PreviewPanel>
  );
}

function DrawerPreview() {
  return (
    <PreviewPanel title="Drawer 右侧详情抽屉" layout="wide">
      <div className="flex flex-col items-center gap-3">
        <BodyText tone="secondary">点击按钮打开管控端右侧详情抽屉，查看 Header、Body、Footer 与紧凑信息组织。</BodyText>
        <Drawer direction="right">
          <DrawerTrigger asChild>
            <Button variant="claw-outline" size="claw-sm">打开 Agent 详情抽屉</Button>
          </DrawerTrigger>
          <DrawerContent className="data-[vaul-drawer-direction=right]:w-[480px] data-[vaul-drawer-direction=right]:sm:max-w-none max-w-[calc(100vw-24px)] h-full rounded-none bg-background p-0">
            <DrawerHeader className="relative bg-background p-4 pr-20 text-left">
              <div className="min-w-0 space-y-1">
                <DrawerTitle asChild>
                  <PanelTitle as="h2">Agent 详情</PanelTitle>
                </DrawerTitle>
                <DrawerDescription asChild>
                  <MetaText as="p">用于管控端对象详情查看与局部配置编辑。</MetaText>
                </DrawerDescription>
              </div>
              <div className="absolute right-4 top-4 flex items-center gap-1">
                <Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-[var(--text-title)] hover:text-[var(--text-emphasis)]" aria-label="刷新">
                  <RefreshCw className="w-4 h-4" />
                </Button>
                <DrawerClose asChild>
                  <Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-[var(--text-title)] hover:text-[var(--text-emphasis)]" aria-label="关闭">
                    <X className="w-4 h-4" />
                  </Button>
                </DrawerClose>
              </div>
            </DrawerHeader>
            <DrawerBody>
              <div className="space-y-6 p-4">
                <section className="min-w-0 space-y-1.5">
                  <PanelTitle as="div" className="truncate leading-tight">Alice 的技术助手</PanelTitle>
                  <CodeText className="block">agent-prod-20260603-01</CodeText>
                  <div className="flex flex-wrap items-center gap-2 pt-1">
                    <StatusTag mode="dot" variant="green">运行中</StatusTag>
                    <StatusTag mode="fill" variant="blue">全部用户</StatusTag>
                    <StatusTag mode="fill" variant="gray">v2.4.0</StatusTag>
                  </div>
                </section>
                <section className="space-y-2">
                  <div className="flex items-center justify-between">
                    <MetaMedium>已应用模型（2）</MetaMedium>
                    <MetaText as="button" tone="brand" className="inline-flex items-center gap-1">
                      <Plus className="size-3" /> 添加模型
                    </MetaText>
                  </div>
                  <SurfaceInner className="rounded-[4px] p-3">
                    <div className="grid grid-cols-[96px_minmax(0,1fr)] gap-x-3 gap-y-2">
                      <MetaText>主模型</MetaText><BodyMedium className="truncate">hunyuan-turbo-latest</BodyMedium>
                      <MetaText>备用模型</MetaText><BodyMedium className="truncate">deepseek-v3</BodyMedium>
                      <MetaText>更新时间</MetaText><InlineNumber>2026-06-03 10:58</InlineNumber>
                    </div>
                  </SurfaceInner>
                </section>
                <section className="space-y-2">
                  <MetaMedium>最近任务</MetaMedium>
                  <div className="overflow-hidden rounded-[4px] border border-[#EAF1F8]">
                    <Table density="compact">
                      <TableHeader><TableRow><TableHead>任务</TableHead><TableHead>状态</TableHead><TableHead className="text-right">耗时</TableHead></TableRow></TableHeader>
                      <TableBody>
                        <TableRow><TableCell>安全检查</TableCell><TableCell><StatusTag mode="dot" variant="green">完成</StatusTag></TableCell><TableCell className="text-right tabular-nums">12s</TableCell></TableRow>
                        <TableRow><TableCell>插件同步</TableCell><TableCell><StatusTag mode="dot" variant="blue">进行中</StatusTag></TableCell><TableCell className="text-right tabular-nums">36s</TableCell></TableRow>
                      </TableBody>
                    </Table>
                  </div>
                </section>
              </div>
            </DrawerBody>
            <DrawerFooter className="flex-row justify-end border-t border-[#E5E5E5] bg-background">
              <DrawerClose asChild><Button variant="claw-outline" size="claw-sm">取消</Button></DrawerClose>
              <Button variant="dialog-confirm" size="claw-sm">保存配置</Button>
            </DrawerFooter>
          </DrawerContent>
        </Drawer>
      </div>
    </PreviewPanel>
  );
}

function FloatingPreview({ id }: { id: ComponentId }) {
  if (id === "tooltip") {
    return <PreviewPanel title="Tooltip 可交互示例"><TooltipProvider><Tooltip><TooltipTrigger asChild><Button variant="claw-outline" size="claw-sm">悬停查看说明</Button></TooltipTrigger><TooltipContent>Tooltip 用于短说明，不承载复杂内容。</TooltipContent></Tooltip></TooltipProvider></PreviewPanel>;
  }
  if (id === "popover") {
    return <PreviewPanel title="Popover 可交互示例"><Popover><PopoverTrigger asChild><Button variant="claw-outline" size="claw-sm">打开浮层</Button></PopoverTrigger><PopoverContent className="w-72"><CardTitle>Popover 浮层</CardTitle><MetaText className="mt-2 block">适合临时筛选、简短说明或轻量操作。</MetaText></PopoverContent></Popover></PreviewPanel>;
  }
  if (id === "info-popover") {
    return (
      <PreviewPanel title="InfoPopover hover 说明">
        <div className="flex items-center gap-2">
          <BodyMedium>全局每日 Token 配额</BodyMedium>
          <InfoPopover title="字段说明" content="-1 表示不限额；配置为正整数时，将按自然日统计全部用户的 Token 用量。" />
        </div>
      </PreviewPanel>
    );
  }
  if (id === "spinner") {
    return (
      <PreviewPanel title="Spinner 加载状态">
        <div className="flex items-center gap-4">
          <Spinner className="text-[#1447E6]" />
          <Button variant="claw-primary" size="claw-sm" disabled><Spinner />同步中</Button>
          <MetaText>用于按钮加载和局部异步状态。</MetaText>
        </div>
      </PreviewPanel>
    );
  }
  return <PreviewPanel title="Progress 状态"><div className="max-w-sm space-y-2"><div className="flex justify-between"><MetaText>示例覆盖度</MetaText><InlineNumber>72%</InlineNumber></div><Progress value={72} /></div></PreviewPanel>;
}

function TablePreview() {
  const rows = [["Button", "操作组件", "运行中", 42], ["Input", "表单组件", "已停止", 39], ["Table", "数据展示", "运行中", 26]] as const;
  const renderTable = (density: "default" | "compact") => (
    <div className="overflow-hidden rounded-[4px] border border-[var(--cp-border)] bg-white">
      <Table density={density}>
        <TableHeader><TableRow><TableHead>组件</TableHead><TableHead>分类</TableHead><TableHead>状态</TableHead><TableHead className="text-right">应用范围</TableHead><TableHead className="w-[160px]">操作</TableHead></TableRow></TableHeader>
        <TableBody>
          {rows.map(([name, group, status, modules]) => (
            <TableRow key={name}>
              <TableCell className="font-medium">{name}</TableCell>
              <TableCell>{group}</TableCell>
              <TableCell><StatusTag mode="text" variant={status === "运行中" ? "green" : "red"}>{status}</StatusTag></TableCell>
              <TableCell className="text-right tabular-nums">约 {modules} 个页面/模块</TableCell>
              <TableActionCell><Button variant="link">查看</Button><Button variant="link">编辑</Button><Button variant="link" disabled>删除</Button></TableActionCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[#f0f0f0]">
        <MetaText className="justify-self-start">共 3 条</MetaText>
        <Pagination total={3} current={1} pageSize={10} showSizeChanger={false} className="justify-self-end" />
      </div>
    </div>
  );

  return (
    <PreviewPanel title="Table 全场景展示" layout="wide">
      <div className="space-y-6">
        <div>
          <MetaMedium className="mb-2 block">标准版（默认）· gray-header · 操作列 variant="link" · 含 disabled 态</MetaMedium>
          {renderTable("default")}
        </div>
        <div>
          <MetaMedium className="mb-2 block">紧凑版 density="compact" · 行高 40px · 12px 字号</MetaMedium>
          {renderTable("compact")}
        </div>
        <div>
          <MetaMedium className="mb-2 block">表格空态（纯文字双行 · 不用插画）</MetaMedium>
          <div className="overflow-hidden rounded-[4px] border border-[var(--cp-border)] bg-white">
            <Table>
              <TableHeader><TableRow><TableHead>名称</TableHead><TableHead>状态</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
              <TableBody><TableRow className="hover:!bg-transparent"><TableCell colSpan={3} className="!h-auto !p-0 hover:!bg-transparent"><div className="text-center py-12 space-y-1"><HelperText>暂无记录</HelperText><HelperText>尝试调整筛选条件，或新建一条记录</HelperText></div></TableCell></TableRow></TableBody>
            </Table>
          </div>
        </div>
        <div>
          <MetaMedium className="mb-2 block">表格内状态标签规则</MetaMedium>
          <div className="overflow-hidden rounded-[4px] border border-[#DDE7F2] bg-white">
            <Table density="compact">
              <TableHeader><TableRow><TableHead>名称</TableHead><TableHead>运行状态</TableHead><TableHead>版本</TableHead><TableHead>类型</TableHead></TableRow></TableHeader>
              <TableBody>
                <TableRow><TableCell>Agent-001</TableCell><TableCell><StatusTag mode="text" variant="green">正常</StatusTag></TableCell><TableCell>v2.1.0</TableCell><TableCell>公共</TableCell></TableRow>
                <TableRow><TableCell>Agent-002</TableCell><TableCell><StatusTag mode="text" variant="red">异常</StatusTag></TableCell><TableCell>v1.8.3</TableCell><TableCell>自定义</TableCell></TableRow>
              </TableBody>
            </Table>
          </div>
          <div className="mt-2 rounded-[4px] bg-[#fafafa] border border-[#e5e5e5] p-3 text-xs text-[#64748B] space-y-1">
            <p>• 状态列：mode="text" 纯文字变色（禁止 fill 胶囊）</p>
            <p>• 版本号：纯文字（禁止 StatusTag 包裹）</p>
            <p>• 类型/来源：纯文字独立列（禁止彩色标签）</p>
          </div>
        </div>
      </div>
    </PreviewPanel>
  );
}

function PaginationPreview() {
  return (
    <div className="space-y-4">
      <PreviewPanel title="完整功能（默认尺寸 32px）" layout="wide"><Pagination total={245} current={1} pageSize={10} showTotal={(total) => `共 ${total} 条`} showSizeChanger /></PreviewPanel>
      <PreviewPanel title="小尺寸 24px（弹窗/抽屉内）" layout="wide"><Pagination total={120} current={1} pageSize={10} size="small" showTotal={(total) => `共 ${total} 条`} /></PreviewPanel>
      <PreviewPanel title="简洁模式（可跳页）"><Pagination total={45} current={1} pageSize={10} mode="simple" /></PreviewPanel>
      <PreviewPanel title="极简只读（卡片标题右侧等狭窄场景）"><Pagination total={50} current={1} pageSize={10} mode="simple" size="small" /></PreviewPanel>
      <PreviewPanel title="仅单页隐藏"><Pagination total={120} current={1} pageSize={10} hideOnSinglePage /></PreviewPanel>
      <PreviewPanel title="表格底部标准布局（总数左 + 分页右）" layout="wide">
        <div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border border-[#f0f0f0] rounded-[4px]">
          <MetaText className="justify-self-start">共 245 条</MetaText>
          <Pagination total={245} current={1} pageSize={10} showSizeChanger className="justify-self-end justify-end flex-nowrap" />
        </div>
      </PreviewPanel>
    </div>
  );
}

function SegmentTabsPreview({ id }: { id: ComponentId }) {
  const [activeSegmentedTab, setActiveSegmentedTab] = useState("overview");
  if (id === "segment") {
    return (
      <PreviewPanel title="Segment 可交互切换">
        <Segment defaultValue="style">
          <SegmentList><SegmentItem value="style">样式</SegmentItem><SegmentItem value="usage">使用指引</SegmentItem><SegmentItem value="migration">迁移建议</SegmentItem><SegmentItem value="disabled" disabled>禁用项</SegmentItem></SegmentList>
          <SegmentContent value="style"><BodyText>用于详情页子内容切换，选中态为白底深色文字。</BodyText></SegmentContent>
          <SegmentContent value="usage"><BodyText>内容区分类切换优先使用 Segment。</BodyText></SegmentContent>
          <SegmentContent value="migration"><BodyText>手写按钮组可以逐步迁移为 Segment。</BodyText></SegmentContent>
        </Segment>
      </PreviewPanel>
    );
  }
  if (id === "segmented-tabs") {
    return (
      <PreviewPanel title="SegmentedTabs 受控切换">
        <div className="space-y-4">
          <SegmentedTabs
            active={activeSegmentedTab}
            onChange={setActiveSegmentedTab}
            ariaLabel="组件展示台分段切换"
            tabs={[{ id: "overview", label: "概览" }, { id: "files", label: "文件列表" }, { id: "settings", label: "设置" }]}
          />
          <BodyText tone="secondary">当前选中：{activeSegmentedTab}</BodyText>
        </div>
      </PreviewPanel>
    );
  }
  return (
    <PreviewPanel title="Tabs 全变体展示" layout="wide">
      <div className="space-y-8">
        <div>
          <MetaMedium className="mb-3 block">LineTabs（下划线式 · 页面标题下方一级导航）</MetaMedium>
          <div className="flex items-center gap-1 border-b border-[#dbe6ff]">
            {["概览", "Agent 列表", "技能配置", "模型管理"].map((tab, i) => (
              <button key={tab} className={`relative px-4 py-3 text-[14px] font-medium transition-colors whitespace-nowrap ${i === 0 ? "text-[#0F172A] border-b-2 border-[#0A0A0A] -mb-px" : "text-[#737373] hover:text-[#0F172A]"}`}>
                {tab}
              </button>
            ))}
          </div>
          <MetaText className="mt-2 block">选中态：text-title + border-b-2 黑色 · 默认态：text-muted · 仅用于页面标题下方</MetaText>
        </div>

        <div>
          <MetaMedium className="mb-3 block">Tab 切换卡（筛选标签按钮 · 黑底白字 active）</MetaMedium>
          <div className="flex items-center gap-2 flex-wrap">
            {["全部", "公共技能", "企业技能", "自定义"].map((cat, i) => (
              <button key={cat} className={`h-8 px-4 rounded-[4px] text-sm leading-[22px] tracking-[0.07px] border transition-colors ${i === 0 ? "bg-[#020617] border-[#020617] text-white" : "bg-white border-[#EAEEF4] text-[#020617] hover:border-[#020617]"}`}>
                {cat}
              </button>
            ))}
            <button className="h-8 px-4 rounded-[4px] text-sm border bg-white border-[#EAEEF4] text-[rgba(0,0,0,0.3)] cursor-not-allowed">禁用</button>
          </div>
          <MetaText className="mt-2 block">Active：黑底白字 · Hover：白底黑边 · Normal：白底灰边 · Disabled：淡字</MetaText>
        </div>

        <div>
          <MetaMedium className="mb-3 block">Segment / Tabs（内容面板切换 · 深黑选中态）</MetaMedium>
          <Tabs defaultValue="preview">
            <TabsList><TabsTrigger value="preview">真实预览</TabsTrigger><TabsTrigger value="guide">使用指引</TabsTrigger><TabsTrigger value="notes">注意事项</TabsTrigger></TabsList>
            <TabsContent value="preview"><BodyText className="mt-3">用于详情页/配置页子内容切换，选中态为深黑文字 + 白底浮起。</BodyText></TabsContent>
            <TabsContent value="guide"><BodyText className="mt-3">保持 active、hover 和 disabled 状态一致。</BodyText></TabsContent>
            <TabsContent value="notes"><BodyText className="mt-3">不建议用散落的 button 组合替代。</BodyText></TabsContent>
          </Tabs>
        </div>

        <div className="rounded-[4px] border border-[#e5e5e5] bg-[#fafafa] p-4 text-xs text-[#64748B] space-y-1">
          <p><strong>使用场景约束</strong></p>
          <p>• LineTabs：仅限页面标题下方的一级导航</p>
          <p>• Tab 切换卡：弹窗/卡片内分类筛选 + 表格工具栏筛选（黑底白字 active）</p>
          <p>• Segment / Tabs：详情页/配置页子内容切换（深黑选中 + 白底浮起）</p>
        </div>
      </div>
    </PreviewPanel>
  );
}

function KbdPreview() {
  return (
    <PreviewPanel title="Kbd / KbdGroup 快捷键标签">
      <div className="flex flex-col items-center gap-4">
        <div className="flex items-center gap-2"><Keyboard className="size-4 text-[#64748B]" /><BodyMedium>打开命令面板</BodyMedium><KbdGroup><Kbd>⌘</Kbd><Kbd>K</Kbd></KbdGroup></div>
        <div className="flex items-center gap-2"><MetaText>快速搜索</MetaText><Kbd>/</Kbd><MetaText>退出</MetaText><Kbd>Esc</Kbd></div>
      </div>
    </PreviewPanel>
  );
}

function StepperPreview() {
  return (
    <PreviewPanel title="Stepper 流程步骤" layout="wide">
      <Stepper
        current={2}
        steps={[{ label: "选择数据源方式" }, { label: "添加应用凭证" }, { label: "设置字段映射" }, { label: "完成" }]}
      />
    </PreviewPanel>
  );
}

function StatusPreview({ id }: { id: ComponentId }) {
  if (id === "badge") {
    return (
      <PreviewPanel title="Badge variants" layout="wide">
        <div className="space-y-5">
          <div>
            <MetaMedium className="mb-2 block">标准变体（4 种）</MetaMedium>
            <div className="flex flex-wrap items-center gap-3"><Badge>Default</Badge><Badge variant="secondary">Secondary</Badge><Badge variant="outline">Outline</Badge><Badge variant="destructive">Destructive</Badge></div>
          </div>
          <div>
            <MetaMedium className="mb-2 block">Custom Colors（4 种，通过 color prop）</MetaMedium>
            <div className="flex flex-wrap items-center gap-3"><Badge color="blue">Blue</Badge><Badge color="green">Green</Badge><Badge color="purple">Purple</Badge><Badge color="red">Red</Badge></div>
          </div>
          <div className="rounded-[4px] border border-[#e5e5e5] bg-[#fafafa] p-3 text-xs text-[#64748B] space-y-1">
            <p>• Badge 用于 New / Beta / 信息标签，不用于运行状态（状态用 StatusTag）</p>
            <p>• Custom Colors 仅 blue/green/purple/red 四种，新增需先补 token</p>
          </div>
        </div>
      </PreviewPanel>
    );
  }
  if (id === "status-tag") {
    return (
      <PreviewPanel title="StatusTag 全模式展示" layout="wide">
        <div className="space-y-5">
          <div>
            <MetaMedium className="mb-2 block">mode="text"（表格状态列默认 · 纯文字变色 · 无底色无圆点）</MetaMedium>
            <div className="flex flex-wrap items-center gap-4">
              <StatusTag mode="text" variant="green">正常</StatusTag>
              <StatusTag mode="text" variant="blue">进行中</StatusTag>
              <StatusTag mode="text" variant="red">异常</StatusTag>
              <StatusTag mode="text" variant="orange">待处理</StatusTag>
              <StatusTag mode="text" variant="gray">已关闭</StatusTag>
            </div>
          </div>
          <div>
            <MetaMedium className="mb-2 block">mode="fill"（信息/分类/版本标签 · 带浅色底）</MetaMedium>
            <div className="flex flex-wrap items-center gap-3">
              <StatusTag mode="fill" variant="blue">全部用户</StatusTag>
              <StatusTag mode="fill" variant="gray">v1.2.0</StatusTag>
              <StatusTag mode="fill" variant="green">已接入</StatusTag>
              <StatusTag mode="fill" variant="red">高风险</StatusTag>
              <StatusTag mode="fill" variant="orange">待确认</StatusTag>
            </div>
          </div>
          <div>
            <MetaMedium className="mb-2 block">mode="soft"（轻量彩色标签 · 带浅底+浅边框 · 4px 圆角）</MetaMedium>
            <div className="flex flex-wrap items-center gap-3">
              <StatusTag mode="soft" variant="blue">OpenClaw</StatusTag>
              <StatusTag mode="soft" variant="green">已验证</StatusTag>
              <StatusTag mode="soft" variant="orange">实验中</StatusTag>
              <StatusTag mode="soft" variant="gray">最新版本</StatusTag>
            </div>
          </div>
          <div>
            <MetaMedium className="mb-2 block">preset 角色标签</MetaMedium>
            <div className="flex flex-wrap items-center gap-3">
              <StatusTag preset="role-admin" />
              <StatusTag preset="role-user" />
            </div>
          </div>
          <div className="rounded-[4px] border border-[#e5e5e5] bg-[#fafafa] p-3 text-xs text-[#64748B] space-y-1">
            <p>• 表格状态列必须用 mode="text"（禁止在表格内用 fill）</p>
            <p>• mode="dot" 已全局废弃（组件 fallback 到 text）</p>
            <p>• 版本号纯文字，不用 StatusTag 包裹</p>
          </div>
        </div>
      </PreviewPanel>
    );
  }
  return (
    <PreviewPanel title="Empty 空状态分层" layout="wide">
      <div className="space-y-5">
        <div>
          <MetaMedium className="mb-2 block">页面级（双行 + 插画）</MetaMedium>
          <Empty className="border-0"><EmptyHeader><EmptyMedia /><EmptyTitle>还没有创建任何 Agent</EmptyTitle><EmptyDescription>创建你的第一个 Agent，开始自动化工作流</EmptyDescription></EmptyHeader></Empty>
        </div>
        <div>
          <MetaMedium className="mb-2 block">页面级（单行 · 禁用粗黑标题）</MetaMedium>
          <Empty className="border-0"><EmptyHeader><EmptyMedia /><EmptyDescription>暂无记录</EmptyDescription></EmptyHeader></Empty>
        </div>
        <div>
          <MetaMedium className="mb-2 block">表格空态（纯文字双行 · 不用插画）</MetaMedium>
          <div className="overflow-hidden rounded-[4px] border border-[var(--cp-border)]">
            <Table><TableHeader><TableRow><TableHead>名称</TableHead><TableHead>状态</TableHead><TableHead>操作</TableHead></TableRow></TableHeader><TableBody><TableRow className="hover:!bg-transparent"><TableCell colSpan={3} className="!h-auto !p-0 hover:!bg-transparent"><div className="text-center py-12 space-y-1"><HelperText>暂无记录</HelperText><HelperText>尝试调整筛选条件，或新建一条记录</HelperText></div></TableCell></TableRow></TableBody></Table>
          </div>
        </div>
        <div>
          <MetaMedium className="mb-2 block">浮层/弹窗空态（纯文字 · 禁用插画）</MetaMedium>
          <div className="max-w-[280px] rounded-[8px] border border-[#e5e5e5] bg-white shadow-lg p-0">
            <div className="px-4 py-3 border-b border-[#f0f0f0]"><BodyMedium>通知</BodyMedium></div>
            <div className="text-center py-6"><HelperText>暂无未读通知</HelperText></div>
          </div>
        </div>
      </div>
    </PreviewPanel>
  );
}

function TenantSectionPreview() {
  return (
    <PreviewPanel title="TenantSection 标题进卡内" layout="wide">
      <TenantSection
        title="MCP 配置"
        headingLevel="panel"
        actions={<Button variant="tenant-primary" size="claw-sm"><Plus />添加 MCP</Button>}
        className="mx-auto max-w-[720px]"
      >
        <Alert variant="info"><AlertInfoIcon /><AlertDescription>标题、操作按钮和内容统一进入 TenantCard，避免用户端段落结构不一致。</AlertDescription></Alert>
        <div className="grid grid-cols-2 gap-3">
          <SurfaceInner className="rounded-[4px] p-3"><CardTitle>stdio-server</CardTitle><MetaText className="mt-1 block">已启用 · 本地工具</MetaText></SurfaceInner>
          <SurfaceInner className="rounded-[4px] p-3"><CardTitle>http-server</CardTitle><MetaText className="mt-1 block">待配置 · 远程服务</MetaText></SurfaceInner>
        </div>
      </TenantSection>
    </PreviewPanel>
  );
}

function AdminPageHeaderPreview() {
  return (
    <PreviewPanel title="AdminPageHeader 管控页头" layout="wide">
      <div className="rounded-[4px] border border-[#DDE7F2] bg-white p-6">
        <AdminPageHeader
          title="模型配置"
          description="统一管控端页面标题、描述、标题附件和右侧操作区域。"
          titleAccessory={<StatusTag mode="fill" variant="blue">已更新</StatusTag>}
          actions={<><Button variant="claw-outline" size="claw-sm">刷新</Button><Button variant="claw-primary" size="claw-sm"><Plus />新增模型</Button></>}
        />
        <SurfaceInner className="rounded-[4px] p-4"><MetaText>下方承载页面主体内容。</MetaText></SurfaceInner>
      </div>
    </PreviewPanel>
  );
}

const notifications: Notification[] = [
  { id: "n1", message: "组件展示台已更新：Button 与 Alert 已接入全状态示例", timestamp: "刚刚", category: "notice", read: false },
  { id: "n2", message: "设计规范同步：Typography 新增 CodeText 示例", timestamp: "10 分钟前", category: "success", read: true },
];

function TopNavPreview() {
  const tabs = [
    { label: "我的 Agent", value: "/my-openclaw" },
    { label: "技能广场", value: "/skill-square" },
    { label: "模型额度", value: "/model-quota" },
  ];

  return (
    <div className="space-y-4">
      <PreviewPanel title="TopNav 组合结构" layout="wide">
        <div className="rounded-[4px] border border-[#DDE7F2] bg-white">
          <div className="overflow-x-auto">
            <div className="w-[1200px]">
              <TopNav
                className="[position:relative] top-auto"
                center={<CenterTabs activeValue="/my-openclaw" items={tabs} />}
                right={
                  <>
                    <NavIconButton icon={<HelpIcon />} label="使用指南" />
                    <NavDivider />
                    <NotificationPanel notifications={notifications} />
                    <NavDivider />
                    <NavIconButton icon={<SwitchAdminIcon />} label="管控端" />
                    <NavDivider />
                    <UserMenu username="miekoyychen" />
                  </>
                }
              />
              <div className="bg-gradient-to-b from-white to-[#F5F5F5] px-7 py-8">
                <MetaText>TopNav 作为用户端导航组件包展示，当前舞台固定 1200px 宽度；展示区较窄时通过横向滚动查看完整导航，避免中间 Tabs 与右侧功能区重叠。</MetaText>
              </div>
            </div>
          </div>
        </div>
      </PreviewPanel>
      <div className="grid grid-cols-2 gap-4">
        <PreviewPanel title="CenterTabs">
          <CenterTabs activeValue="/my-openclaw" items={tabs} />
        </PreviewPanel>
        <PreviewPanel title="NavIconButton / UserMenu">
          <div className="flex flex-wrap items-center gap-3"><NavIconButton icon={<HelpIcon />} label="使用指南" /><NavDivider /><NavIconButton icon={<SwitchAdminIcon />} label="管控端" /><NavDivider /><UserMenu username="miekoyychen" /></div>
        </PreviewPanel>
      </div>
    </div>
  );
}

function ShowcaseGoTenantIcon({ className }: { className?: string }) {
  return (
    <svg width="15" height="14" viewBox="0 0 15 14" fill="none" aria-hidden="true" className={className}>
      <path
        d="M5.83333 8.5H3.83333C1.99239 8.5 0.5 9.9924 0.5 11.8333V12.5H5.8672M11.1946 12.8347L13.5014 10.5013L11.1946 8.16801M12.8334 10.5013H7.83203M9.16667 3.5C9.16667 5.15685 7.82353 6.5 6.16667 6.5C4.50981 6.5 3.16667 5.15685 3.16667 3.5C3.16667 1.84315 4.50981 0.5 6.16667 0.5C7.82353 0.5 9.16667 1.84315 9.16667 3.5Z"
        stroke="currentColor"
        strokeLinecap="square"
      />
    </svg>
  );
}

function AdminSidebarPreview() {
  const [collapsed, setCollapsed] = useState(false);

  const collapsedIconGroups = [
    ["basic-info", "platform-policy", "user-management"],
    ["model-config", "channel-config", "skill-config", "agent-tool-library", "agent-startup-config"],
    ["agent-list", "tokens-monitor", "ops-observation"],
    ["memory-management", "file-management", "cloud-dev"],
    ["ai-agent-security", "session-management", "audit-log"],
  ];

  return (
    <PreviewPanel title="AdminSidebar 关键元素" layout="wide">
      <div className={`grid gap-8 transition-[grid-template-columns] duration-200 ${collapsed ? "grid-cols-[64px_minmax(0,1fr)]" : "grid-cols-[240px_minmax(0,1fr)]"}`}>
        <div className="admin-theme overflow-hidden rounded-[4px] border border-[#EAEEF4] bg-white [--admin-sidebar-action-bg:#ffffff] [--admin-sidebar-action-border:#EAEEF4] [--admin-sidebar-foreground:#0A0A0A] [--admin-sidebar-muted:#737373] [--admin-sidebar-item-height:34px] [--admin-sidebar-item-radius:4px]">
          <div className="flex h-[780px] flex-col overflow-hidden">
            {collapsed ? (
              <>
                <div className="flex shrink-0 flex-col items-center gap-2 px-2 py-3">
                  <Tooltip delayDuration={0}>
                    <TooltipTrigger asChild>
                      <button type="button" onClick={() => setCollapsed(false)} className="group/expand relative flex h-10 w-7 items-center justify-center rounded-[4px] text-[var(--admin-sidebar-muted)] transition-colors focus-visible:ring-2 focus-visible:ring-[var(--brand-blue)]" aria-label="展开导航">
                        <AdminSidebarLogo className="w-7 shrink-0 transition-opacity group-hover/expand:opacity-0 group-focus-visible/expand:opacity-0" />
                        <SidebarCollapseIcon className="absolute size-4 opacity-0 transition-opacity group-hover/expand:opacity-100 group-focus-visible/expand:opacity-100" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent side="right" sideOffset={8}>展开导航</TooltipContent>
                  </Tooltip>
                  <Tooltip delayDuration={0}>
                    <TooltipTrigger asChild>
                      <AdminSidebarHeaderAction aria-label="前往用户端"><ShowcaseGoTenantIcon className="size-4" /></AdminSidebarHeaderAction>
                    </TooltipTrigger>
                    <TooltipContent side="right" sideOffset={8}>前往用户端</TooltipContent>
                  </Tooltip>
                </div>
                <div className="flex-1 px-2 py-2">
                  <div className="flex flex-col">
                    {collapsedIconGroups.map((group, groupIndex) => (
                      <div key={group.join("-")}> 
                        {groupIndex > 0 && <div className="mx-2 my-2 border-t border-[#E5E5E5]" />}
                        <AdminSidebarMenu>
                          {group.map((icon) => (
                            <AdminSidebarMenuItem key={icon}>
                              <AdminSidebarMenuButton isActive={icon === "platform-policy"} className="justify-center px-0">
                                <img src={`${ADMIN_ICON_BASE}/${icon}.svg`} alt="" />
                              </AdminSidebarMenuButton>
                            </AdminSidebarMenuItem>
                          ))}
                        </AdminSidebarMenu>
                      </div>
                    ))}
                  </div>
                </div>
                <div className="relative flex h-[72px] shrink-0 items-center justify-center before:absolute before:left-4 before:right-4 before:top-0 before:h-px before:bg-[var(--admin-sidebar-border)] before:content-['']">
                  <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-[var(--admin-sidebar-avatar-bg)] font-mono text-[14.22px] leading-none text-[var(--admin-sidebar-avatar-foreground)]">J</div>
                </div>
              </>
            ) : (
              <>
                <div className="flex h-[104px] shrink-0 flex-col">
                  <div className="flex h-[72px] items-start justify-between px-4 pt-4">
                    <AdminSidebarBrand className="!gap-2">
                      <span className="flex size-11 shrink-0 items-center justify-center rounded-[8px] bg-white [box-shadow:var(--admin-sidebar-logo-shadow)]" aria-hidden="true">
                        <AdminSidebarLogo className="w-8 shrink-0" />
                      </span>
                      <div className="flex h-[42px] min-w-0 flex-col justify-center">
                        <PanelTitle as="p" className="truncate font-medium leading-[22px] tracking-[0.08px]">管控端</PanelTitle>
                        <MetaText className="truncate leading-5 tracking-[0.06px]">ClawPro Admin</MetaText>
                      </div>
                    </AdminSidebarBrand>
                    <Tooltip delayDuration={0}>
                      <TooltipTrigger asChild>
                        <button type="button" onClick={() => setCollapsed(true)} className="mt-3 flex size-5 shrink-0 items-center justify-center rounded-[4px] text-[var(--admin-sidebar-foreground)] transition-colors hover:text-[var(--text-brand)] focus-visible:ring-2 focus-visible:ring-[var(--brand-blue)]" aria-label="收起导航">
                          <SidebarCollapseIcon className="size-4" />
                        </button>
                      </TooltipTrigger>
                      <TooltipContent side="right" sideOffset={8}>收起导航</TooltipContent>
                    </Tooltip>
                  </div>
                  <AdminSidebarHeaderAction className="group mx-4 !h-8 !w-[208px] justify-center rounded-[4px] px-0 py-0 text-center !text-[var(--text-emphasis)]">
                    <span className="inline-flex items-center justify-center">
                      <MiniBodyText as="span" tone="emphasis" className="leading-5 transition-[transform] duration-300 ease-[cubic-bezier(0.22,1,0.36,1)]">
                        前往用户端
                      </MiniBodyText>
                      <span className="ml-0 inline-flex w-0 overflow-hidden transition-[width,margin] duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] group-hover:ml-1 group-hover:w-3.5">
                        <img src={goTenantArrowIcon} alt="" aria-hidden="true" className="size-3.5 shrink-0 translate-x-[-6px] opacity-0 transition-[transform,opacity] duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] group-hover:translate-x-0 group-hover:opacity-100" />
                      </span>
                    </span>
                  </AdminSidebarHeaderAction>
                </div>

                <div className="scrollbar-on-hover flex-1 overflow-y-auto px-4 pb-4 pt-4">
                  <AdminSidebarGroupLabel>基础信息</AdminSidebarGroupLabel>
                  <AdminSidebarMenu>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/basic-info.svg`} alt="" /><span className="min-w-0 flex-1 truncate">基础信息配置</span></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton isActive><img src={`${ADMIN_ICON_BASE}/platform-policy.svg`} alt="" /><span className="min-w-0 flex-1 truncate">平台策略</span><AdminSidebarBadge>New</AdminSidebarBadge></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/user-management.svg`} alt="" /><span className="min-w-0 flex-1 truncate">用户管理</span></AdminSidebarMenuButton></AdminSidebarMenuItem>
                  </AdminSidebarMenu>

                  <AdminSidebarGroupLabel className="mt-5">Agent 配置</AdminSidebarGroupLabel>
                  <AdminSidebarMenu>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/model-config.svg`} alt="" /><span className="min-w-0 flex-1 truncate">模型配置</span></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/channel-config.svg`} alt="" /><span className="min-w-0 flex-1 truncate">通道配置</span></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/skill-config.svg`} alt="" /><span className="min-w-0 flex-1 truncate">技能配置</span><AdminSidebarBadge>New</AdminSidebarBadge></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/agent-tool-library.svg`} alt="" /><span className="min-w-0 flex-1 truncate">Agent 工具库</span><AdminSidebarBadge>New</AdminSidebarBadge></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem>
                      <AdminSidebarMenuButton tone="muted" className="mt-1">
                        <img src={`${ADMIN_ICON_BASE}/agent-startup-config.svg`} alt="" />
                        <span className="min-w-0 flex-1 truncate">Agent 启动配置</span>
                        <svg className="size-3 shrink-0 text-[var(--text-muted)]" viewBox="0 0 12 12" fill="none" aria-hidden="true"><path d="M3 7.5L6 4.5L9 7.5" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" /></svg>
                      </AdminSidebarMenuButton>
                    </AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton className="pl-7"><img src={`${ADMIN_ICON_BASE}/image-management.svg`} alt="" /><span className="min-w-0 flex-1 truncate">Agent 类型</span><AdminSidebarBadge variant="custom">原镜像管理</AdminSidebarBadge></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton className="pl-7"><img src={`${ADMIN_ICON_BASE}/agent.svg`} alt="" /><span className="min-w-0 flex-1 truncate">资源管理</span><AdminSidebarBadge variant="coming-soon" /></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton className="pl-7"><img src={`${ADMIN_ICON_BASE}/network-management.svg`} alt="" /><span className="min-w-0 flex-1 truncate">网络管理</span></AdminSidebarMenuButton></AdminSidebarMenuItem>
                  </AdminSidebarMenu>

                  <AdminSidebarGroupLabel className="mt-5">运维与观测</AdminSidebarGroupLabel>
                  <AdminSidebarMenu>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/agent-list.svg`} alt="" /><span className="min-w-0 flex-1 truncate">Agent 列表</span></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/tokens-monitor.svg`} alt="" /><span className="min-w-0 flex-1 truncate">Tokens 监控</span></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/ops-observation.svg`} alt="" /><span className="min-w-0 flex-1 truncate">运维观测</span><AdminSidebarBadge>New</AdminSidebarBadge></AdminSidebarMenuButton></AdminSidebarMenuItem>
                  </AdminSidebarMenu>

                  <AdminSidebarGroupLabel className="mt-5">Agent 服务</AdminSidebarGroupLabel>
                  <AdminSidebarMenu>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/memory-management.svg`} alt="" /><span className="min-w-0 flex-1 truncate">记忆管理</span></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/file-management.svg`} alt="" /><span className="min-w-0 flex-1 truncate">网盘管理</span></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/cloud-dev.svg`} alt="" /><span className="min-w-0 flex-1 truncate">云开发管理</span><AdminSidebarBadge variant="coming-soon" /></AdminSidebarMenuButton></AdminSidebarMenuItem>
                  </AdminSidebarMenu>

                  <AdminSidebarGroupLabel className="mt-5">安全审计</AdminSidebarGroupLabel>
                  <AdminSidebarMenu>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/ai-agent-security.svg`} alt="" /><span className="min-w-0 flex-1 truncate">AI Agent 安全</span><AdminSidebarBadge>New</AdminSidebarBadge></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/session-management.svg`} alt="" /><span className="min-w-0 flex-1 truncate">会话管理</span><AdminSidebarBadge>New</AdminSidebarBadge></AdminSidebarMenuButton></AdminSidebarMenuItem>
                    <AdminSidebarMenuItem><AdminSidebarMenuButton><img src={`${ADMIN_ICON_BASE}/audit-log.svg`} alt="" /><span className="min-w-0 flex-1 truncate">操作记录</span></AdminSidebarMenuButton></AdminSidebarMenuItem>
                  </AdminSidebarMenu>
                </div>

                <AdminSidebarFooter>
                  <AdminSidebarUser name="jingsujiang" role="管理员" fallback="J" />
                  <AdminSidebarFooterAction aria-label="更多管理操作"><MoreHorizontal /></AdminSidebarFooterAction>
                </AdminSidebarFooter>
              </>
            )}
          </div>
        </div>

        <div className="flex min-w-0 flex-col justify-between py-2">
          <div>
            <PanelTitle>展示说明</PanelTitle>
            <BodyText tone="secondary" className="mt-2 max-w-xl">AdminSidebar 预览对齐真实管控端侧栏结构，展开态宽度 240px，收起态宽度 64px；点击侧栏右上角图标可切换状态。</BodyText>
            <div className="mt-5 grid gap-3">
              <div><MetaMedium>完整结构</MetaMedium><MetaText className="mt-1 block" tone="secondary">覆盖品牌区、用户端入口、一级组织、二级组织、完整菜单、底部用户区块与更多操作入口。</MetaText></div>
              <div><MetaMedium>展示高度</MetaMedium><MetaText className="mt-1 block" tone="secondary">预览舞台高度 780px，菜单区域可独立滚动，避免因展示区过低截断真实侧栏效果。</MetaText></div>
            </div>
          </div>
          <div className="pt-6 [--admin-sidebar-muted:#737373]">
            <MetaMedium className="mb-2 block">AdminSidebarBadge 状态</MetaMedium>
            <div className="flex flex-wrap items-center gap-3">
              <AdminSidebarBadge>New</AdminSidebarBadge>
              <AdminSidebarBadge variant="coming-soon" />
              <AdminSidebarBadge variant="custom">原镜像管理</AdminSidebarBadge>
            </div>
          </div>
        </div>
      </div>
    </PreviewPanel>
  );
}

function getComponentIntro(component: ComponentMeta) {
  return `用于${component.applicationScope}；${component.applicationSummary}`;
}

const USAGE_DATA = componentUsage as Record<string, { moduleCount: number; instanceCount: number; pages: ApplicationPage[] }>;

function getApplicationPages(component: ComponentMeta): ApplicationPage[] {
  if (component.applicationPages?.length) return component.applicationPages;

  // 优先使用脚本扫描出的真实应用页面
  const real = USAGE_DATA[component.id];
  if (real?.pages?.length) return real.pages;

  if (component.platform === "Tenant 用户端") {
    return [
      { name: "我的 Agent", path: "/my-openclaw", platform: "Tenant 用户端", priority: "高", usage: "用户端导航、卡片列表和主操作入口" },
      { name: "技能广场", path: "/skill-square", platform: "Tenant 用户端", priority: "高", usage: "用户端卡片、筛选和状态展示" },
      { name: "模型额度", path: "/model-quota", platform: "Tenant 用户端", priority: "中", usage: "用户端数据概览、表格和额度状态" },
    ];
  }

  if (component.platform === "Admin 管控端") {
    return [
      { name: "平台策略", path: "/admin/platform-policy", platform: "Admin 管控端", priority: "高", usage: "管控端配置卡、表单和操作说明" },
      { name: "模型配置", path: "/admin/model-config", platform: "Admin 管控端", priority: "高", usage: "管控端表单、按钮、筛选和表格操作" },
      { name: "成员管理", path: "/admin/members", platform: "Admin 管控端", priority: "中", usage: "管控端列表、弹窗和权限配置" },
    ];
  }

  if (["table", "pagination", "status-tag", "badge", "empty"].includes(component.id)) {
    return [
      { name: "模型配置", path: "/admin/model-config", platform: "Admin 管控端", priority: "高", usage: "表格、状态标签、分页和行操作" },
      { name: "Tokens 监控", path: "/admin/tokens-monitor", platform: "Admin 管控端", priority: "高", usage: "数据列表、统计和分页" },
      { name: "会话管理", path: "/admin/session-management", platform: "Admin 管控端", priority: "中", usage: "列表状态、操作列和筛选" },
    ];
  }

  if (["surface-card", "surface-inner", "surface-config", "tenant-card", "tenant-section", "typography"].includes(component.id)) {
    return [
      { name: "我的 Agent", path: "/my-openclaw", platform: "Tenant 用户端", priority: "高", usage: "用户端卡片、文字层级和页面骨架" },
      { name: "OpenClaw 详情", path: "/openclaw/1", platform: "Tenant 用户端", priority: "高", usage: "详情页标题、卡片层级和配置展示" },
      { name: "平台策略", path: "/admin/platform-policy", platform: "Admin 管控端", priority: "中", usage: "管控端配置卡和说明区" },
    ];
  }

  return [
    { name: "模型配置", path: "/admin/model-config", platform: "Admin 管控端", priority: "高", usage: "配置页常用基础组件组合" },
    { name: "成员管理", path: "/admin/members", platform: "Admin 管控端", priority: "中", usage: "表单、弹窗和列表操作" },
    { name: "我的 Agent", path: "/my-openclaw", platform: "Tenant 用户端", priority: "补充", usage: "用户端组件效果参考" },
  ];
}

/* ========== 新增 Preview 组件 ========== */

function ToastPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">类型变体</MetaMedium>
        <p className="text-xs text-[#64748B] mb-3">所有类型统一白底 + #EAEEF4 边框 · 12px 圆角 · 关闭按钮右上角</p>
        <div className="flex flex-wrap gap-3">
          <Button variant="claw-outline" size="claw-sm" onClick={() => { const id = Date.now(); toast.success(() => <>{`操作成功`}{withClose(id)}</>); }}>✓ Success</Button>
          <Button variant="claw-outline" size="claw-sm" onClick={() => { const id = Date.now(); toast.error(() => <>{`请输入用户 ID`}{withClose(id)}</>, { id }); }}>✗ Error</Button>
          <Button variant="claw-outline" size="claw-sm" onClick={() => { const id = Date.now(); toast.info(() => <>{`系统将于 10 分钟后维护`}{withClose(id)}</>, { id }); }}>ℹ Info</Button>
          <Button variant="claw-outline" size="claw-sm" onClick={() => { const id = Date.now(); toast.warning(() => <>{`配额即将用尽`}{withClose(id)}</>, { id }); }}>⚠ Warning</Button>
          <Button variant="claw-outline" size="claw-sm" onClick={() => { const id = Date.now(); toast(() => <>{`普通提示消息`}{withClose(id)}</>, { id }); }}>— Default</Button>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">长文本 / 换行</MetaMedium>
        <div className="flex flex-wrap gap-3">
          <Button variant="claw-outline" size="claw-sm" onClick={() => { const id = Date.now(); toast.error(() => <>{`操作失败：当前用户没有权限执行此操作，请联系管理员授权后重试。如需帮助请联系系统管理员。`}{withClose(id)}</>, { id }); }}>超长文本</Button>
          <Button variant="claw-outline" size="claw-sm" onClick={() => { const id = Date.now(); toast.success(() => <>{`已成功导出 2,048 条记录到 data-export-2026-06.csv`}{withClose(id)}</>, { id }); }}>带数字</Button>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">连续触发</MetaMedium>
        <Button variant="claw-outline" size="claw-sm" onClick={() => { toast.success("第 1 条"); setTimeout(() => toast.info("第 2 条"), 300); setTimeout(() => toast.warning("第 3 条"), 600); }}>连续 3 条</Button>
      </div>
      <div className="rounded-[4px] border border-[#e5e5e5] bg-[#fafafa] p-4 text-xs text-[#64748B] space-y-1">
        <p><strong>视觉规范</strong></p>
        <p>• 背景 #FFFFFF · 边框 #EAEEF4 · 圆角 12px · padding 12px 16px</p>
        <p>• 字号 14px / font-medium · 文字色 #09090b</p>
        <p>• 关闭按钮 20×20 右侧垂直居中 · hover bg-[#f4f4f5]</p>
        <p>• z-index: 99999 · 定位：顶部居中 · 阴影 shadow-lg</p>
        <p>• 禁止：按类型换底色 · 关闭按钮左上角 · 自行拼装通知 UI</p>
      </div>
    </div>
  );
}

function AvatarPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">尺寸变体</MetaMedium>
        <p className="text-xs text-[#64748B] mb-3">4 档标准尺寸 · 圆形裁切 · 首字母 Fallback</p>
        <div className="flex items-end gap-8">
          {([["h-6 w-6", "24px 行内/紧凑", "A"], ["h-8 w-8", "32px 表格/侧栏", "AB"], ["h-10 w-10", "40px 卡片头部", "JX"], ["h-12 w-12", "48px 个人中心", "MK"]] as const).map(([size, label, initials]) => (
            <div key={label} className="flex flex-col items-center gap-2">
              <Avatar className={size}><AvatarFallback>{initials}</AvatarFallback></Avatar>
              <MetaText className="text-center">{label}</MetaText>
            </div>
          ))}
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">状态变体</MetaMedium>
        <div className="flex items-end gap-8">
          <div className="flex flex-col items-center gap-2">
            <Avatar className="h-10 w-10"><AvatarImage src="https://api.dicebear.com/7.x/initials/svg?seed=JX" /><AvatarFallback>JX</AvatarFallback></Avatar>
            <MetaText>有图片</MetaText>
          </div>
          <div className="flex flex-col items-center gap-2">
            <Avatar className="h-10 w-10"><AvatarImage src="/broken-404.png" /><AvatarFallback>ER</AvatarFallback></Avatar>
            <MetaText>加载失败 → Fallback</MetaText>
          </div>
          <div className="flex flex-col items-center gap-2">
            <Avatar className="h-10 w-10"><AvatarFallback className="bg-[#EFF6FF] text-[#1447E6]">AI</AvatarFallback></Avatar>
            <MetaText>品牌色 Fallback</MetaText>
          </div>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">组合场景</MetaMedium>
        <div className="space-y-3">
          <div className="flex items-center gap-3 rounded-[4px] border border-[#e5e5e5] p-3">
            <Avatar className="h-8 w-8"><AvatarFallback>张</AvatarFallback></Avatar>
            <div><div className="text-sm font-medium text-[#0A0A0A]">张三</div><MetaText>管理员 · 最近在线 2 分钟前</MetaText></div>
          </div>
          <div className="flex items-center gap-3 rounded-[4px] border border-[#e5e5e5] p-3">
            <Avatar className="h-10 w-10"><AvatarFallback className="bg-[#EFF6FF] text-[#1447E6]">🤖</AvatarFallback></Avatar>
            <div><div className="text-sm font-medium text-[#0A0A0A]">Alice 的技术助手</div><MetaText>Agent · GPT-4o · 运行中</MetaText></div>
          </div>
        </div>
      </div>
    </div>
  );
}

function TreePreviewSection() {
  const [activeId, setActiveId] = useState("1-1");
  const [expanded, setExpanded] = useState<Record<string, boolean>>({ "1": true, "1-3": true });
  const toggle = (id: string) => setExpanded((p) => ({ ...p, [id]: !p[id] }));

  const nodes = [
    { id: "1", label: "全部用户", count: 128, children: [
      { id: "1-1", label: "管理员组", count: 5 },
      { id: "1-2", label: "普通用户组", count: 98 },
      { id: "1-3", label: "开发团队", count: 25, children: [
        { id: "1-3-1", label: "前端组", count: 12 },
        { id: "1-3-2", label: "后端组", count: 8 },
      ]},
    ]},
    { id: "2", label: "已归档", count: 15, disabled: true, children: [
      { id: "2-1", label: "2025 年归档", count: 10, disabled: true },
    ]},
  ];

  const renderNode = (node: any, depth: number) => {
    const hasChildren = !!node.children?.length;
    const isActive = activeId === node.id;
    const isExpanded = expanded[node.id];
    return (
      <div key={node.id}>
        <div
          className={`group flex items-center gap-1.5 h-8 pr-3 text-sm cursor-pointer rounded-[4px] transition-colors ${node.disabled ? "text-[#a1a1aa] cursor-not-allowed opacity-60" : isActive ? "bg-[#f4f4f5] text-[#09090b] font-medium" : "text-[#09090b] hover:bg-[#f4f4f5]"}`}
          style={{ paddingLeft: 8 + depth * 16 }}
          onClick={() => { if (!node.disabled) { setActiveId(node.id); if (hasChildren) toggle(node.id); } }}
        >
          <span className={`w-4 h-4 flex items-center justify-center text-[#71717a] shrink-0 transition-transform ${hasChildren && isExpanded ? "rotate-90" : ""}`}>
            {hasChildren && <ChevronRight className="w-3.5 h-3.5" />}
          </span>
          <span className="w-4 h-4 flex items-center justify-center text-[#71717a] shrink-0">
            {hasChildren ? (isExpanded ? <FolderOpen className="w-3.5 h-3.5" /> : <Folder className="w-3.5 h-3.5" />) : <FileText className="w-3.5 h-3.5" />}
          </span>
          <span className="truncate">{node.label}</span>
          {node.count != null && <span className="text-[11px] tabular-nums shrink-0 text-[#a1a1aa]">({node.count})</span>}
        </div>
        {isExpanded && hasChildren && node.children.map((c: any) => renderNode(c, depth + 1))}
      </div>
    );
  };

  return (
    <div className="space-y-4">
      <p className="text-xs text-[#64748B]">行高 32px · 图标色 #71717a · 缩进 16px/层 · 选中态 #f4f4f5</p>
      <div className="max-w-[320px] rounded-[4px] border border-[#e5e5e5] bg-white p-2">
        <div className="flex flex-col gap-0.5">{nodes.map((n) => renderNode(n, 0))}</div>
      </div>
    </div>
  );
}

function BreadcrumbPreviewSection() {
  const examples = [
    [{ label: "首页", href: "#" }, { label: "用户管理" }],
    [{ label: "首页", href: "#" }, { label: "Agent 列表", href: "#" }, { label: "Alice 的技术助手" }],
    [{ label: "首页", href: "#" }, { label: "安全管理", href: "#" }, { label: "AI Agent", href: "#" }, { label: "策略详情" }],
  ];
  return (
    <div className="space-y-6">
      <p className="text-xs text-[#64748B]">祖先页灰色可点击 · 当前页深色不可点 · 分隔符 /</p>
      {examples.map((items, i) => (
        <div key={i}>
          <MetaText className="mb-2 block">{items.length} 级</MetaText>
          <nav className="flex items-center gap-1.5 text-sm">
            {items.map((item: any, j: number) => (
              <span key={j} className="flex items-center gap-1.5">
                {j > 0 && <span className="text-[#94A3B8]">/</span>}
                {item.href ? (
                  <a href={item.href} className="text-[#737373] hover:text-[#0F172A]">{item.label}</a>
                ) : (
                  <span className="font-medium text-[#0F172A]">{item.label}</span>
                )}
              </span>
            ))}
          </nav>
        </div>
      ))}
    </div>
  );
}

function TransferPreviewSection() {
  const [selected, setSelected] = useState<string[]>(["item-1", "item-3"]);
  const mockData = Array.from({ length: 12 }, (_, i) => ({
    key: `item-${i}`,
    name: `Agent-${String(i + 1).padStart(3, "0")}`,
    ip: `10.0.0.${i + 1}`,
    version: i % 3 === 0 ? "基础版" : "旗舰版",
  }));
  return (
    <div className="space-y-4">
      <p className="text-xs text-[#64748B]">instant 模式 · 左侧勾选立即搬右 · Table compact 密度 · 弹窗内紧凑分页</p>
      <Transfer<any>
        dataSource={mockData}
        rowKey="key"
        targetKeys={selected}
        onChange={(nextKeys) => setSelected(nextKeys)}
        showSearch
        searchPlaceholder={["搜索名称 / IP", "搜索已选"]}
        pagination={{ pageSize: 5 }}
        height={240}
        titles={["全部资产", "已选资产"]}
        columns={[
          { key: "name", header: "名称", render: (h: any) => h.name },
          { key: "ip", header: "IP", width: 110, render: (h: any) => h.ip },
          { key: "version", header: "版本", width: 80, render: (h: any) => h.version },
        ]}
      />
    </div>
  );
}

function SearchFilterBarPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">基础筛选条</MetaMedium>
        <p className="text-xs text-[#64748B] mb-3">搜索框 + 状态筛选 + 刷新 · gap-3</p>
        <div className="flex items-center gap-3">
          <div className="relative flex-1 max-w-[280px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#94A3B8]" />
            <Input className="pl-9" placeholder="搜索名称 / ID" />
          </div>
          <Select><SelectTrigger className="w-[140px]"><SelectValue placeholder="全部状态" /></SelectTrigger><SelectContent><SelectItem value="all">全部</SelectItem><SelectItem value="running">运行中</SelectItem><SelectItem value="stopped">已停止</SelectItem></SelectContent></Select>
          <Button variant="claw-outline" size="icon" className="w-9 h-9"><RefreshCw className="w-4 h-4" /></Button>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">带日期 + 多筛选</MetaMedium>
        <div className="flex items-center gap-3">
          <div className="relative flex-1 max-w-[240px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#94A3B8]" />
            <Input className="pl-9" placeholder="搜索" />
          </div>
          <Select><SelectTrigger className="w-[120px]"><SelectValue placeholder="类型" /></SelectTrigger><SelectContent><SelectItem value="all">全部</SelectItem></SelectContent></Select>
          <Select><SelectTrigger className="w-[120px]"><SelectValue placeholder="级别" /></SelectTrigger><SelectContent><SelectItem value="all">全部</SelectItem></SelectContent></Select>
          <DatePicker />
          <Button variant="claw-outline" size="icon" className="w-9 h-9"><RefreshCw className="w-4 h-4" /></Button>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">紧凑版（弹窗内）</MetaMedium>
        <div className="flex items-center gap-2">
          <div className="relative flex-1 max-w-[200px]">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[#94A3B8]" />
            <Input className="pl-8 h-8 text-xs" placeholder="搜索" />
          </div>
          <Select><SelectTrigger className="w-[100px] h-8 text-xs"><SelectValue placeholder="筛选" /></SelectTrigger><SelectContent><SelectItem value="all">全部</SelectItem></SelectContent></Select>
        </div>
      </div>
      <div className="rounded-[4px] border border-[#e5e5e5] bg-[#fafafa] p-4 text-xs text-[#64748B] space-y-1">
        <p><strong>规则</strong></p>
        <p>• 搜索框左侧 Search 图标，图标用 --text-weak</p>
        <p>• 筛选区用 gap-3，不要每个控件单独写 margin</p>
        <p>• 刷新按钮标准写法：Button variant="claw-outline" size="icon" w-9 h-9</p>
        <p>• Popover/Select 内容通过 Portal 逃逸 overflow 裁剪</p>
      </div>
    </div>
  );
}

function BatchActionsBarPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">部分选中状态</MetaMedium>
        <div className="relative rounded-[4px] border border-[var(--cp-border)] bg-white overflow-hidden">
          <Table>
            <TableHeader><TableRow><TableHead className="w-[44px]"><Checkbox checked="indeterminate" /></TableHead><TableHead>名称</TableHead><TableHead>状态</TableHead></TableRow></TableHeader>
            <TableBody>
              <TableRow data-state="selected"><TableCell className="w-[44px]"><Checkbox checked /></TableCell><TableCell>Agent-001</TableCell><TableCell><StatusTag mode="text" variant="green">运行中</StatusTag></TableCell></TableRow>
              <TableRow data-state="selected"><TableCell className="w-[44px]"><Checkbox checked /></TableCell><TableCell>Agent-002</TableCell><TableCell><StatusTag mode="text" variant="red">已停止</StatusTag></TableCell></TableRow>
              <TableRow><TableCell className="w-[44px]"><Checkbox /></TableCell><TableCell>Agent-003</TableCell><TableCell><StatusTag mode="text" variant="green">运行中</StatusTag></TableCell></TableRow>
            </TableBody>
          </Table>
          <div className="sticky bottom-0 flex items-center justify-between gap-4 border-t border-[#EAEEF4] bg-white px-4 py-2.5">
            <div className="flex items-center gap-2">
              <MetaText>已选择 <span className="text-[#1447E6] font-medium">2</span> 项</MetaText>
              <span className="text-xs text-[#94A3B8]">·</span>
              <button className="text-xs text-[#1447E6] hover:underline">选择全部 156 项</button>
            </div>
            <div className="flex items-center gap-3">
              <Button variant="claw-outline" size="claw-sm">批量删除</Button>
              <Button variant="claw-outline" size="claw-sm">批量导出</Button>
              <Button variant="ghost" size="claw-sm">取消选择</Button>
            </div>
          </div>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">全选 / 跨页选择态</MetaMedium>
        <div className="rounded-[4px] border border-[#EAEEF4] bg-[#EFF6FF] px-4 py-2.5 flex items-center justify-between">
          <MetaText>已选择全部 <span className="text-[#1447E6] font-medium">156</span> 项（跨页）</MetaText>
          <div className="flex items-center gap-3">
            <Button variant="claw-outline" size="claw-sm">批量删除</Button>
            <Button variant="ghost" size="claw-sm">取消选择</Button>
          </div>
        </div>
      </div>
      <div className="rounded-[4px] border border-[#e5e5e5] bg-[#fafafa] p-4 text-xs text-[#64748B] space-y-1">
        <p><strong>规则</strong></p>
        <p>• sticky bottom，表格勾选后浮出</p>
        <p>• 左侧：已选数量 + 跨页全选链接</p>
        <p>• 右侧：批量操作按钮 + 取消选择</p>
        <p>• 全选态时背景变浅蓝 #EFF6FF</p>
      </div>
    </div>
  );
}

function ChartStatPreviewSection() {
  return (
    <div className="space-y-4">
      <p className="text-xs text-[#64748B]">NumberCard 标准数字卡片 · 折线图/环形图占位 · DIN 数字字体</p>
      <div className="grid grid-cols-4 gap-4">
        <NumberCard icon={<TotalTokensIcon />} label="总 Agent 数" value="128" />
        <NumberCard icon={<RequestsIcon />} label="运行中" value="96" />
        <NumberCard icon={<InputTokensIcon />} label="已停止" value="24" />
        <NumberCard icon={<OutputTokensIcon />} label="今日请求" value="12,840" />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="rounded-[4px] border border-[#e5e5e5] bg-white p-4 h-[160px] flex items-center justify-center">
          <MetaText tone="weak">折线图占位（recharts）</MetaText>
        </div>
        <div className="rounded-[4px] border border-[#e5e5e5] bg-white p-4 h-[160px] flex items-center justify-center">
          <MetaText tone="weak">环形图占位（recharts）</MetaText>
        </div>
      </div>
    </div>
  );
}

function UploadPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">拖拽上传区（默认态）</MetaMedium>
        <div className="rounded-[4px] border border-dashed border-[#EAEEF4] bg-white p-8 text-center hover:border-[#1447E6] transition-colors cursor-pointer">
          <div className="text-sm text-[#737373]">拖拽文件到此处，或 <span className="text-[#1447E6] hover:underline">点击上传</span></div>
          <MetaText className="mt-1 block">支持 .csv / .xlsx / .json，单个文件不超过 10MB</MetaText>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">拖拽上传区（Disabled 态）</MetaMedium>
        <div className="rounded-[4px] border border-dashed border-[#EAEEF4] bg-[#f3f3f4] p-8 text-center cursor-not-allowed opacity-60">
          <div className="text-sm text-[#737373]">拖拽文件到此处，或 <span className="text-[#A3A3A3]">点击上传</span></div>
          <MetaText className="mt-1 block">上传功能暂不可用</MetaText>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">文件列表（多状态）</MetaMedium>
        <div className="space-y-2">
          <div className="flex items-center gap-3 rounded-[4px] border border-[#e5e5e5] bg-white px-3 py-2">
            <FileText className="w-4 h-4 text-[#71717a] shrink-0" />
            <span className="text-sm flex-1 truncate">data-export-2026-06.csv</span>
            <MetaText>2.4 MB</MetaText>
            <Progress value={45} className="w-20 h-1.5" />
            <MetaText tone="muted">45%</MetaText>
            <Button variant="ghost" size="icon" className="w-6 h-6"><X className="w-3.5 h-3.5" /></Button>
          </div>
          <div className="flex items-center gap-3 rounded-[4px] border border-[#e5e5e5] bg-white px-3 py-2">
            <FileText className="w-4 h-4 text-[#71717a] shrink-0" />
            <span className="text-sm flex-1 truncate">config-backup.json</span>
            <MetaText>128 KB</MetaText>
            <MetaText tone="brand">上传完成</MetaText>
            <Button variant="ghost" size="icon" className="w-6 h-6"><X className="w-3.5 h-3.5" /></Button>
          </div>
          <div className="flex items-center gap-3 rounded-[4px] border border-[#d42a1e]/30 bg-[#FEF2F2] px-3 py-2">
            <FileText className="w-4 h-4 text-[#d42a1e] shrink-0" />
            <span className="text-sm flex-1 truncate">too-large-file.zip</span>
            <MetaText>156 MB</MetaText>
            <MetaText tone="danger">超出大小限制</MetaText>
            <Button variant="ghost" size="icon" className="w-6 h-6"><X className="w-3.5 h-3.5" /></Button>
          </div>
        </div>
      </div>
      <div className="rounded-[4px] border border-[#e5e5e5] bg-[#fafafa] p-4 text-xs text-[#64748B] space-y-1">
        <p><strong>规则</strong></p>
        <p>• 拖拽区：dashed 边框 1px，hover 变品牌蓝</p>
        <p>• 文件列表：名称 + 大小 + 进度/状态 + 删除按钮</p>
        <p>• 错误态：红色边框 + 红色文字提示</p>
        <p>• 禁止使用默认 Upload 图标</p>
      </div>
    </div>
  );
}

function TagPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">基础 Tag</MetaMedium>
        <p className="text-xs text-[#64748B] mb-3">22px 高 · 4px 圆角 · px-2 · 12px 文字</p>
        <div className="flex flex-wrap gap-2">
          <span className="inline-flex items-center h-[22px] px-2 rounded-[4px] bg-[#F5F5F5] text-xs text-[#1E293B]">默认标签</span>
          <span className="inline-flex items-center h-[22px] px-2 rounded-[4px] bg-[#F5F5F5] text-xs text-[#1E293B]">分类 A</span>
          <span className="inline-flex items-center h-[22px] px-2 rounded-[4px] bg-[#F5F5F5] text-xs text-[#1E293B]">分类 B</span>
          <span className="inline-flex items-center h-[22px] px-2 rounded-[4px] bg-[#F5F5F5] text-xs text-[#1E293B]">分类 C</span>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">可关闭 Tag</MetaMedium>
        <div className="flex flex-wrap gap-2">
          <span className="inline-flex items-center gap-1 h-[22px] px-2 rounded-[4px] bg-[#F5F5F5] text-xs text-[#1E293B]">可删除 <X className="w-3 h-3 text-[#737373] cursor-pointer hover:text-[#0A0A0A]" /></span>
          <span className="inline-flex items-center gap-1 h-[22px] px-2 rounded-[4px] bg-[#F5F5F5] text-xs text-[#1E293B]">另一个 <X className="w-3 h-3 text-[#737373] cursor-pointer hover:text-[#0A0A0A]" /></span>
          <span className="inline-flex items-center gap-1 h-[22px] px-2 rounded-[4px] bg-[#F5F5F5] text-xs text-[#1E293B]">第三个 <X className="w-3 h-3 text-[#737373] cursor-pointer hover:text-[#0A0A0A]" /></span>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">彩色分类 Tag（预定义色板）</MetaMedium>
        <p className="text-xs text-[#64748B] mb-3">新增颜色需先在 token 层登记，禁止业务侧自拼</p>
        <div className="flex flex-wrap gap-2">
          <span className="inline-flex items-center h-[22px] px-2 rounded-[4px] bg-[#E8ECFE] text-xs text-[#1447E6]">Blue 蓝色</span>
          <span className="inline-flex items-center h-[22px] px-2 rounded-[4px] bg-[#E9F8EB] text-xs text-[#008236]">Green 绿色</span>
          <span className="inline-flex items-center h-[22px] px-2 rounded-[4px] bg-[#FFFBEB] text-xs text-[#B45309]">Orange 橙色</span>
          <span className="inline-flex items-center h-[22px] px-2 rounded-[4px] bg-[#F5F5F5] text-xs text-[#0A0A0A]">Gray 灰色</span>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">带边框变体（soft 模式）</MetaMedium>
        <div className="flex flex-wrap gap-2">
          <span className="inline-flex items-center h-[22px] px-2 rounded-[4px] bg-[#E8ECFE] border border-[#C7D7FE] text-xs text-[#1447E6]">Blue soft</span>
          <span className="inline-flex items-center h-[22px] px-2 rounded-[4px] bg-[#E9F8EB] border border-[#BFE8C8] text-xs text-[#008236]">Green soft</span>
          <span className="inline-flex items-center h-[22px] px-2 rounded-[4px] bg-[#FFFBEB] border border-[#FDE68A] text-xs text-[#B45309]">Orange soft</span>
          <span className="inline-flex items-center h-[22px] px-2 rounded-[4px] bg-[#F5F5F5] border border-[#E5E5E5] text-xs text-[#0A0A0A]">Gray soft</span>
        </div>
      </div>
      <div className="rounded-[4px] border border-[#e5e5e5] bg-[#fafafa] p-4 text-xs text-[#64748B] space-y-1">
        <p><strong>区分 Tag vs StatusBadge</strong></p>
        <p>• Tag：用户自建标签、分类标签、筛选 chip → 4px 圆角</p>
        <p>• StatusBadge：运行状态（正常/异常/停止）→ rounded-full</p>
        <p>• 禁止混用：不要用 Tag 表达运行状态，不要用 StatusBadge 当分类标签</p>
      </div>
    </div>
  );
}

function AccordionPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">单项展开（type=single）</MetaMedium>
        <Accordion type="single" collapsible className="w-full max-w-md">
          <AccordionItem value="item-1">
            <AccordionTrigger>什么是 ClawPro？</AccordionTrigger>
            <AccordionContent>ClawPro 是一个企业级 AI Agent 管理平台，提供模型配置、技能编排和安全管控能力。</AccordionContent>
          </AccordionItem>
          <AccordionItem value="item-2">
            <AccordionTrigger>如何创建 Agent？</AccordionTrigger>
            <AccordionContent>进入管控端 → Agent 管理 → 点击"新建 Agent"按钮，按引导完成配置。</AccordionContent>
          </AccordionItem>
          <AccordionItem value="item-3">
            <AccordionTrigger>支持哪些大模型？</AccordionTrigger>
            <AccordionContent>支持 GPT-4、Claude、文心一言、混元等主流大模型接入。</AccordionContent>
          </AccordionItem>
        </Accordion>
      </div>
      <div>
        <MetaMedium className="mb-2 block">多项展开（type=multiple）</MetaMedium>
        <Accordion type="multiple" className="w-full max-w-md">
          <AccordionItem value="a">
            <AccordionTrigger>配置说明</AccordionTrigger>
            <AccordionContent>可同时展开多个面板查看详情信息。</AccordionContent>
          </AccordionItem>
          <AccordionItem value="b">
            <AccordionTrigger>使用须知</AccordionTrigger>
            <AccordionContent>多项展开模式下无互斥限制。</AccordionContent>
          </AccordionItem>
        </Accordion>
      </div>
    </div>
  );
}

function CardPreviewSection() {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 gap-4 max-w-2xl">
        <Card>
          <CardHeader>
            <CardTitleUI>卡片标题</CardTitleUI>
            <CardDescription>这是卡片的描述信息，支持多行文字。</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-[#475569]">卡片内容区域，可以放置任意组件。</p>
          </CardContent>
          <CardFooter className="flex justify-between">
            <Button variant="outline" size="sm">取消</Button>
            <Button size="sm">确认</Button>
          </CardFooter>
        </Card>
        <Card>
          <CardHeader>
            <CardTitleUI>统计卡片</CardTitleUI>
            <CardDescription>含数据展示</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">1,234</div>
            <p className="text-xs text-[#737373]">较上月 +12%</p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function DropdownMenuPreviewSection() {
  return (
    <div className="space-y-6">
      <div className="flex gap-4">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline">打开菜单</Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>查看详情</DropdownMenuItem>
            <DropdownMenuItem>编辑配置</DropdownMenuItem>
            <DropdownMenuItem>复制链接</DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-red-600">删除</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm"><MoreHorizontal className="size-4" /></Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>导出</DropdownMenuItem>
            <DropdownMenuItem>分享</DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-red-600">移除</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}

function LineTabsPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">默认状态</MetaMedium>
        <LineTabs
          tabs={[{ id: "overview", label: "概览" }, { id: "config", label: "配置" }, { id: "logs", label: "日志" }, { id: "settings", label: "设置" }]}
          active="overview"
          onChange={() => {}}
        />
      </div>
      <div>
        <MetaMedium className="mb-2 block">选中第二项</MetaMedium>
        <LineTabs
          tabs={[{ id: "members", label: "成员管理" }, { id: "roles", label: "角色权限" }, { id: "security", label: "安全策略" }]}
          active="roles"
          onChange={() => {}}
        />
      </div>
    </div>
  );
}

function SheetPreviewSection() {
  return (
    <div className="space-y-6">
      <div className="flex gap-4">
        <Sheet>
          <SheetTrigger asChild>
            <Button variant="outline">打开右侧面板</Button>
          </SheetTrigger>
          <SheetContent>
            <SheetHeader>
              <SheetTitle>面板标题</SheetTitle>
              <SheetDescription>这是一个从右侧滑出的面板，可放置详情或编辑表单。</SheetDescription>
            </SheetHeader>
            <div className="py-4">
              <p className="text-sm text-[#475569]">面板内容区域</p>
            </div>
          </SheetContent>
        </Sheet>
      </div>
    </div>
  );
}

function SkeletonPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">卡片骨架屏</MetaMedium>
        <div className="w-[320px] space-y-3 rounded-[4px] border border-[#DDE7F2] p-4">
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-3 w-full" />
          <Skeleton className="h-3 w-5/6" />
          <Skeleton className="h-8 w-24 mt-2" />
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">列表骨架屏</MetaMedium>
        <div className="space-y-2 max-w-md">
          {[1, 2, 3].map((i) => (
            <div key={i} className="flex items-center gap-3">
              <Skeleton className="size-10 rounded-full" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-3 w-1/2" />
                <Skeleton className="h-3 w-3/4" />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function SliderPreviewSection() {
  return (
    <div className="space-y-6 max-w-md">
      <div>
        <MetaMedium className="mb-2 block">默认滑块</MetaMedium>
        <Slider defaultValue={[50]} max={100} step={1} />
      </div>
      <div>
        <MetaMedium className="mb-2 block">范围滑块</MetaMedium>
        <Slider defaultValue={[25, 75]} max={100} step={1} />
      </div>
      <div>
        <MetaMedium className="mb-2 block">带刻度</MetaMedium>
        <Slider defaultValue={[30]} max={100} step={10} />
        <div className="flex justify-between mt-1 text-xs text-[#737373]">
          <span>0</span><span>50</span><span>100</span>
        </div>
      </div>
    </div>
  );
}

function SeparatorPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">水平分割线</MetaMedium>
        <div className="max-w-md space-y-3">
          <p className="text-sm">上方内容</p>
          <Separator />
          <p className="text-sm">下方内容</p>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">垂直分割线</MetaMedium>
        <div className="flex items-center gap-3 h-6">
          <span className="text-sm">选项 A</span>
          <Separator orientation="vertical" />
          <span className="text-sm">选项 B</span>
          <Separator orientation="vertical" />
          <span className="text-sm">选项 C</span>
        </div>
      </div>
    </div>
  );
}

function ScrollAreaPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">垂直滚动</MetaMedium>
        <ScrollArea className="h-[200px] w-[300px] rounded-[4px] border border-[#DDE7F2] p-4">
          <div className="space-y-3">
            {Array.from({ length: 20 }, (_, i) => (
              <div key={i} className="text-sm border-b border-[#DDE7F2] pb-2">列表项 {i + 1} — 示例内容行</div>
            ))}
          </div>
        </ScrollArea>
      </div>
    </div>
  );
}

function CollapsiblePreviewSection() {
  return (
    <div className="space-y-6 max-w-md">
      <Collapsible>
        <div className="flex items-center justify-between rounded-[4px] border border-[#DDE7F2] px-4 py-2">
          <span className="text-sm font-medium">高级设置</span>
          <CollapsibleTrigger asChild>
            <Button variant="ghost" size="sm"><ChevronRight className="size-4" /></Button>
          </CollapsibleTrigger>
        </div>
        <CollapsibleContent className="mt-2 space-y-2 rounded-[4px] border border-[#DDE7F2] px-4 py-3">
          <p className="text-sm text-[#475569]">超时设置：30s</p>
          <p className="text-sm text-[#475569]">重试次数：3</p>
          <p className="text-sm text-[#475569]">并发限制：10</p>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}

function ToggleGroupPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">单选模式</MetaMedium>
        <ToggleGroup type="single" defaultValue="center">
          <ToggleGroupItem value="left" aria-label="左对齐">左</ToggleGroupItem>
          <ToggleGroupItem value="center" aria-label="居中">中</ToggleGroupItem>
          <ToggleGroupItem value="right" aria-label="右对齐">右</ToggleGroupItem>
        </ToggleGroup>
      </div>
      <div>
        <MetaMedium className="mb-2 block">多选模式</MetaMedium>
        <ToggleGroup type="multiple">
          <ToggleGroupItem value="bold" aria-label="粗体"><span className="font-bold">B</span></ToggleGroupItem>
          <ToggleGroupItem value="italic" aria-label="斜体"><span className="italic">I</span></ToggleGroupItem>
          <ToggleGroupItem value="underline" aria-label="下划线"><span className="underline">U</span></ToggleGroupItem>
        </ToggleGroup>
      </div>
    </div>
  );
}

function HoverCardPreviewSection() {
  return (
    <div className="space-y-6">
      <HoverCard>
        <HoverCardTrigger asChild>
          <Button variant="link">悬停查看用户信息 @admin</Button>
        </HoverCardTrigger>
        <HoverCardContent className="w-80">
          <div className="flex gap-3">
            <Avatar><AvatarFallback>A</AvatarFallback></Avatar>
            <div>
              <p className="text-sm font-medium">Admin 管理员</p>
              <p className="text-xs text-[#737373]">admin@company.com</p>
              <p className="text-xs text-[#737373] mt-1">最后登录：2 小时前</p>
            </div>
          </div>
        </HoverCardContent>
      </HoverCard>
    </div>
  );
}

function ContextMenuPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">右键此区域触发菜单</MetaMedium>
        <ContextMenu>
          <ContextMenuTrigger className="flex h-[150px] w-[300px] items-center justify-center rounded-[4px] border border-dashed border-[#DDE7F2] text-sm text-[#737373]">
            右键点击此区域
          </ContextMenuTrigger>
          <ContextMenuContent>
            <ContextMenuItem>复制</ContextMenuItem>
            <ContextMenuItem>粘贴</ContextMenuItem>
            <ContextMenuItem>刷新</ContextMenuItem>
            <ContextMenuItem className="text-red-600">删除</ContextMenuItem>
          </ContextMenuContent>
        </ContextMenu>
      </div>
    </div>
  );
}

function AllUsersTagPreviewSection() {
  return (
    <div className="space-y-6">
      <div className="flex gap-3 items-center">
        <AllUsersTag />
        <MetaText>← 管控端"全部用户"统一标签</MetaText>
      </div>
    </div>
  );
}

function BackButtonPreviewSection() {
  return (
    <div className="space-y-6">
      <div className="flex gap-4 items-center">
        <BackButton />
        <MetaText>← 标准返回按钮</MetaText>
      </div>
    </div>
  );
}

function FavoriteButtonPreviewSection() {
  return (
    <div className="space-y-6">
      <div className="flex gap-4 items-center">
        <FavoriteButton isFavorited={false} onToggle={() => {}} />
        <FavoriteButton isFavorited={true} onToggle={() => {}} />
        <MetaText>未收藏 / 已收藏</MetaText>
      </div>
    </div>
  );
}

function MoreActionsDropdownPreviewSection() {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
        <VariantCell tag="单选 · 即时 · 扁平" api="<MoreActionsDropdown />">
          <MoreActionsDropdown
            items={[
              { label: "查看详情", icon: FileText, onClick: () => {} },
              { label: "编辑", icon: Settings, onClick: () => {} },
              { label: "删除", icon: X, onClick: () => {}, variant: "destructive" },
            ]}
          />
        </VariantCell>
        <VariantCell tag="单选 · 即时 · 扁平（文字触发）" api='<MoreActionsDropdown triggerType="text" />'>
          <MoreActionsDropdown
            triggerType="text"
            items={[
              { label: "查看详情", icon: FileText, onClick: () => {} },
              { label: "编辑", icon: Settings, onClick: () => {} },
            ]}
          />
        </VariantCell>
      </div>
    </div>
  );
}

function FileBrowserPreviewSection() {
  const versions: VersionInfo[] = useMemo(() => [
    {
      version: "v1.3.0",
      date: "2026-06-05",
      isLatest: true,
      changeLog: "新增 SKILL.md 入口导航；补齐组件清单与协作流程。",
    },
    {
      version: "v1.2.0",
      date: "2026-05-21",
      changeLog: "重构 src/index.ts，调整 portable 模板路径。",
    },
    {
      version: "v1.1.0",
      date: "2026-05-08",
      changeLog: "首次接入 FileTree，移除老版自写浏览器。",
    },
  ], []);

  const filesByVersion: Record<string, FileEntry[]> = useMemo(() => ({
    "v1.3.0": [
      { name: "SKILL.md", size: 8240 },
      { name: "README.md", size: 1024 },
      { name: "src/index.ts", size: 2310 },
      { name: "src/utils/format.ts", size: 540 },
      { name: "portable/css/tokens.css", size: 1812 },
      { name: "portable/html-css/file-browser.html", size: 4860 },
      { name: "package.json", size: 412 },
    ],
    "v1.2.0": [
      { name: "SKILL.md", size: 7200 },
      { name: "src/index.ts", size: 2080 },
      { name: "package.json", size: 412 },
    ],
    "v1.1.0": [
      { name: "SKILL.md", size: 6420 },
      { name: "src/index.ts", size: 1980 },
    ],
  }), []);

  const contents: Record<string, string> = useMemo(() => ({
    "SKILL.md": "# Portable Design Skill\n\n这是一个示例 SKILL.md，用于演示 FileBrowser 默认会优先选中并以 Preview 模式渲染。\n\n## 章节\n\n- 入口导航\n- 组件清单\n- 走查标准\n",
    "README.md": "# README\n\n资产包说明文档。",
    "src/index.ts": "export { FileBrowser } from \"./file-browser\";\nexport type { VersionInfo } from \"./file-browser\";\n",
    "src/utils/format.ts": "export const formatSize = (n: number) => `${(n / 1024).toFixed(1)} KB`;\n",
    "portable/css/tokens.css": ":root {\n  --bg-page: #f8fafc;\n  --border-soft: #e5e7eb;\n}\n",
    "portable/html-css/file-browser.html": "<!doctype html>\n<html>\n  <body>\n    <div class=\"file-browser\">...</div>\n  </body>\n</html>\n",
    "package.json": "{\n  \"name\": \"portable-design-skill\",\n  \"version\": \"1.3.0\"\n}\n",
  }), []);

  const [currentVersion, setCurrentVersion] = useState("v1.3.0");
  const files = filesByVersion[currentVersion] || [];

  return (
    <div className="space-y-3">
      <p className="text-xs text-[#64748B]">
        三栏布局 · 版本列 14% / 文件树 22% / 内容 flex-1 · 默认优先选中 SKILL.md · .md 文件自动 Preview
      </p>
      <FileBrowser
        versions={versions}
        files={files}
        getFileContent={(name) => contents[name]}
        height="36rem"
        showDownload
        onDownload={(v) => toast.info(`模拟下载 ${v}`)}
        onVersionChange={(v) => setCurrentVersion(v)}
      />
    </div>
  );
}

function CarouselPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">基础轮播（3 张幻灯片）</MetaMedium>
        <Carousel className="w-full max-w-md">
          <CarouselContent>
            {[1, 2, 3].map((n) => (
              <CarouselItem key={n}>
                <div className="flex aspect-[16/9] items-center justify-center rounded-[4px] border border-[#DDE7F2] bg-[#F4F8FC]">
                  <span className="text-3xl font-semibold text-[#1447E6]">幻灯片 {n}</span>
                </div>
              </CarouselItem>
            ))}
          </CarouselContent>
          <CarouselPrevious />
          <CarouselNext />
        </Carousel>
      </div>
    </div>
  );
}

function FormPreviewSection() {
  const form = useForm<{ email: string; name: string }>({
    defaultValues: { email: "", name: "" },
  });
  const onSubmit = (values: { email: string; name: string }) => {
    toast.info(`提交：${JSON.stringify(values)}`);
  };
  return (
    <div className="space-y-6 max-w-md">
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <FormField
            control={form.control}
            name="name"
            rules={{ required: "请输入名称" }}
            render={({ field }) => (
              <FormItem>
                <FormLabel>名称</FormLabel>
                <FormControl>
                  <Input placeholder="请输入名称" {...field} />
                </FormControl>
                <FormDescription>用于在控制台中展示的显示名。</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="email"
            rules={{
              required: "请输入邮箱",
              pattern: { value: /.+@.+\..+/, message: "邮箱格式不合法" },
            }}
            render={({ field }) => (
              <FormItem>
                <FormLabel>邮箱</FormLabel>
                <FormControl>
                  <Input type="email" placeholder="name@example.com" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <div className="flex gap-2">
            <Button type="submit">提交</Button>
            <Button type="button" variant="outline" onClick={() => form.reset()}>
              重置
            </Button>
          </div>
        </form>
      </Form>
    </div>
  );
}

function CalendarPreviewSection() {
  const [date, setDate] = useState<Date | undefined>(new Date());
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">单选模式（默认）</MetaMedium>
        <div className="inline-block rounded-[4px] border border-[#DDE7F2] bg-white">
          <Calendar mode="single" selected={date} onSelect={setDate} />
        </div>
        <p className="mt-2 text-xs text-[#737373]">已选：{date ? date.toLocaleDateString() : "无"}</p>
      </div>
    </div>
  );
}

function InputOTPPreviewSection() {
  const [value, setValue] = useState("");
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">6 位验证码（含分隔符）</MetaMedium>
        <InputOTP maxLength={6} value={value} onChange={setValue}>
          <InputOTPGroup>
            <InputOTPSlot index={0} />
            <InputOTPSlot index={1} />
            <InputOTPSlot index={2} />
          </InputOTPGroup>
          <InputOTPSeparator />
          <InputOTPGroup>
            <InputOTPSlot index={3} />
            <InputOTPSlot index={4} />
            <InputOTPSlot index={5} />
          </InputOTPGroup>
        </InputOTP>
        <p className="mt-2 text-xs text-[#737373]">当前值：{value || "（空）"}</p>
      </div>
    </div>
  );
}

function AspectRatioPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">16 / 9 比例容器</MetaMedium>
        <div className="w-full max-w-md">
          <AspectRatio ratio={16 / 9}>
            <div className="flex h-full w-full items-center justify-center rounded-[4px] bg-gradient-to-br from-[#1447E6] to-[#60A5FA] text-white">
              <span className="text-2xl font-semibold">16 : 9</span>
            </div>
          </AspectRatio>
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">1 / 1 比例容器</MetaMedium>
        <div className="w-40">
          <AspectRatio ratio={1}>
            <div className="flex h-full w-full items-center justify-center rounded-[4px] border border-[#DDE7F2] bg-[#F4F8FC] text-sm text-[#334155]">
              方形 1 : 1
            </div>
          </AspectRatio>
        </div>
      </div>
    </div>
  );
}

function NavigationMenuPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">水平导航 + 下拉面板</MetaMedium>
        <NavigationMenu>
          <NavigationMenuList>
            <NavigationMenuItem>
              <NavigationMenuTrigger>产品</NavigationMenuTrigger>
              <NavigationMenuContent>
                <ul className="grid w-[280px] gap-2 p-3">
                  <li>
                    <NavigationMenuLink className="block rounded-[4px] p-2 hover:bg-[#F4F8FC]">
                      <div className="text-sm font-medium">Agent 平台</div>
                      <div className="text-xs text-[#737373]">企业级 Agent 编排</div>
                    </NavigationMenuLink>
                  </li>
                  <li>
                    <NavigationMenuLink className="block rounded-[4px] p-2 hover:bg-[#F4F8FC]">
                      <div className="text-sm font-medium">模型管理</div>
                      <div className="text-xs text-[#737373]">多模型接入 / 配额</div>
                    </NavigationMenuLink>
                  </li>
                </ul>
              </NavigationMenuContent>
            </NavigationMenuItem>
            <NavigationMenuItem>
              <NavigationMenuTrigger>解决方案</NavigationMenuTrigger>
              <NavigationMenuContent>
                <ul className="grid w-[260px] gap-2 p-3">
                  <li>
                    <NavigationMenuLink className="block rounded-[4px] p-2 hover:bg-[#F4F8FC]">
                      <div className="text-sm font-medium">客服场景</div>
                    </NavigationMenuLink>
                  </li>
                  <li>
                    <NavigationMenuLink className="block rounded-[4px] p-2 hover:bg-[#F4F8FC]">
                      <div className="text-sm font-medium">研发提效</div>
                    </NavigationMenuLink>
                  </li>
                </ul>
              </NavigationMenuContent>
            </NavigationMenuItem>
            <NavigationMenuItem>
              <NavigationMenuLink className="px-3 py-2 text-sm">文档</NavigationMenuLink>
            </NavigationMenuItem>
          </NavigationMenuList>
        </NavigationMenu>
      </div>
    </div>
  );
}

function MenubarPreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">应用顶部菜单栏</MetaMedium>
        <Menubar>
          <MenubarMenu>
            <MenubarTrigger>文件</MenubarTrigger>
            <MenubarContent>
              <MenubarItem>
                新建 <MenubarShortcut>⌘N</MenubarShortcut>
              </MenubarItem>
              <MenubarItem>
                打开 <MenubarShortcut>⌘O</MenubarShortcut>
              </MenubarItem>
              <MenubarSeparator />
              <MenubarItem>
                保存 <MenubarShortcut>⌘S</MenubarShortcut>
              </MenubarItem>
            </MenubarContent>
          </MenubarMenu>
          <MenubarMenu>
            <MenubarTrigger>编辑</MenubarTrigger>
            <MenubarContent>
              <MenubarItem>
                撤销 <MenubarShortcut>⌘Z</MenubarShortcut>
              </MenubarItem>
              <MenubarItem>
                重做 <MenubarShortcut>⇧⌘Z</MenubarShortcut>
              </MenubarItem>
              <MenubarSeparator />
              <MenubarItem>查找…</MenubarItem>
            </MenubarContent>
          </MenubarMenu>
          <MenubarMenu>
            <MenubarTrigger>视图</MenubarTrigger>
            <MenubarContent>
              <MenubarItem>放大</MenubarItem>
              <MenubarItem>缩小</MenubarItem>
            </MenubarContent>
          </MenubarMenu>
        </Menubar>
      </div>
    </div>
  );
}

function ResizablePreviewSection() {
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">水平双栏（可拖拽）</MetaMedium>
        <ResizablePanelGroup
          direction="horizontal"
          className="h-[180px] max-w-2xl rounded-[4px] border border-[#DDE7F2]"
        >
          <ResizablePanel defaultSize={30}>
            <div className="flex h-full items-center justify-center bg-[#F4F8FC] text-sm text-[#334155]">
              侧栏 30%
            </div>
          </ResizablePanel>
          <ResizableHandle withHandle />
          <ResizablePanel defaultSize={70}>
            <div className="flex h-full items-center justify-center bg-white text-sm text-[#334155]">
              主区 70%
            </div>
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>
      <div>
        <MetaMedium className="mb-2 block">三栏 + 嵌套垂直分割</MetaMedium>
        <ResizablePanelGroup
          direction="horizontal"
          className="h-[200px] max-w-2xl rounded-[4px] border border-[#DDE7F2]"
        >
          <ResizablePanel defaultSize={25}>
            <div className="flex h-full items-center justify-center bg-[#F4F8FC] text-sm">导航</div>
          </ResizablePanel>
          <ResizableHandle />
          <ResizablePanel defaultSize={75}>
            <ResizablePanelGroup direction="vertical">
              <ResizablePanel defaultSize={60}>
                <div className="flex h-full items-center justify-center bg-white text-sm">主内容</div>
              </ResizablePanel>
              <ResizableHandle />
              <ResizablePanel defaultSize={40}>
                <div className="flex h-full items-center justify-center bg-[#F8FAFF] text-sm">日志</div>
              </ResizablePanel>
            </ResizablePanelGroup>
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>
    </div>
  );
}

function FilterTriggerPreviewSection() {
  const [open1, setOpen1] = useState(false);
  return (
    <div className="space-y-6">
      <div>
        <MetaMedium className="mb-2 block">variant=&quot;button&quot;（表单场景，仿 Select）</MetaMedium>
        <div className="grid max-w-2xl grid-cols-3 gap-3">
          <FilterTrigger variant="button" placeholder="请选择环境" />
          <FilterTrigger
            variant="button"
            label="生产环境"
            active
            open={open1}
            onClick={() => setOpen1((v) => !v)}
          />
          <FilterTrigger variant="button" placeholder="禁用" disabled />
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">variant=&quot;icon&quot;（表头列筛选）</MetaMedium>
        <div className="flex items-center gap-6">
          <FilterTrigger variant="icon" title="状态" />
          <FilterTrigger variant="icon" title="环境" active />
        </div>
      </div>
      <div>
        <MetaMedium className="mb-2 block">variant=&quot;badge-pencil&quot;（内联编辑）</MetaMedium>
        <div className="flex items-center gap-2">
          <Badge variant="outline">范围：仅当前用户</Badge>
          <FilterTrigger variant="badge-pencil" onClick={() => toast.info("点击编辑")} />
        </div>
      </div>
    </div>
  );
}

/* ───────────── 筛选面板组件套件（集中预览，对齐设计稿三张图） ─────────────
 * 按"数据结构同构性"分类：
 *   一、可合并组（同构 → 统一为同一组件的变体）：Select / TreeSelect / ScopeSelect
 *   二、独立组件（数据结构异构 → 不合并）：GroupSelect / TokenValueEditor / ActionsMenu
 *   三、底层骨架（组合积木，不参与合并）：SelectPanel / FilterTrigger
 */

/** 套件内单个组件标题（序号/字母 + 名称 + 类型徽标 + 一句话说明） */
function SuiteItemTitle({
  index,
  name,
  desc,
}: {
  index: string;
  name: string;
  desc?: string;
}) {
  return (
    <div className="mb-3">
      <div className="flex items-center gap-2">
        <span className="text-sm font-semibold text-[#0A0A0A]">
          {index}. {name}
        </span>
      </div>
      {desc && <p className="mt-1 text-xs text-[#94A3B8]">{desc}</p>}
    </div>
  );
}

/** 单个变体卡片：变体标签 + 说明 + 演示体 + API */
function VariantCell({
  tag,
  note,
  api,
  children,
}: {
  tag: string;
  note?: string;
  api: string;
  children: React.ReactNode;
}) {
  const [copied, setCopied] = useState(false);
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(api);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // 忽略复制失败
    }
  };
  return (
    <div className="rounded-[8px] bg-[#F8FAFC] border border-[#E2E8F0] p-4 space-y-3">
      <div className="flex items-center gap-2">
        <span className="rounded-[4px] bg-[#E2E8F0] px-1.5 py-0.5 text-xs font-medium text-[#475569]">
          {tag}
        </span>
        {note && <span className="text-xs text-[#94A3B8]">{note}</span>}
      </div>
      <div>{children}</div>
      <div className="flex items-center gap-1.5 text-xs text-[#94A3B8]">
        <span>组件引用名：</span>
        <span className="font-mono text-[#475569]">{api}</span>
        <button
          type="button"
          onClick={handleCopy}
          aria-label={copied ? "已复制" : "复制组件引用名"}
          title={copied ? "已复制" : "复制"}
          className="inline-flex h-5 w-5 items-center justify-center rounded-[4px] text-[#94A3B8] transition-colors hover:bg-white hover:text-[#475569]"
        >
          {copied ? <Check className="h-3.5 w-3.5 text-[#16A34A]" /> : <Copy className="h-3.5 w-3.5" />}
        </button>
      </div>
    </div>
  );
}

// 套件演示用 mock 数据
const SUITE_VPC_OPTIONS = [
  { value: "vpc-default", label: "vpc-default（默认）" },
  { value: "vpc-production", label: "vpc-production" },
  { value: "vpc-staging", label: "vpc-staging" },
  { value: "vpc-dev-team-alpha", label: "vpc-dev-team-alpha" },
  { value: "vpc-isolated", label: "vpc-isolated" },
];
const SUITE_STATUS_OPTIONS = [
  { value: "running", label: "运行中" },
  { value: "stopped", label: "已停止" },
  { value: "creating", label: "创建中" },
  { value: "error", label: "异常" },
];
const SUITE_INSTANT_SECTIONS = [
  { label: "生产环境", options: [{ value: "p-run", label: "运行中" }, { value: "p-stop", label: "已停止" }] },
  { label: "测试环境", options: [{ value: "t-run", label: "运行中" }, { value: "t-creating", label: "创建中" }] },
];
const SUITE_SCOPE_GROUPS = [
  { id: "rd-center", name: "研发中心" },
  { id: "frontend", name: "前端组", parentId: "rd-center" },
  { id: "backend", name: "后端组", parentId: "rd-center" },
  { id: "qa", name: "测试组", parentId: "rd-center" },
  { id: "design", name: "设计组" },
];
const SUITE_USER_GROUPS: UserGroup[] = [
  { id: "dept-1", name: "研发中心", parentId: null, source: "oneid-dept", readonly: true, createdAt: "2026-01-01" },
  { id: "dept-1-1", name: "前端组", parentId: "dept-1", source: "oneid-dept", readonly: true, createdAt: "2026-01-01" },
  { id: "dept-1-2", name: "后端组", parentId: "dept-1", source: "oneid-dept", readonly: true, createdAt: "2026-01-01" },
  { id: "manual-1", name: "项目 A 小组", parentId: null, source: "manual", readonly: false, createdAt: "2026-01-01" },
];

function FilterPanelSuitePreviewSection() {
  // 可合并组状态
  const [searchableVal, setSearchableVal] = useState("vpc-production");
  const [filterMulti, setFilterMulti] = useState<Set<string>>(new Set(["running"]));
  const [instantMultiFlat, setInstantMultiFlat] = useState<Set<string>>(new Set(["vpc-production"]));
  const [instantMultiGroup, setInstantMultiGroup] = useState<Set<string>>(new Set(["p-run"]));
  const [treeVal, setTreeVal] = useState("");
  const [scopeWithTrigger, setScopeWithTrigger] = useState<Set<string>>(new Set());
  const [scopeConfirm, setScopeConfirm] = useState<string[]>([]);
  const [scopeConfirmType, setScopeConfirmType] = useState<"all" | "groups">("all");
  // 独立组件状态
  const [groupSel, setGroupSel] = useState<string[]>([]);
  const [tokenMode, setTokenMode] = useState<"custom" | "unlimited">("unlimited");
  const [tokenVal, setTokenVal] = useState("");

  const scopeInstantGroups = [
    { id: "all-users", name: "全部用户" },
    ...SUITE_SCOPE_GROUPS,
  ];

  return (
    <div className="space-y-2">

      {/* 1. Select */}
      <div className="mt-6">
        <SuiteItemTitle index="1" name="Select" />
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <VariantCell tag="单选 · 即时 · 扁平（基础）" api="<Select />">
            <Select defaultValue="admin">
              <SelectTrigger className="w-full"><SelectValue placeholder="请选择" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="global">Global 全局</SelectItem>
                <SelectItem value="tenant">Tenant 用户端</SelectItem>
                <SelectItem value="admin">Admin 管控端</SelectItem>
              </SelectContent>
            </Select>
          </VariantCell>
          <VariantCell tag="单选 · 即时 · 扁平（搜索）" api="<SearchableSelect />">
            <SearchableSelect
              options={SUITE_VPC_OPTIONS}
              value={searchableVal}
              onChange={setSearchableVal}
              placeholder="选择 VPC"
              searchPlaceholder="搜索 VPC"
            />
          </VariantCell>
          <VariantCell tag="多选 · 确认 · 扁平" api="<FilterMultiSelect />">
            <FilterMultiSelect
              title="状态"
              options={SUITE_STATUS_OPTIONS.map((o) => ({ value: o.value, label: o.label }))}
              value={filterMulti}
              onChange={setFilterMulti}
            />
          </VariantCell>
          <VariantCell tag="多选 · 即时 · 扁平" api="<InstantMultiSelect options={[]} />">
            <InstantMultiSelect
              options={SUITE_VPC_OPTIONS}
              value={instantMultiFlat}
              onChange={setInstantMultiFlat}
              placeholder="vpc-production"
            />
          </VariantCell>
          <VariantCell tag="多选 · 即时 · 分组" api="<InstantMultiSelect sections={[]} />">
            <InstantMultiSelect
              sections={SUITE_INSTANT_SECTIONS}
              value={instantMultiGroup}
              onChange={setInstantMultiGroup}
              placeholder="生产环境、运行中"
            />
          </VariantCell>
        </div>
      </div>

      {/* 2. TreeSelect */}
      <div className="mt-8">
        <SuiteItemTitle index="2" name="TreeSelect" />
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <VariantCell tag="单选 · 确认 · 树（页面）" api='<TreeSelect triggerVariant="button" />'>
            <TreeSelect
              nodes={[
                { id: "root", name: "全部部门", children: [
                  { id: "fe", name: "前端组" },
                  { id: "be", name: "后端组", children: [{ id: "be-1", name: "基础架构" }] },
                ] },
              ]}
              value={treeVal}
              onChange={setTreeVal}
            />
          </VariantCell>
          <VariantCell tag="单选 · 确认 · 树（表头）" api='<TreeSelect triggerVariant="filter-icon" title="" />'>
            <div className="flex h-9 items-center">
              <TreeSelect
                triggerVariant="filter-icon"
                title="部门"
                nodes={[
                  { id: "root", name: "全部部门", children: [
                    { id: "fe", name: "前端组" },
                    { id: "be", name: "后端组", children: [{ id: "be-1", name: "基础架构" }] },
                  ] },
                ]}
                value={treeVal}
                onChange={setTreeVal}
              />
            </div>
          </VariantCell>
        </div>
      </div>

      {/* 3. ScopeSelect */}
      <div className="mt-8">
        <SuiteItemTitle index="3" name="ScopeSelect" />
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <VariantCell tag="多选 · 即时 · 扁平（含触发器）" api='<ScopeSelect mode="instant" withTrigger />'>
            <ScopeSelect
              mode="instant"
              withTrigger
              groups={scopeInstantGroups}
              value={scopeWithTrigger}
              onChange={setScopeWithTrigger}
              publicKey="all-users"
              triggerPlaceholder="选择分组"
            />
          </VariantCell>
          <VariantCell tag="多选 · 确认 · 树" api='<ScopeSelect mode="confirm" scope="all" />'>
            <ScopeSelect
              mode="confirm"
              scope={scopeConfirmType}
              selectedGroupIds={scopeConfirm}
              groups={SUITE_SCOPE_GROUPS}
              onConfirm={(scope, ids) => { setScopeConfirmType(scope); setScopeConfirm(ids); }}
            />
          </VariantCell>
        </div>
      </div>

      {/* 4. GroupSelect & TokenValueEditor */}
      <div className="mt-6">
        <SuiteItemTitle index="4" name="GroupSelect / TokenValueEditor" />
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <VariantCell tag="多选 · 即时 · 树" api="<GroupSelect />">
            <GroupSelect
              groups={SUITE_USER_GROUPS}
              selectedIds={groupSel}
              onChange={setGroupSel}
              placeholder="选择分组"
            />
          </VariantCell>
          <VariantCell tag="单选 · 确认 · 扁平" api="<TokenValueEditor />">
            <TokenValueEditor
              mode={tokenMode}
              valStr={tokenVal}
              onCommit={(m, v) => {
                setTokenMode(m);
                setTokenVal(v);
              }}
            />
          </VariantCell>
        </div>
      </div>






    </div>
  );
}

/**
 * TreeSelect 专属预览：仅展示树形单选下拉的两个触发变体，与 FilterPanelSuite 全景套件区分开，做到名实相符。
 * 两个变体使用各自独立 state，互不串选。
 */
function TreeSelectPreviewSection() {
  const [treeValPage, setTreeValPage] = useState("");
  const [treeValHeader, setTreeValHeader] = useState("");
  const TREE_NODES = useMemo(() => [
    { id: "root", name: "全部部门", children: [
      { id: "fe", name: "前端组" },
      { id: "be", name: "后端组", children: [{ id: "be-1", name: "基础架构" }] },
    ] },
  ], []);

  return (
    <div className="space-y-2">
      <div className="mt-6">
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <VariantCell tag="单选 · 确认 · 树（页面）" api='<TreeSelect triggerVariant="button" />'>
            <TreeSelect
              nodes={TREE_NODES}
              value={treeValPage}
              onChange={setTreeValPage}
            />
          </VariantCell>
          <VariantCell tag="单选 · 确认 · 树（表头）" api='<TreeSelect triggerVariant="filter-icon" title="" />'>
            <div className="flex h-9 items-center">
              <TreeSelect
                triggerVariant="filter-icon"
                title="部门"
                nodes={TREE_NODES}
                value={treeValHeader}
                onChange={setTreeValHeader}
              />
            </div>
          </VariantCell>
        </div>
      </div>
    </div>
  );
}

/* ========== END 新增 Preview 组件 ========== */

function renderPreview(id: ComponentId) {
  if (id === "color") return <ColorPreview />;
  if (id === "typography") return <TypographyPreview />;
  if (["surface-card", "surface-inner", "surface-config", "surface-overlay", "tenant-card"].includes(id)) return <SurfacePreview id={id} />;
  if (id === "number-card") return <NumberCardPreview />;
  if (id === "dark-veil") return <DarkVeilPreview />;
  if (id === "button") return <ButtonPreview />;
  if (["input", "input-group", "textarea", "date-picker", "date-time-picker", "checkbox", "field", "radio-group", "radio-card", "switch", "filter-chip"].includes(id)) return <FormPreview id={id} />;
  if (id === "alert") return <AlertPreview />;
  if (id === "dialog") return <DialogPreview />;
  if (id === "alert-dialog") return <AlertDialogPreview />;
  if (id === "drawer") return <DrawerPreview />;
  if (["tooltip", "popover", "info-popover", "progress", "spinner"].includes(id)) return <FloatingPreview id={id} />;
  if (id === "table") return <TablePreview />;
  if (id === "pagination") return <PaginationPreview />;
  if (id === "kbd") return <KbdPreview />;
  if (id === "stepper") return <StepperPreview />;
  if (["segment", "segmented-tabs", "tabs"].includes(id)) return <SegmentTabsPreview id={id} />;
  if (["badge", "status-tag", "empty"].includes(id)) return <StatusPreview id={id} />;
  if (id === "tenant-section") return <TenantSectionPreview />;
  if (id === "topnav") return <TopNavPreview />;
  if (id === "admin-page-header") return <AdminPageHeaderPreview />;
  if (id === "admin-sidebar") return <AdminSidebarPreview />;
  if (id === "toast") return <ToastPreviewSection />;
  if (id === "avatar") return <AvatarPreviewSection />;
  if (id === "tree") return <TreePreviewSection />;
  if (id === "breadcrumb") return <BreadcrumbPreviewSection />;
  if (id === "transfer") return <TransferPreviewSection />;
  if (id === "search-filter-bar") return <SearchFilterBarPreviewSection />;
  if (id === "batch-actions-bar") return <BatchActionsBarPreviewSection />;
  if (id === "chart-stat") return <ChartStatPreviewSection />;
  if (id === "upload") return <UploadPreviewSection />;
  if (id === "tag") return <TagPreviewSection />;
  if (id === "accordion") return <AccordionPreviewSection />;
  if (id === "card") return <CardPreviewSection />;
  if (id === "dropdown-menu") return <DropdownMenuPreviewSection />;
  if (id === "line-tabs") return <LineTabsPreviewSection />;
  if (id === "sheet") return <SheetPreviewSection />;
  if (id === "skeleton") return <SkeletonPreviewSection />;
  if (id === "slider") return <SliderPreviewSection />;
  if (id === "separator") return <SeparatorPreviewSection />;
  if (id === "scroll-area") return <ScrollAreaPreviewSection />;
  if (id === "collapsible") return <CollapsiblePreviewSection />;
  if (id === "toggle-group") return <ToggleGroupPreviewSection />;
  if (id === "hover-card") return <HoverCardPreviewSection />;
  if (id === "context-menu") return <ContextMenuPreviewSection />;
  if (id === "all-users-tag") return <AllUsersTagPreviewSection />;
  if (id === "back-button") return <BackButtonPreviewSection />;
  if (id === "favorite-button") return <FavoriteButtonPreviewSection />;
  if (id === "more-actions-dropdown") return <MoreActionsDropdownPreviewSection />;
  if (id === "tree-select") return <TreeSelectPreviewSection />;
  if (id === "file-browser") return <FileBrowserPreviewSection />;
  if (id === "carousel") return <CarouselPreviewSection />;
  if (id === "form") return <FormPreviewSection />;
  if (id === "calendar") return <CalendarPreviewSection />;
  if (id === "input-otp") return <InputOTPPreviewSection />;
  if (id === "aspect-ratio") return <AspectRatioPreviewSection />;
  if (id === "navigation-menu") return <NavigationMenuPreviewSection />;
  if (id === "menubar") return <MenubarPreviewSection />;
  if (id === "resizable") return <ResizablePreviewSection />;
  if (id === "filter-trigger") return <FilterTriggerPreviewSection />;
  if (id === "filter-panel-suite") return <FilterPanelSuitePreviewSection />;
  return null;
}

/**
 * 典型页面样例区块：以缩略图卡片 + 关键组件 chips 的形式展示 7 类 page-references 标杆样本。
 * 数据来自 PAGE_REFERENCES（同源 .codebuddy/skills/clawpro-portable-design-skill/assets/page-references/*.md）。
 */
function PageReferencesSection({ keyword }: { keyword: string }) {
  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    if (!kw) return PAGE_REFERENCES;
    return PAGE_REFERENCES.filter((page) =>
      [page.name, page.cnName, page.category, page.route, page.description, page.whyTypical, ...page.keyComponents]
        .some((text) => text.toLowerCase().includes(kw))
    );
  }, [keyword]);

  if (filtered.length === 0) {
    return (
      <SurfaceCard className="rounded-[4px] border-[#DDE7F2] bg-white p-12">
        <div className="flex flex-col items-center justify-center gap-2 text-center">
          <BodyMedium tone="emphasis">没有匹配的典型页面样例</BodyMedium>
          <BodyText>试试搜索：配置 / 列表 / 看板 / 空页面 / 服务开通</BodyText>
        </div>
      </SurfaceCard>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <BodyText>
          展示 7 类典型页面（配置 / 基础列表 / 富功能列表 / 空页面 / 复杂列表 / 数据看板 / 服务开通），
          作为生成新页面时的标杆样本。结构骨架优先复用对应 spec md。
        </BodyText>
        <MetaText className="shrink-0">
          数据源：<CodeText>.codebuddy/skills/clawpro-portable-design-skill/assets/page-references/</CodeText>
        </MetaText>
      </div>
      <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
        {filtered.map((page) => (
          <SurfaceCard key={page.id} className="overflow-hidden rounded-[4px] border-[#DDE7F2] bg-white transition-shadow hover:shadow-md">
            <div className="grid grid-cols-1 md:grid-cols-[280px_minmax(0,1fr)]">
              <a
                href={page.route}
                target="_blank"
                rel="noreferrer"
                className="block aspect-[16/10] overflow-hidden border-b border-[#DDE7F2] bg-[#F7FAFF] md:aspect-auto md:border-b-0 md:border-r"
                aria-label={`打开 ${page.cnName}`}
              >
                <img
                  src={page.screenshot}
                  alt={`${page.cnName} 页面截图`}
                  loading="lazy"
                  className="h-full w-full object-cover object-top transition-transform hover:scale-[1.02]"
                />
              </a>
              <div className="flex min-w-0 flex-col gap-3 p-5">
                <div className="flex items-center gap-2">
                  <span className="inline-flex h-5 items-center rounded-[4px] border border-[#DDE7F2] bg-[#F7FAFF] px-2 text-xs font-medium text-[#1447E6]">
                    {page.category}
                  </span>
                  <CodeText>{page.route}</CodeText>
                </div>
                <div>
                  <div className="flex items-baseline gap-2">
                    <BodyMedium tone="emphasis" className="text-base">{page.cnName}</BodyMedium>
                    <MetaText>{page.name}</MetaText>
                  </div>
                  <BodyText className="mt-1 line-clamp-2">{page.description}</BodyText>
                </div>
                <div className="flex flex-wrap gap-1">
                  {page.keyComponents.slice(0, 7).map((kc) => (
                    <span key={kc} className="inline-flex h-5 items-center rounded-[3px] border border-[#DDE7F2] bg-white px-1.5 text-[11px] text-[#334155]">
                      {kc}
                    </span>
                  ))}
                </div>
                <details className="group">
                  <summary className="flex w-max cursor-pointer list-none items-center gap-1 whitespace-nowrap text-xs text-[#334155] transition-colors hover:text-[#0A0A0A] [&::-webkit-details-marker]:hidden">
                    <span className="transition-transform group-open:rotate-90">›</span>
                    为何典型 / 源码 / spec
                  </summary>
                  <div className="mt-2 grid grid-cols-[64px_minmax(0,1fr)] gap-x-3 gap-y-1.5 border-t border-[#DDE7F2] pt-2">
                    <MetaText>定位</MetaText><MetaText tone="secondary">{page.whyTypical}</MetaText>
                    <MetaText>源码</MetaText><CodeText>{page.source}</CodeText>
                    <MetaText>规范</MetaText><CodeText>{page.spec}</CodeText>
                  </div>
                </details>
                <div className="mt-auto flex items-center gap-3 pt-1">
                  <a
                    href={page.route}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex h-8 items-center gap-1 rounded-[4px] bg-[#020617] px-3 text-xs font-medium text-white transition-colors hover:bg-[#1447E6]"
                  >
                    打开页面 <ArrowRight className="size-3" />
                  </a>
                  <a
                    href={page.screenshot}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex h-8 items-center gap-1 rounded-[4px] border border-[#DDE7F2] bg-white px-3 text-xs font-medium text-[#334155] transition-colors hover:border-[#9FB6D8]"
                  >
                    查看大图
                  </a>
                </div>
              </div>
            </div>
          </SurfaceCard>
        ))}
      </div>
    </div>
  );
}

export default function DesignSystemComponents() {
  // URL 直链：?id=xxx 命中已注册组件 id 时初始选中该组件（无效则回落 color）
  const [selectedId, setSelectedId] = useState<ComponentId>(() => {
    if (typeof window === "undefined") return "color";
    const idParam = new URLSearchParams(window.location.search).get("id");
    const hit = idParam && COMPONENTS.find((c) => c.id === idParam);
    return hit ? (idParam as ComponentId) : "color";
  });
  const [keyword, setKeyword] = useState("");
  const [platformFilter, setPlatformFilter] = useState<"全部组件" | Platform | "典型页面">("全部组件");

  const isAdminPlatform = platformFilter === "Admin 管控端";
  const isPagesTab = platformFilter === "典型页面";

  const filteredComponents = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    return COMPONENTS.filter((item) => {
      if (item.hidden) return false; // 软隐藏组件不参与 Tab/组织/搜索
      const matchesKeyword = !kw || [item.name, item.cnName, item.description, item.source, item.platform, item.adoption, item.applicationScope, ...item.tags].some((text) => text.toLowerCase().includes(kw));
      // Admin tab 由 ADMIN_SPEC_GROUPS 单独驱动，这里只处理 全部 / Global / Tenant
      const matchesPlatform = platformFilter === "全部组件" || (platformFilter !== "Admin 管控端" && item.platform === platformFilter);
      return matchesKeyword && matchesPlatform;
    });
  }, [platformFilter, keyword]);

  // selected 仍走全集：URL 直链 ?id=surface-overlay 可访问详情页；fallback 落到首个可见项
  const selected = COMPONENTS.find((item) => item.id === selectedId) ?? COMPONENTS.find((item) => !item.hidden) ?? COMPONENTS[0];
  const applicationPages = useMemo(() => getApplicationPages(selected), [selected]);
  const grouped = useMemo(() => {
    return (Object.keys(GROUP_LABELS) as GroupKey[]).map((group) => ({
      group,
      label: GROUP_LABELS[group],
      items: filteredComponents.filter((item) => item.group === group),
    })).filter((group) => group.items.length > 0);
  }, [filteredComponents]);

  /** Admin 管控端 tab 专属：按 36 个 spec 组织渲染，并按关键字过滤组件 */
  const groupedAdmin = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    return ADMIN_SPEC_GROUPS.map((spec) => {
      const items = spec.components
        .map((cid) => COMPONENTS.find((c) => c.id === cid))
        .filter((c): c is ComponentMeta => Boolean(c))
        .filter((c) =>
          !kw ||
          [c.name, c.cnName, c.description, c.source, c.applicationScope, ...c.tags, spec.name, spec.cnName, spec.id]
            .some((t) => t.toLowerCase().includes(kw))
        );
      return { spec, items };
    }).filter((g) => g.items.length > 0);
  }, [keyword]);

  const platformCount = (platform: Platform) =>
    platform === "Admin 管控端"
      ? ADMIN_SPEC_GROUPS.length
      : COMPONENTS.filter((item) => !item.hidden && item.platform === platform).length;

  return (
    <div className="min-h-screen bg-[#F4F8FC] text-[#0A0A0A]">
      <header className="relative overflow-hidden border-b border-[#DDE7F2] bg-[linear-gradient(180deg,#FFFFFF_0%,#F7FAFF_100%)]">
        <div className="pointer-events-none absolute right-[-140px] top-[-180px] h-[380px] w-[380px] rounded-full bg-[#1447E6]/10 blur-3xl" />
        <div className="pointer-events-none absolute left-[20%] top-[-220px] h-[320px] w-[320px] rounded-full bg-[#60A5FA]/8 blur-3xl" />
        <div className="relative mx-auto max-w-[1680px] px-8 py-7">
          <div className="flex items-start justify-between gap-8">
            <div className="min-w-0">
              <div className="mb-3 flex flex-wrap items-center gap-2">
                <span className="inline-flex h-6 items-center rounded-[4px] border border-[#DDE7F2] bg-white/75 px-2.5 text-xs font-medium text-[#334155]">
                  内部设计资产
                </span>
                <span className="inline-flex h-6 items-center rounded-[4px] border border-[#DDE7F2] bg-white/75 px-2.5 text-xs font-medium text-[#334155]">
                  组件规范维护：miekoyychen / addietang
                </span>
                <span className="inline-flex h-6 items-center rounded-[4px] border border-[#DDE7F2] bg-white/75 px-2.5 text-xs font-medium text-[#334155]">
                  数据来源: .codebuddy/skills/clawpro-portable-design-skill
                </span>
              </div>
              <div className="flex items-center gap-2">
                <TenantPageTitle>ClawPro 全局组件展示台</TenantPageTitle>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button type="button" className="flex size-6 items-center justify-center rounded-full text-[#737373] transition-colors hover:bg-white hover:text-[#1447E6]" aria-label="说明">
                        <Info className="size-4" />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent className="max-w-[360px]">
                      当前展示台优先接入设计团队高频参考组件；底层能力组件会按页面修复和设计规范沉淀节奏持续补充。
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>
              <BodyText className="mt-2 max-w-3xl">展示 ClawPro 已沉淀的全局组件资产，包括真实样式、交互状态、使用指引和页面效果校准参考。</BodyText>
            </div>
            <div className="grid w-[360px] shrink-0 grid-cols-2 gap-3">
              <StatCard label="已沉淀规范组件" value={DOCUMENTED_COMPONENT_COUNT} hint="项" />
              <StatCard label="已接入预览组件" value={`${COMPONENTS.filter((c) => !c.hidden).length}`} hint="个" />
            </div>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-[1680px] px-8 py-8">
        <div className="mb-5">
          <SectionTitle>{isPagesTab ? "典型页面样例" : "组件分类展示"}</SectionTitle>
          <BodyText className="mt-1">
            {isPagesTab
              ? "按页面类别浏览 7 类典型页面（配置 / 列表 / 看板 / 空页面 / 服务开通），生成新页面时优先复用这些标杆样本的结构骨架。"
              : "按分类浏览单个组件，查看真实样式、所有状态、交互示例、使用指引与页面效果校准参考。"}
          </BodyText>
          <div className="mt-5 flex flex-wrap items-center justify-between gap-3">
            <div className="flex flex-wrap items-center gap-2">
              {(["全部组件", "Global 全局", "Tenant 用户端", "Admin 管控端", "典型页面"] as const).map((item) => {
                const active = platformFilter === item;
                const count =
                  item === "全部组件"
                    ? COMPONENTS.filter((c) => !c.hidden).length
                    : item === "典型页面"
                      ? PAGE_REFERENCES.length
                      : platformCount(item);
                return (
                  <button
                    key={item}
                    type="button"
                    onClick={() => setPlatformFilter(item)}
                    className={`h-8 rounded-[4px] border px-4 text-sm transition-colors ${active ? "border-[#020617] bg-[#020617] text-white" : "border-[#DDE7F2] bg-white text-[#020617] hover:border-[#9FB6D8]"}`}
                  >
                    {item} <span className={active ? "text-white/70" : "text-[#737373]"}>{count}</span>
                  </button>
                );
              })}
            </div>
            <div className="relative w-[360px] max-w-full">
              <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[#A3A3A3]" />
              <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} className="border-[#DDE7F2] pl-9 hover:border-[#9FB6D8] focus:border-[#1447E6]" placeholder={isPagesTab ? "搜索典型页面" : "搜索组件"} />
            </div>
          </div>
        </div>

        {isPagesTab ? (
          <PageReferencesSection keyword={keyword} />
        ) : (
        <SurfaceCard className="overflow-hidden rounded-[4px] border-[#DDE7F2] bg-white">
          <div className="grid grid-cols-[300px_minmax(0,1fr)] items-stretch bg-white">
            <aside className="self-stretch border-r border-[#DDE7F2] bg-[#F7FAFF]">
              <div className="sticky top-4 max-h-[calc(100vh-32px)] overflow-y-auto p-4 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
                <div className="space-y-5">
                  {isAdminPlatform
                    ? groupedAdmin.map(({ spec, items }) => (
                        <div key={spec.id}>
                          <div className="mb-2 flex items-center justify-between px-2">
                            <MetaText className="uppercase tracking-[0.08em]">{spec.cnName}</MetaText>
                            <MetaText>{items.length} 个</MetaText>
                          </div>
                          <div className="space-y-0.5">
                            {items.map((item) => {
                              const active = item.id === selectedId;
                              return (
                                <button
                                  key={`${spec.id}__${item.id}`}
                                  type="button"
                                  onClick={() => setSelectedId(item.id)}
                                  className={`relative w-full rounded-[4px] px-3 py-2 text-left transition-colors ${active ? "bg-white text-[#0A0A0A]" : "text-[#0A0A0A] hover:bg-white/70"}`}
                                >
                                  {active && <span className="absolute left-0 top-2 bottom-2 w-px bg-[#0A0A0A]" />}
                                  <BodyMedium className="block truncate pl-1" tone="emphasis">{item.name}</BodyMedium>
                                  <MetaText className="mt-1 block truncate pl-1">{item.cnName} · 约 {USAGE_DATA[item.id]?.moduleCount ?? item.moduleCount} 个页面/模块</MetaText>
                                </button>
                              );
                            })}
                          </div>
                        </div>
                      ))
                    : grouped.map((group) => (
                    <div key={group.group}>
                      <div className="mb-2 flex items-center justify-between px-2">
                        <MetaText className="uppercase tracking-[0.08em]">{group.label}</MetaText>
                        <MetaText>{group.items.length} 个</MetaText>
                      </div>
                      <div className="space-y-0.5">
                        {group.items.map((item) => {
                          const active = item.id === selectedId;
                          return (
                            <button
                              key={item.id}
                              type="button"
                              onClick={() => setSelectedId(item.id)}
                            className={`relative w-full rounded-[4px] px-3 py-2 text-left transition-colors ${active ? "bg-white text-[#0A0A0A]" : "text-[#0A0A0A] hover:bg-white/70"}`}
                          >
                            {active && <span className="absolute left-0 top-2 bottom-2 w-px bg-[#0A0A0A]" />}
                            <BodyMedium className="block truncate pl-1" tone="emphasis">{item.name}</BodyMedium>
                            <MetaText className="mt-1 block truncate pl-1">{item.cnName} · 约 {USAGE_DATA[item.id]?.moduleCount ?? item.moduleCount} 个页面/模块</MetaText>
                            </button>
                          );
                        })}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </aside>

            <section className="min-w-0 space-y-7 bg-white p-6">
              <div className="pb-1">
                <div className="flex items-start justify-between gap-6">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-baseline gap-3">
                      <TenantPageTitle>{selected.name}</TenantPageTitle>
                      <BodyText>{selected.cnName}</BodyText>
                    </div>
                    <BodyText className="mt-2 max-w-3xl">{getComponentIntro(selected)}</BodyText>
                    <details className="group mt-2 inline-block">
                      <summary className="flex w-max cursor-pointer list-none items-center gap-1 whitespace-nowrap text-sm font-normal text-[#334155] transition-colors hover:text-[#0A0A0A] [&::-webkit-details-marker]:hidden">
                        <span className="transition-transform group-open:rotate-90">›</span>
                        更多信息
                      </summary>
                      <div className="mt-2 grid grid-cols-[88px_minmax(0,1fr)] gap-x-3 gap-y-1.5 border-t border-[#DDE7F2] pt-2">
                        <MetaText>维护人</MetaText><MetaText tone="secondary">{selected.maintainer ?? selected.owner}</MetaText>
                        <MetaText>源码路径</MetaText><CodeText>{selected.source}</CodeText>
                        <MetaText>规范来源</MetaText><CodeText>{selected.doc}</CodeText>
                      </div>
                    </details>
                  </div>
                  <div className="w-[280px] shrink-0 border-l border-[#DDE7F2] pl-6">
                    <div className="grid grid-cols-2 gap-6">
                      <div><MetaText>应用范围</MetaText><div className="mt-2"><StatNumber>{USAGE_DATA[selected.id]?.moduleCount ?? selected.moduleCount}</StatNumber><MetaText className="ml-1">页面/模块</MetaText></div></div>
                      <div><MetaText>组件实例</MetaText><div className="mt-2"><StatNumber>{USAGE_DATA[selected.id]?.instanceCount ?? selected.instanceCount}</StatNumber><MetaText className="ml-1">处</MetaText></div></div>
                    </div>
                    <Popover>
                      <PopoverTrigger asChild>
                        <button type="button" className="mt-3 text-sm font-medium text-[#1447E6] transition-colors hover:text-[#0A226F] hover:underline">
                          查看应用页面（{applicationPages.length}）
                        </button>
                      </PopoverTrigger>
                      <PopoverContent align="end" className="w-[540px] p-0">
                        <div className="border-b border-[#DDE7F2] px-4 py-3">
                          <BodyMedium>应用页面</BodyMedium>
                          <MetaText className="mt-1 block">按参考优先级排序，点击行可打开页面查看实际效果。</MetaText>
                        </div>
                        <div className="divide-y divide-[#DDE7F2]">
                          {applicationPages.map((page, index) => (
                            <a key={`${page.path}-${page.name}`} href={page.path} target="_blank" rel="noreferrer" className="grid grid-cols-[28px_120px_100px_minmax(0,1fr)_88px] items-center gap-3 px-4 py-3 transition-colors hover:bg-[#F8FAFF]">
                              <MetaText>{index + 1}</MetaText>
                              <BodyMedium className="truncate">{page.name}</BodyMedium>
                              <MetaText className="truncate">{page.platform}</MetaText>
                              <MetaText className="truncate" tone="secondary">{page.usage}</MetaText>
                              <span className="inline-flex items-center justify-end gap-1 text-xs font-medium text-[#1447E6]">打开页面<ArrowRight className="size-3" /></span>
                            </a>
                          ))}
                        </div>
                      </PopoverContent>
                    </Popover>
                  </div>
                </div>
              </div>

              <DetailSection title="真实组件预览与全状态展示">
                {renderPreview(selected.id)}
              </DetailSection>

              <DetailSection title="使用指引">
                <div className="grid grid-cols-3 gap-8">
                  <GuidanceBlock title="推荐使用场景" items={selected.usage} variant="usage" />
                  <GuidanceBlock title="注意事项" items={selected.notes} variant="notice" />
                  <GuidanceBlock title="页面效果校准 / 迁移建议" items={selected.migration} variant="migration" />
                </div>
              </DetailSection>
            </section>
          </div>
        </SurfaceCard>
        )}
      </main>
    </div>
  );
}
