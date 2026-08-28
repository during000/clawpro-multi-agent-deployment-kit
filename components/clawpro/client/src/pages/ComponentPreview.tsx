/**
 * ComponentPreview
 * 组件状态预览页 —— 用于索引文档 iframe 内嵌展示
 * 路由：/component-preview/:name
 */
import { Fragment, useState, type ReactElement } from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Progress } from "@/components/ui/progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableActionCell,
  TableExpandedRow,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Pagination } from "@/components/ui/pagination";
import { SurfaceCard } from "@/components/ui/Surface";
import { StatusTag } from "@/components/ui/status-tag";
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription } from "@/components/ui/empty";
import { useRoute } from "wouter";
import { ChevronDown, Info, MoreHorizontal } from "lucide-react";

function InputPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">Input 输入框</h2>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Normal（未输入）</label>
        <Input placeholder="请输入企业邮箱" />
      </div>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Filled（已输入）</label>
        <Input defaultValue="zhangsan@tencent.com" />
      </div>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Disabled（未输入）</label>
        <Input placeholder="请输入企业邮箱" disabled />
      </div>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Disabled（已输入）</label>
        <Input defaultValue="zhangsan@tencent.com" disabled />
      </div>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Error（报错）</label>
        <div className="space-y-1">
          <Input aria-invalid defaultValue="zhangsan" />
          <p className="text-xs text-[#d42a1e] leading-5">请输入正确企业邮箱</p>
        </div>
      </div>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Error（未输入）</label>
        <div className="space-y-1">
          <Input aria-invalid placeholder="请输入企业邮箱" />
          <p className="text-xs text-[#d42a1e] leading-5">请输入企业邮箱</p>
        </div>
      </div>
    </div>
  );
}

function SelectPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">Select 下拉选择</h2>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Normal（未选择）</label>
        <Select>
          <SelectTrigger className="w-[304px]">
            <SelectValue placeholder="请选择所属行业" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="gov">政务</SelectItem>
            <SelectItem value="internet">互联网</SelectItem>
            <SelectItem value="finance">金融</SelectItem>
            <SelectItem value="manufacture">制造</SelectItem>
            <SelectItem value="retail">零售</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Normal（已选择）</label>
        <Select defaultValue="gov">
          <SelectTrigger className="w-[304px]">
            <SelectValue placeholder="请选择所属行业" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="gov">政务</SelectItem>
            <SelectItem value="internet">互联网</SelectItem>
            <SelectItem value="finance">金融</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Disabled（未选择）</label>
        <Select disabled>
          <SelectTrigger className="w-[304px]">
            <SelectValue placeholder="请选择所属行业" />
          </SelectTrigger>
          <SelectContent><SelectItem value="gov">政务</SelectItem></SelectContent>
        </Select>
      </div>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Disabled（已选择）</label>
        <Select defaultValue="gov" disabled>
          <SelectTrigger className="w-[304px]">
            <SelectValue placeholder="请选择所属行业" />
          </SelectTrigger>
          <SelectContent><SelectItem value="gov">政务</SelectItem></SelectContent>
        </Select>
      </div>
    </div>
  );
}

function ButtonPreview() {
  return (
    <div className="space-y-8">
      <h2 className="text-base font-semibold text-gray-900">Button 按钮</h2>

      {/* 状态标题行 */}
      <div className="grid grid-cols-4 gap-4 text-xs text-gray-400 text-center">
        <span>normal</span><span>hover</span><span>click</span><span>disabled</span>
      </div>

      {/* 主要按钮 */}
      <div className="space-y-2">
        <label className="text-sm text-gray-500">主要按钮（claw-primary）</label>
        <div className="grid grid-cols-4 gap-4 items-center">
          <Button variant="claw-primary" size="claw">创建 Agent</Button>
          <Button variant="claw-primary" size="claw" className="bg-[linear-gradient(90deg,#020617_70%,#0A226F_110%)]">创建 Agent</Button>
          <Button variant="claw-primary" size="claw" className="bg-[linear-gradient(90deg,rgba(255,255,255,0.2),rgba(255,255,255,0.2)),linear-gradient(90deg,#020617_70%,#0A226F_110%)]">创建 Agent</Button>
          <Button variant="claw-primary" size="claw" disabled>创建 Agent</Button>
        </div>
      </div>

      {/* 次要按钮 */}
      <div className="space-y-2">
        <label className="text-sm text-gray-500">次要按钮（claw-outline）</label>
        <div className="grid grid-cols-4 gap-4 items-center">
          <Button variant="claw-outline" size="claw">详细配置</Button>
          <Button variant="claw-outline" size="claw" className="bg-[#f5f5f5] border-[#e3e3e3]">详细配置</Button>
          <Button variant="claw-outline" size="claw" className="bg-white border-[#e3e3e3]">详细配置</Button>
          <Button variant="claw-outline" size="claw" disabled>详细配置</Button>
        </div>
      </div>

      {/* 危险按钮 */}
      <div className="space-y-2">
        <label className="text-sm text-gray-500">危险按钮（destructive）</label>
        <div className="grid grid-cols-4 gap-4 items-center">
          <Button variant="destructive" size="default">删除</Button>
          <Button variant="destructive" size="default" className="bg-red-600">删除</Button>
          <Button variant="destructive" size="default" className="bg-red-700">删除</Button>
          <Button variant="destructive" size="default" disabled>删除</Button>
        </div>
      </div>

      {/* Ghost 按钮 */}
      <div className="space-y-2">
        <label className="text-sm text-gray-500">Ghost 按钮</label>
        <div className="grid grid-cols-4 gap-4 items-center">
          <Button variant="ghost" size="default">Ghost</Button>
          <Button variant="ghost" size="default" className="bg-gray-100">Ghost</Button>
          <Button variant="ghost" size="default" className="bg-gray-200">Ghost</Button>
          <Button variant="ghost" size="default" disabled>Ghost</Button>
        </div>
      </div>

      {/* Link 按钮 */}
      <div className="space-y-2">
        <label className="text-sm text-gray-500">Link 按钮</label>
        <div className="grid grid-cols-4 gap-4 items-center">
          <Button variant="link" size="default">Link</Button>
          <Button variant="link" size="default" className="underline">Link</Button>
          <Button variant="link" size="default" className="underline text-blue-800">Link</Button>
          <Button variant="link" size="default" disabled>Link</Button>
        </div>
      </div>

      {/* 尺寸 */}
      <div className="space-y-2">
        <label className="text-sm text-gray-500">尺寸（大 40px / 中 36px / 小 32px）</label>
        <div className="flex gap-4 items-center">
          <Button variant="claw-primary" size="claw-lg">Large 40</Button>
          <Button variant="claw-primary" size="claw">Default 36</Button>
          <Button variant="claw-primary" size="claw-sm">Small 32</Button>
        </div>
        <div className="flex gap-4 items-center mt-3">
          <Button variant="claw-outline" size="claw-lg">Large 40</Button>
          <Button variant="claw-outline" size="claw">Default 36</Button>
          <Button variant="claw-outline" size="claw-sm">Small 32</Button>
        </div>
      </div>
    </div>
  );
}

function DialogPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">Dialog 弹窗</h2>
      <p className="text-sm text-gray-500">参考 TDesign Dialog：12px圆角、遮罩60%、头部/底部分割线、底部按钮右对齐。</p>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">点击打开弹窗</label>
        <div className="flex gap-3">
          <Dialog>
            <DialogTrigger asChild>
              <Button variant="outline">普通弹窗</Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-md">
              <DialogHeader>
                <DialogTitle>确认操作</DialogTitle>
                <DialogDescription>这是一个弹窗示例，展示 Dialog 组件的样式。</DialogDescription>
              </DialogHeader>
              <div className="px-6 py-4">
                <p className="text-sm text-gray-600">弹窗内容区域，可以放置表单、提示信息等。</p>
              </div>
              <DialogFooter>
                <Button variant="outline">取消</Button>
                <Button variant="dialog-confirm">确认</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          <Dialog>
            <DialogTrigger asChild>
              <Button variant="destructive">危险操作弹窗</Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-sm">
              <DialogHeader>
                <DialogTitle>确认删除</DialogTitle>
                <DialogDescription>此操作不可恢复</DialogDescription>
              </DialogHeader>
              <div className="px-6 py-4">
                <p className="text-sm text-gray-600">删除后数据将无法找回，请谨慎操作。</p>
              </div>
              <DialogFooter>
                <Button variant="outline">取消</Button>
                <Button variant="destructive">确认删除</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>
    </div>
  );
}

function TooltipPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">Tooltip 悬浮提示</h2>
      <TooltipProvider>
        <div className="space-y-3">
          <label className="text-sm text-gray-500">Hover 查看 Tooltip</label>
          <div className="flex gap-6 items-center">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="outline" size="sm">悬停我</Button>
              </TooltipTrigger>
              <TooltipContent>这是一个提示信息</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="inline-flex items-center gap-1 text-sm text-gray-600 cursor-help">
                  用户 ID <Info className="w-3.5 h-3.5 text-gray-400" />
                </span>
              </TooltipTrigger>
              <TooltipContent>用户的唯一标识符，不可修改</TooltipContent>
            </Tooltip>
          </div>
        </div>
      </TooltipProvider>
    </div>
  );
}

function LabelPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">Label 标签</h2>
      <div className="space-y-4">
        <div className="space-y-2">
          <Label>企业名称</Label>
          <Input placeholder="请输入企业名称" />
        </div>
        <div className="space-y-2">
          <Label>邮箱地址 <span className="text-red-500">*</span></Label>
          <Input placeholder="请输入邮箱" />
        </div>
        <div className="space-y-2">
          <Label className="text-gray-400">禁用字段</Label>
          <Input placeholder="不可编辑" disabled />
        </div>
      </div>
    </div>
  );
}

function CheckboxPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">Checkbox 复选框</h2>
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <Checkbox id="c1" />
          <label htmlFor="c1" className="text-sm">未选中</label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox id="c2" defaultChecked />
          <label htmlFor="c2" className="text-sm">已选中</label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox id="c3" disabled />
          <label htmlFor="c3" className="text-sm text-gray-400">禁用</label>
        </div>
        <div className="flex items-center gap-2">
          <Checkbox id="c4" disabled defaultChecked />
          <label htmlFor="c4" className="text-sm text-gray-400">禁用 + 选中</label>
        </div>
      </div>
    </div>
  );
}

function SwitchPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">Switch 开关</h2>
      <div className="space-y-4">
        <div className="flex items-center gap-3">
          <Switch id="s1" />
          <Label htmlFor="s1">关闭</Label>
        </div>
        <div className="flex items-center gap-3">
          <Switch id="s2" defaultChecked />
          <Label htmlFor="s2">开启</Label>
        </div>
        <div className="flex items-center gap-3">
          <Switch id="s3" disabled />
          <Label htmlFor="s3" className="text-gray-400">禁用（关）</Label>
        </div>
        <div className="flex items-center gap-3">
          <Switch id="s4" disabled defaultChecked />
          <Label htmlFor="s4" className="text-gray-400">禁用（开）</Label>
        </div>
      </div>
    </div>
  );
}

function BadgePreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">Badge 徽标</h2>
      <div className="flex gap-3 flex-wrap">
        <Badge>Default</Badge>
        <Badge variant="secondary">Secondary</Badge>
        <Badge variant="outline">Outline</Badge>
        <Badge variant="destructive">Destructive</Badge>
      </div>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">业务状态徽章</label>
        <div className="flex gap-3 flex-wrap">
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium" style={{ background: "rgba(52,199,89,0.12)", color: "#1a8c3a" }}>
            <span className="w-1.5 h-1.5 rounded-full bg-green-500" />运行中
          </span>
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium" style={{ background: "rgba(255,59,48,0.1)", color: "#c0392b" }}>
            <span className="w-1.5 h-1.5 rounded-full bg-red-500" />已停止
          </span>
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium" style={{ background: "rgba(255,149,0,0.1)", color: "#b8640a" }}>
            <span className="w-1.5 h-1.5 rounded-full bg-yellow-500" />待处理
          </span>
        </div>
      </div>
    </div>
  );
}

function TabsPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">Tabs 标签页</h2>
      <Tabs defaultValue="tab1" className="w-full">
        <TabsList>
          <TabsTrigger value="tab1">基础配置</TabsTrigger>
          <TabsTrigger value="tab2">高级设置</TabsTrigger>
          <TabsTrigger value="tab3">操作日志</TabsTrigger>
        </TabsList>
        <TabsContent value="tab1" className="p-4 text-sm text-gray-600">基础配置内容区域</TabsContent>
        <TabsContent value="tab2" className="p-4 text-sm text-gray-600">高级设置内容区域</TabsContent>
        <TabsContent value="tab3" className="p-4 text-sm text-gray-600">操作日志内容区域</TabsContent>
      </Tabs>
    </div>
  );
}

function TextareaPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">Textarea 多行输入</h2>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Normal</label>
        <Textarea placeholder="请输入描述信息..." />
      </div>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Filled</label>
        <Textarea defaultValue="这是一段已经输入的描述内容，展示多行输入的样式效果。" />
      </div>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">Disabled</label>
        <Textarea placeholder="不可编辑" disabled />
      </div>
    </div>
  );
}

function DropdownMenuPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">DropdownMenu 下拉菜单</h2>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">点击触发</label>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm">
              操作 <ChevronDown className="w-4 h-4 ml-1" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>编辑</DropdownMenuItem>
            <DropdownMenuItem>复制</DropdownMenuItem>
            <DropdownMenuItem>移动</DropdownMenuItem>
            <DropdownMenuItem className="text-red-600">删除</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              <MoreHorizontal className="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>查看详情</DropdownMenuItem>
            <DropdownMenuItem>导出</DropdownMenuItem>
            <DropdownMenuItem className="text-red-600">删除</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}

function PopoverPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">Popover 气泡浮层</h2>
      <div className="space-y-3">
        <label className="text-sm text-gray-500">点击打开</label>
        <Popover>
          <PopoverTrigger asChild>
            <Button variant="outline" size="sm">打开气泡</Button>
          </PopoverTrigger>
          <PopoverContent className="w-64">
            <div className="space-y-2">
              <p className="text-sm font-medium">筛选条件</p>
              <p className="text-xs text-gray-500">这里可以放置筛选表单、信息卡片等内容。</p>
            </div>
          </PopoverContent>
        </Popover>
      </div>
    </div>
  );
}

function ProgressPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">Progress 进度条</h2>
      <div className="space-y-4">
        <div className="space-y-2">
          <label className="text-sm text-gray-500">0%</label>
          <Progress value={0} />
        </div>
        <div className="space-y-2">
          <label className="text-sm text-gray-500">33%</label>
          <Progress value={33} />
        </div>
        <div className="space-y-2">
          <label className="text-sm text-gray-500">66%</label>
          <Progress value={66} />
        </div>
        <div className="space-y-2">
          <label className="text-sm text-gray-500">100%</label>
          <Progress value={100} />
        </div>
      </div>
    </div>
  );
}

function AlertDialogPreview() {
  return (
    <div className="space-y-6">
      <h2 className="text-base font-semibold text-gray-900">AlertDialog 确认弹窗</h2>
      <p className="text-sm text-gray-500">用于危险操作确认，红色确认按钮。</p>
      <Dialog>
        <DialogTrigger asChild>
          <Button variant="destructive">删除项目</Button>
        </DialogTrigger>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>此操作不可恢复，确认要删除该项目吗？</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline">取消</Button>
            <Button variant="destructive">确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

/* ────────────────────────────────────────────────────────────────────
 * TablePreview - Table 组件全场景可视化预览
 * 覆盖：密度 / variant / 固定列 / 多列同侧固定 / 选中态 /
 *       展开行 / 空数据 / Footer / 与 Pagination 的搭配
 * ──────────────────────────────────────────────────────────────────── */

type DemoUser = {
  id: string;
  name: string;
  email: string;
  dept: string;
  role: string;
  status: "running" | "stopped" | "pending";
  quota: string;
  lastActive: string;
};

const DEMO_USERS: DemoUser[] = [
  { id: "ins-hermes01", name: "Hermes 助手", email: "hermes@tencent.com", dept: "技术工程事业群 / 平台开发部", role: "管理员", status: "running", quota: "1,200,000 / 2,000,000", lastActive: "2026-06-05 21:30" },
  { id: "ins-athena02", name: "Athena 数据分析", email: "athena@tencent.com", dept: "云与智慧产业事业群", role: "成员", status: "running", quota: "320,500 / 500,000", lastActive: "2026-06-05 18:12" },
  { id: "ins-orion03", name: "Orion 安全审计", email: "orion@tencent.com", dept: "技术工程事业群 / 安全平台部", role: "审计员", status: "stopped", quota: "0 / 100,000", lastActive: "2026-05-28 10:04" },
  { id: "ins-lyra04", name: "Lyra 文档助手", email: "lyra@tencent.com", dept: "互动娱乐事业群", role: "成员", status: "pending", quota: "—", lastActive: "—" },
  { id: "ins-vega05", name: "Vega 客服机器人", email: "vega@tencent.com", dept: "云与智慧产业事业群 / 客服中台", role: "成员", status: "running", quota: "880,200 / 1,000,000", lastActive: "2026-06-05 20:55" },
];

// 表格状态列：统一使用标准 StatusTag 组件（mode="text" 无底色彩色文字，符合表格状态列惯例）
function StatusBadge({ status }: { status: DemoUser["status"] }) {
  const map = {
    running: { variant: "green", text: "运行中" },
    stopped: { variant: "red", text: "已停止" },
    pending: { variant: "orange", text: "待启用" },
  } as const;
  const s = map[status];
  return (
    <StatusTag mode="text" variant={s.variant}>
      {s.text}
    </StatusTag>
  );
}

function TablePreview() {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [expandedId, setExpandedId] = useState<string | null>("ins-hermes01");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const toggleRow = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  return (
    <div className="space-y-12">
      <div className="space-y-2">
        <h2 className="text-base font-semibold text-gray-900">Table 表格</h2>
        <p className="text-sm text-gray-500">企业管控端表格的"权威标准"。下方覆盖密度、变体、固定列、选中、展开、空态、分页等全部关键场景的标准用法。</p>
      </div>

      {/* 1. 默认 - gray-header + default 密度 + 完整列 */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-gray-900">① 默认：gray-header + density="default" + 操作列</h3>
        <p className="text-xs text-gray-500">行高 54px，表头灰底，body 白底；操作列使用 link 蓝色按钮。</p>
        <SurfaceCard className="overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>实例 ID</TableHead>
                <TableHead>名称</TableHead>
                <TableHead>所属部门</TableHead>
                <TableHead>角色</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>最近活跃</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {DEMO_USERS.slice(0, 4).map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="text-gray-500">{u.id}</TableCell>
                  <TableCell>{u.name}</TableCell>
                  <TableCell>{u.dept}</TableCell>
                  <TableCell>{u.role}</TableCell>
                  <TableCell><StatusBadge status={u.status} /></TableCell>
                  <TableCell className="text-gray-500">{u.lastActive}</TableCell>
                  <TableActionCell>
                    <Button variant="link">编辑</Button>
                    <Button variant="link">详情</Button>
                    <Button variant="link">删除</Button>
                  </TableActionCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </SurfaceCard>
      </section>

      {/* 2. compact 密度 */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-gray-900">② compact 密度（行高 40px / 表头字色变浅）</h3>
        <SurfaceCard className="overflow-hidden">
          <Table density="compact">
            <TableHeader>
              <TableRow>
                <TableHead>实例 ID</TableHead>
                <TableHead>名称</TableHead>
                <TableHead>角色</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>额度使用</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {DEMO_USERS.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="text-gray-500">{u.id}</TableCell>
                  <TableCell>{u.name}</TableCell>
                  <TableCell>{u.role}</TableCell>
                  <TableCell><StatusBadge status={u.status} /></TableCell>
                  <TableCell>{u.quota}</TableCell>
                  <TableActionCell>
                    <Button variant="link">编辑</Button>
                    <Button variant="link">删除</Button>
                  </TableActionCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </SurfaceCard>
      </section>

      {/* 3. variant="white" - 放在灰底容器中（必须非白底） */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-gray-900">③ variant="white"（白卡浮起，仅用于非白底容器）</h3>
        <p className="text-xs text-gray-500">禁止在 SurfaceCard / 白底 Dialog 内使用，会"白上加白"。</p>
        <div className="p-6 rounded-xl bg-[linear-gradient(135deg,#EEF2FF_0%,#E0F2FE_100%)]">
          <Table variant="white">
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>角色</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {DEMO_USERS.slice(0, 3).map((u) => (
                <TableRow key={u.id}>
                  <TableCell>{u.name}</TableCell>
                  <TableCell>{u.role}</TableCell>
                  <TableCell><StatusBadge status={u.status} /></TableCell>
                  <TableActionCell>
                    <Button variant="link">查看</Button>
                  </TableActionCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </section>

      {/* 4. 横向滚动 + 自动固定列 */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-gray-900">④ 横向滚动 + 自动固定首列 / 操作列（scrollX=1500）</h3>
        <p className="text-xs text-gray-500">向右滚动后，左侧"实例 ID"首列与右侧操作列自动保持 sticky，并出现边界阴影。</p>
        <SurfaceCard className="overflow-hidden">
          <Table scrollX={1500}>
            <TableHeader>
              <TableRow>
                <TableHead>实例 ID</TableHead>
                <TableHead>名称</TableHead>
                <TableHead>邮箱</TableHead>
                <TableHead>所属部门</TableHead>
                <TableHead>角色</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>额度使用</TableHead>
                <TableHead>最近活跃</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {DEMO_USERS.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="text-gray-500">{u.id}</TableCell>
                  <TableCell>{u.name}</TableCell>
                  <TableCell className="text-gray-500">{u.email}</TableCell>
                  <TableCell>{u.dept}</TableCell>
                  <TableCell>{u.role}</TableCell>
                  <TableCell><StatusBadge status={u.status} /></TableCell>
                  <TableCell>{u.quota}</TableCell>
                  <TableCell className="text-gray-500">{u.lastActive}</TableCell>
                  <TableActionCell>
                    <Button variant="link">编辑</Button>
                    <Button variant="link">详情</Button>
                    <Button variant="link">删除</Button>
                  </TableActionCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </SurfaceCard>
      </section>

      {/* 5. 多列同侧固定 + 行选中 */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-gray-900">⑤ 多列同侧固定（checkbox + 名称同时 fixed="left"）+ 选中态</h3>
        <p className="text-xs text-gray-500">复选框列与名称列都 fixed="left"，自动累加偏移；勾选通过 Checkbox 自身表达，选中行不再有蓝色背景。</p>
        <SurfaceCard className="overflow-hidden">
          <Table scrollX={1400}>
            <TableHeader>
              <TableRow>
                <TableHead fixed="left" fixedShadow={false} className="w-[48px]">
                  <Checkbox />
                </TableHead>
                <TableHead fixed="left" className="w-[200px]">名称</TableHead>
                <TableHead className="w-[140px]">实例 ID</TableHead>
                <TableHead className="w-[200px]">邮箱</TableHead>
                <TableHead className="w-[260px]">所属部门</TableHead>
                <TableHead className="w-[80px]">角色</TableHead>
                <TableHead className="w-[100px]">状态</TableHead>
                <TableHead className="w-[180px]">额度使用</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {DEMO_USERS.map((u) => {
                const checked = selectedIds.has(u.id);
                return (
                  <TableRow key={u.id} data-state={checked ? "selected" : undefined}>
                    <TableCell fixed="left" fixedShadow={false} className="w-[48px]">
                      <Checkbox checked={checked} onCheckedChange={() => toggleRow(u.id)} />
                    </TableCell>
                    <TableCell fixed="left" className="w-[200px]">{u.name}</TableCell>
                    <TableCell className="text-gray-500 w-[140px]">{u.id}</TableCell>
                    <TableCell className="text-gray-500 w-[200px]">{u.email}</TableCell>
                    <TableCell className="w-[260px]">{u.dept}</TableCell>
                    <TableCell className="w-[80px]">{u.role}</TableCell>
                    <TableCell className="w-[100px]"><StatusBadge status={u.status} /></TableCell>
                    <TableCell className="w-[180px]">{u.quota}</TableCell>
                    <TableActionCell fixed="right">
                      <Button variant="link">编辑</Button>
                      <Button variant="link">删除</Button>
                    </TableActionCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </SurfaceCard>
      </section>

      {/* 6. TableExpandedRow 展开行 */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-gray-900">⑥ 展开行 TableExpandedRow（白底、禁用 hover）</h3>
        <SurfaceCard className="overflow-hidden">
          <Table variant="white">
            <TableHeader>
              <TableRow>
                <TableHead className="w-[36px]" />
                <TableHead>名称</TableHead>
                <TableHead>角色</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {DEMO_USERS.slice(0, 3).map((u) => (
                <Fragment key={u.id}>
                  <TableRow>
                    <TableCell>
                      <button
                        type="button"
                        className="inline-flex items-center justify-center w-5 h-5 rounded hover:bg-gray-100 transition-colors"
                        onClick={() => setExpandedId(expandedId === u.id ? null : u.id)}
                        aria-label="toggle expand"
                      >
                        <ChevronDown className={`w-4 h-4 transition-transform ${expandedId === u.id ? "" : "-rotate-90"}`} />
                      </button>
                    </TableCell>
                    <TableCell>{u.name}</TableCell>
                    <TableCell>{u.role}</TableCell>
                    <TableCell><StatusBadge status={u.status} /></TableCell>
                    <TableActionCell>
                      <Button variant="link">编辑</Button>
                    </TableActionCell>
                  </TableRow>
                  {expandedId === u.id && (
                    <TableExpandedRow>
                      <TableCell colSpan={5}>
                        <div className="px-2 py-3 space-y-1 text-gray-600">
                          <div><span className="text-gray-400 mr-2">实例 ID：</span>{u.id}</div>
                          <div><span className="text-gray-400 mr-2">邮箱：</span>{u.email}</div>
                          <div><span className="text-gray-400 mr-2">所属部门：</span>{u.dept}</div>
                          <div><span className="text-gray-400 mr-2">额度使用：</span>{u.quota}</div>
                        </div>
                      </TableCell>
                    </TableExpandedRow>
                  )}
                </Fragment>
              ))}
            </TableBody>
          </Table>
        </SurfaceCard>
      </section>

      {/* 7. TableFooter + TableCaption */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-gray-900">⑦ TableFooter + TableCaption</h3>
        <SurfaceCard className="overflow-hidden">
          <Table>
            <TableCaption>近 7 日 Token 使用汇总</TableCaption>
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>已用</TableHead>
                <TableHead>配额</TableHead>
                <TableHead className="text-right pr-4">使用率</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell>Hermes 助手</TableCell>
                <TableCell>1,200,000</TableCell>
                <TableCell>2,000,000</TableCell>
                <TableCell className="text-right">60%</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>Athena 数据分析</TableCell>
                <TableCell>320,500</TableCell>
                <TableCell>500,000</TableCell>
                <TableCell className="text-right">64%</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>Vega 客服机器人</TableCell>
                <TableCell>880,200</TableCell>
                <TableCell>1,000,000</TableCell>
                <TableCell className="text-right">88%</TableCell>
              </TableRow>
            </TableBody>
            <TableFooter>
              <TableRow>
                <TableCell>合计</TableCell>
                <TableCell>2,400,700</TableCell>
                <TableCell>3,500,000</TableCell>
                <TableCell className="text-right">68.6%</TableCell>
              </TableRow>
            </TableFooter>
          </Table>
        </SurfaceCard>
      </section>

      {/* 8. 空数据态 */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-gray-900">⑧ 空数据态（标准 Empty 组件 + 统一插画）</h3>
        <SurfaceCard className="overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>角色</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell colSpan={4}>
                  <Empty className="border-none py-8">
                    <EmptyHeader>
                      <EmptyMedia />
                      <EmptyTitle>暂无数据</EmptyTitle>
                      <EmptyDescription>当前没有符合条件的记录，调整筛选条件后重试。</EmptyDescription>
                    </EmptyHeader>
                  </Empty>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </SurfaceCard>
      </section>

      {/* 9. 与 Pagination 搭配（标准结构） */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-gray-900">⑨ 标准结构：SurfaceCard + Table + Pagination</h3>
        <SurfaceCard className="overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>实例 ID</TableHead>
                <TableHead>名称</TableHead>
                <TableHead>角色</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {DEMO_USERS.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="text-gray-500">{u.id}</TableCell>
                  <TableCell>{u.name}</TableCell>
                  <TableCell>{u.role}</TableCell>
                  <TableCell><StatusBadge status={u.status} /></TableCell>
                  <TableActionCell>
                    <Button variant="link">编辑</Button>
                    <Button variant="link">删除</Button>
                  </TableActionCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <div className="px-4 py-3 border-t border-gray-200">
            <Pagination
              total={142}
              current={page}
              pageSize={pageSize}
              showTotal={(t) => `共 ${t} 条记录`}
              showSizeChanger
              onChange={(p, s) => { setPage(p); setPageSize(s); }}
            />
          </div>
        </SurfaceCard>
      </section>

      {/* 10. 表头背景一致性 */}
      <section className="space-y-3">
        <h3 className="text-sm font-semibold text-gray-900">⑩ 表头背景一致性（固定列 vs 普通列表头同色）</h3>
        <p className="text-xs text-gray-500">
          表头所有单元格（普通列 + 固定列）统一使用 <code>var(--table-head-bg, var(--bg-grey-normal))</code>：
          跟随 <code>TableHeader</code> 按 variant 注入的底色，缺失时统一 fallback 到灰。横滚下方两个表格，表头始终整条同色。
        </p>

        <p className="text-xs text-gray-400">gray-header（默认）：表头应整条 #FAFBFD 灰，含首列/操作列固定列</p>
        <SurfaceCard className="overflow-hidden">
          <Table scrollX={1500}>
            <TableHeader>
              <TableRow>
                <TableHead>实例 ID</TableHead>
                <TableHead>名称</TableHead>
                <TableHead>邮箱</TableHead>
                <TableHead>所属部门</TableHead>
                <TableHead>角色</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>额度使用</TableHead>
                <TableHead>最近活跃</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {DEMO_USERS.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="text-gray-500">{u.id}</TableCell>
                  <TableCell>{u.name}</TableCell>
                  <TableCell className="text-gray-500">{u.email}</TableCell>
                  <TableCell>{u.dept}</TableCell>
                  <TableCell>{u.role}</TableCell>
                  <TableCell><StatusBadge status={u.status} /></TableCell>
                  <TableCell>{u.quota}</TableCell>
                  <TableCell className="text-gray-500">{u.lastActive}</TableCell>
                  <TableActionCell>
                    <Button variant="link">编辑</Button>
                    <Button variant="link">删除</Button>
                  </TableActionCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </SurfaceCard>

        <p className="text-xs text-gray-400 mt-2">white variant：表头应整条纯白，含固定列</p>
        <div className="p-6 rounded-xl bg-[linear-gradient(135deg,#EEF2FF_0%,#E0F2FE_100%)]">
          <Table variant="white" scrollX={1500}>
            <TableHeader>
              <TableRow>
                <TableHead>实例 ID</TableHead>
                <TableHead>名称</TableHead>
                <TableHead>邮箱</TableHead>
                <TableHead>所属部门</TableHead>
                <TableHead>角色</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>最近活跃</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {DEMO_USERS.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="text-gray-500">{u.id}</TableCell>
                  <TableCell>{u.name}</TableCell>
                  <TableCell className="text-gray-500">{u.email}</TableCell>
                  <TableCell>{u.dept}</TableCell>
                  <TableCell>{u.role}</TableCell>
                  <TableCell><StatusBadge status={u.status} /></TableCell>
                  <TableCell className="text-gray-500">{u.lastActive}</TableCell>
                  <TableActionCell>
                    <Button variant="link">编辑</Button>
                    <Button variant="link">删除</Button>
                  </TableActionCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </section>
    </div>
  );
}

function GenericPreview({ name }: { name: string }) {
  return (
    <div className="space-y-4">
      <h2 className="text-base font-semibold text-gray-900">{name}.tsx</h2>
      <p className="text-sm text-gray-500">该组件预览尚未配置，后续改造完成后将添加详细状态展示。</p>
      <p className="text-xs text-gray-400">
        路径：<code className="bg-gray-100 px-1 py-0.5 rounded">client/src/components/ui/{name}.tsx</code>
      </p>
    </div>
  );
}

const PREVIEWS: Record<string, (() => ReactElement) | null> = {
  input: InputPreview,
  select: SelectPreview,
  button: ButtonPreview,
  dialog: DialogPreview,
  tooltip: TooltipPreview,
  label: LabelPreview,
  checkbox: CheckboxPreview,
  switch: SwitchPreview,
  badge: BadgePreview,
  tabs: TabsPreview,
  textarea: TextareaPreview,
  "dropdown-menu": DropdownMenuPreview,
  popover: PopoverPreview,
  progress: ProgressPreview,
  "alert-dialog": AlertDialogPreview,
  table: TablePreview,
};

export default function ComponentPreview() {
  const [, params] = useRoute("/component-preview/:name");
  const name = params?.name || "";
  const Preview = PREVIEWS[name];

  // Table 等"宽组件"需要更宽的容器，普通表单组件保持窄一些更聚焦
  const wideNames = new Set(["table"]);
  const isWide = wideNames.has(name);

  return (
    <div className={`${isWide ? "max-w-[1280px]" : "max-w-[640px]"} mx-auto p-8`}>
      {Preview ? <Preview /> : <GenericPreview name={name} />}
      <p className="mt-8 text-xs text-gray-400 text-center">
        交互提示：hover 查看 hover 态，点击查看 focus/展开态
      </p>
    </div>
  );
}
