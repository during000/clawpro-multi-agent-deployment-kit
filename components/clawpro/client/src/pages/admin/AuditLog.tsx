/**
 * AuditLog - 管控端操作审计页
 */
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Pagination } from "@/components/ui/pagination";
import { Input } from "@/components/ui/input";
import { SurfaceCard } from "@/components/ui/Surface";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { StatusTag } from "@/components/ui/status-tag";
import { HelperText } from "@/components/ui/Typography";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import { Search, ClipboardList, RefreshCw, ChevronLeft, ChevronRight } from "lucide-react";
import { toast } from "sonner";
import { DatePicker } from "@/components/ui/date-picker";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { useAuditLogCalendarBillingExempt } from "./AuditLog/useAuditLogCalendarBillingExempt";

const PAGE_SIZE = 10;

const MOCK_LOGS = [
  {
    id: "log-001", operator: "admin@acompany.com", action: "updateMember",
    requestTime: "2026-03-09 15:45:37", responseTime: "2026-03-09 15:45:38", success: true,
    detail: {
      eventId: "6af57777-10bd-4032-b881-f2e2f8872cd0",
      request: '{"memberId":"alice@acompany.com","openclawLimit":5,"tokenLimit":100000}',
      endDate: "2026-03-09 15:45:38", serviceAccount: "true",
      userAgent: "okhttp/4.10.0", invokerName: "ak.ADMIN",
      duration: "158", application: "openclaw-enterprise",
      sourceIp: "30.42.219.99", success: "true",
      action: "updateMember", invokerId: "1",
      startDate: "2026-03-09 15:45:37",
    },
  },
  {
    id: "log-002", operator: "admin@acompany.com", action: "createModel",
    requestTime: "2026-03-09 14:30:12", responseTime: "2026-03-09 14:30:13", success: true,
    detail: {
      eventId: "7bf68888-20cd-5143-c992-g3f3g9983de1",
      request: '{"provider":"腾讯云DeepSeek","version":"DeepSeek V3 0324","dailyLimit":500000}',
      endDate: "2026-03-09 14:30:13", serviceAccount: "true",
      userAgent: "okhttp/4.10.0", invokerName: "ak.ADMIN",
      duration: "203", application: "openclaw-enterprise",
      sourceIp: "30.42.219.99", success: "true",
      action: "createModel", invokerId: "1",
      startDate: "2026-03-09 14:30:12",
    },
  },
  {
    id: "log-003", operator: "superadmin@acompany.com", action: "updateBasicInfo",
    requestTime: "2026-03-09 11:20:05", responseTime: "2026-03-09 11:20:06", success: true,
    detail: {
      eventId: "8cg79999-31de-6254-d003-h4g4h0094ef2",
      request: '{"siteName":"A公司企业版OpenClaw","siteDesc":"企业专属AI助理平台"}',
      endDate: "2026-03-09 11:20:06", serviceAccount: "true",
      userAgent: "okhttp/4.10.0", invokerName: "ak.SUPERADMIN",
      duration: "89", application: "openclaw-enterprise",
      sourceIp: "30.42.219.88", success: "true",
      action: "updateBasicInfo", invokerId: "0",
      startDate: "2026-03-09 11:20:05",
    },
  },
  {
    id: "log-004", operator: "admin@acompany.com", action: "deleteMember",
    requestTime: "2026-03-08 16:45:22", responseTime: "2026-03-08 16:45:22", success: false,
    detail: {
      eventId: "9dh80000-42ef-7365-e114-i5h5i1105fg3",
      request: '{"memberId":"frank@acompany.com"}',
      endDate: "2026-03-08 16:45:22", serviceAccount: "true",
      userAgent: "okhttp/4.10.0", invokerName: "ak.ADMIN",
      duration: "45", application: "openclaw-enterprise",
      sourceIp: "30.42.219.99", success: "false",
      action: "deleteMember", invokerId: "1",
      startDate: "2026-03-08 16:45:22",
    },
  },
  {
    id: "log-005", operator: "admin@acompany.com", action: "updateSecurityGroup",
    requestTime: "2026-03-08 10:12:33", responseTime: "2026-03-08 10:12:34", success: true,
    detail: {
      eventId: "0ei91111-53fg-8476-f225-j6i6j2216gh4",
      request: '{"ruleType":"inbound","source":"0.0.0.0/0","protocol":"TCP","port":"18789","policy":"允许"}',
      endDate: "2026-03-08 10:12:34", serviceAccount: "true",
      userAgent: "okhttp/4.10.0", invokerName: "ak.ADMIN",
      duration: "112", application: "openclaw-enterprise",
      sourceIp: "30.42.219.99", success: "true",
      action: "updateSecurityGroup", invokerId: "1",
      startDate: "2026-03-08 10:12:33",
    },
  },
  {
    id: "log-006", operator: "admin@acompany.com", action: "createMember",
    requestTime: "2026-03-07 09:30:11", responseTime: "2026-03-07 09:30:12", success: true,
    detail: {
      eventId: "1fj02222-64gh-9587-g336-k7j7k3327hi5",
      request: '{"memberId":"grace@acompany.com","role":"member"}',
      endDate: "2026-03-07 09:30:12", serviceAccount: "true",
      userAgent: "okhttp/4.10.0", invokerName: "ak.ADMIN",
      duration: "134", application: "openclaw-enterprise",
      sourceIp: "30.42.219.99", success: "true",
      action: "createMember", invokerId: "1",
      startDate: "2026-03-07 09:30:11",
    },
  },
  {
    id: "log-007", operator: "admin@acompany.com", action: "resetPassword",
    requestTime: "2026-03-07 08:15:44", responseTime: "2026-03-07 08:15:44", success: true,
    detail: {
      eventId: "2gk13333-75hi-0698-h447-l8k8l4438ij6",
      request: '{"memberId":"henry@acompany.com"}',
      endDate: "2026-03-07 08:15:44", serviceAccount: "true",
      userAgent: "okhttp/4.10.0", invokerName: "ak.ADMIN",
      duration: "67", application: "openclaw-enterprise",
      sourceIp: "30.42.219.99", success: "true",
      action: "resetPassword", invokerId: "1",
      startDate: "2026-03-07 08:15:44",
    },
  },
  {
    id: "log-008", operator: "superadmin@acompany.com", action: "importImage",
    requestTime: "2026-03-06 17:22:09", responseTime: "2026-03-06 17:22:11", success: true,
    detail: {
      eventId: "3hl24444-86ij-1709-i558-m9l9m5549jk7",
      request: '{"imageId":"img-abc123","imageName":"Agent镜像v2.1"}',
      endDate: "2026-03-06 17:22:11", serviceAccount: "true",
      userAgent: "okhttp/4.10.0", invokerName: "ak.SUPERADMIN",
      duration: "1842", application: "openclaw-enterprise",
      sourceIp: "30.42.219.88", success: "true",
      action: "importImage", invokerId: "0",
      startDate: "2026-03-06 17:22:09",
    },
  },
  {
    id: "log-009", operator: "admin@acompany.com", action: "deleteModel",
    requestTime: "2026-03-06 14:05:33", responseTime: "2026-03-06 14:05:33", success: false,
    detail: {
      eventId: "4im35555-97jk-2810-j669-n0m0n6650kl8",
      request: '{"modelId":"model-xyz789"}',
      endDate: "2026-03-06 14:05:33", serviceAccount: "true",
      userAgent: "okhttp/4.10.0", invokerName: "ak.ADMIN",
      duration: "23", application: "openclaw-enterprise",
      sourceIp: "30.42.219.99", success: "false",
      action: "deleteModel", invokerId: "1",
      startDate: "2026-03-06 14:05:33",
    },
  },
  {
    id: "log-010", operator: "admin@acompany.com", action: "updateChannelConfig",
    requestTime: "2026-03-05 16:48:27", responseTime: "2026-03-05 16:48:28", success: true,
    detail: {
      eventId: "5jn46666-08kl-3921-k770-o1n1o7761lm9",
      request: '{"channel":"feishu","appId":"cli_abc","appSecret":"xxx"}',
      endDate: "2026-03-05 16:48:28", serviceAccount: "true",
      userAgent: "okhttp/4.10.0", invokerName: "ak.ADMIN",
      duration: "245", application: "openclaw-enterprise",
      sourceIp: "30.42.219.99", success: "true",
      action: "updateChannelConfig", invokerId: "1",
      startDate: "2026-03-05 16:48:27",
    },
  },
  {
    id: "log-011", operator: "superadmin@acompany.com", action: "updateGlobalQuota",
    requestTime: "2026-03-05 10:30:00", responseTime: "2026-03-05 10:30:01", success: true,
    detail: {
      eventId: "6ko57777-19lm-4032-l881-p2o2p8872mn0",
      request: '{"dailyGlobalLimit":2000000}',
      endDate: "2026-03-05 10:30:01", serviceAccount: "true",
      userAgent: "okhttp/4.10.0", invokerName: "ak.SUPERADMIN",
      duration: "98", application: "openclaw-enterprise",
      sourceIp: "30.42.219.88", success: "true",
      action: "updateGlobalQuota", invokerId: "0",
      startDate: "2026-03-05 10:30:00",
    },
  },
  {
    id: "log-012", operator: "admin@acompany.com", action: "disableMember",
    requestTime: "2026-03-04 13:22:15", responseTime: "2026-03-04 13:22:15", success: true,
    detail: {
      eventId: "7lp68888-20mn-5143-m992-q3p3q9983no1",
      request: '{"memberId":"ivan@acompany.com","status":"disabled"}',
      endDate: "2026-03-04 13:22:15", serviceAccount: "true",
      userAgent: "okhttp/4.10.0", invokerName: "ak.ADMIN",
      duration: "56", application: "openclaw-enterprise",
      sourceIp: "30.42.219.99", success: "true",
      action: "disableMember", invokerId: "1",
      startDate: "2026-03-04 13:22:15",
    },
  },
];

export default function AuditLog() {
  // 停服态下，本页两个"开始日期 / 结束日期" DatePicker 弹出的日历面板需保持 100% 可用。
  // 触发器已通过外层 <div data-billing-exempt> 打标；但面板经 Radix Portal 挂到<body>，
  // 不在触发器的祖先链上，因此需要独立的页面级 hook 打标 + 注入 CSS 补充规则；
  // 详见 ./AuditLog/useAuditLogCalendarBillingExempt.ts 头部注释。
  useAuditLogCalendarBillingExempt();
  const [search, setSearch] = useState("");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [selectedLog, setSelectedLog] = useState<(typeof MOCK_LOGS)[0] | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [page, setPage] = useState(1);

  const handleRefresh = () => {
    setRefreshing(true);
    setTimeout(() => { setRefreshing(false); toast.success("列表已刷新"); }, 1000);
  };

  const hasFilter = search || dateFrom || dateTo;

  const filtered = MOCK_LOGS.filter((log) => {
    const matchSearch = !search || log.operator.includes(search) || log.action.includes(search);
    const logDate = log.requestTime.slice(0, 10);
    const matchFrom = !dateFrom || logDate >= dateFrom;
    const matchTo = !dateTo || logDate <= dateTo;
    return matchSearch && matchFrom && matchTo;
  });

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages);
  const paged = filtered.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE);

  const handleSearch = (v: string) => { setSearch(v); setPage(1); };
  const handleDateFrom = (v: string) => { setDateFrom(v); setPage(1); };
  const handleDateTo = (v: string) => { setDateTo(v); setPage(1); };

  return (
    <>
      <div className="page-enter">
        <div className="mb-8">
          <AdminPageHeader title="操作记录" description="记录管理员在管控端的所有操作，包括 API 调用详情。" />
        </div>

        {/* Filters */}
        <div className="flex flex-wrap gap-3 mb-4 items-center">
          <div className="relative flex-1 min-w-48 max-w-xs">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#A3A3A3]" />
            <Input
              placeholder="搜索操作人或操作事件"
              value={search}
              onChange={(e) => handleSearch(e.target.value)}
              className="pl-9 bg-white"
            />
          </div>
          {/* 日期范围筛选
            * 停服态豁免：日期筛选属查看类操作（仅改变列表筛选口径，不产生业务变更），
            * 需保持100% 不透明与正常交互。
            * 组件内部未透传 disabled，"停服前已禁用则延续禁用"约束
            * 通过 DatePicker 自身的 disabled 属性依然生效（此处无）。*/}
          <div className="flex items-center gap-2" data-billing-exempt>
            <DatePicker
              value={dateFrom}
              onChange={handleDateFrom}
              placeholder="开始日期"
            />
            <span className="text-[#A3A3A3] text-sm shrink-0">—</span>
            <DatePicker
              value={dateTo}
              onChange={handleDateTo}
              placeholder="结束日期"
            />
          </div>
          {hasFilter && (
            <Button variant="outline" size="sm" onClick={() => { setSearch(""); setDateFrom(""); setDateTo(""); setPage(1); }}>
              清除筛选
            </Button>
          )}
          {/* 刷新按钮
            * 停服态豁免：刷新仅重新拉取当前列表数据展示，不产生业务变更，
            * 需保持 100% 不透明与正常交互。
            * "停服前已禁用则延续禁用"：disabled={refreshing} 是页面本地加载态
            * （点击刷新期间避免并发），豁免祖先不影响原生 disabled 的呈现与交互拦截。 */}
          <Button
            variant="claw-outline"
            size="icon"
            onClick={handleRefresh}
            disabled={refreshing}
            title="刷新列表"
            className="w-9 h-9"
            data-billing-exempt
          >
            <RefreshCw className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
          </Button>
        </div>

        {/* Table */}
        <SurfaceCard className="overflow-hidden">
          <Table variant="white">
            <TableHeader>
              <TableRow>
                <TableHead className="w-[26%]">操作人的用户 ID</TableHead>
                <TableHead className="w-[22%]">操作事件</TableHead>
                <TableHead className="w-[22%]">请求时间</TableHead>
                <TableHead className="w-[22%]">返回时间</TableHead>
                <TableHead className="w-[8%]">执行结果</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paged.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5}>
                    <div className="text-center py-12">
                      <HelperText>暂无操作记录</HelperText>
                    </div>
                  </TableCell>
                </TableRow>
              ) : paged.map((log) => (
                <TableRow key={log.id}>
                  <TableCell>{log.operator}</TableCell>
                  <TableCell>
                    <span className="font-mono bg-[#f5f5f5] px-2 py-0.5 rounded">{log.action}</span>
                  </TableCell>
                  <TableCell>{log.requestTime}</TableCell>
                  <TableCell>{log.responseTime}</TableCell>
                  <TableCell>
                    {log.success ? (
                      <StatusTag mode="text" variant="green">成功</StatusTag>
                    ) : (
                      <StatusTag mode="text" variant="red">失败</StatusTag>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {/* Footer: count + pagination
            * 停服态豁免：翻页 / 每页条数属查看类操作（仅切换列表窗口，不产生业务变更），
            * 需保持 100% 不透明与正常交互。
            * Pagination 内部对首/末页仍会自然禁用对应箭头（原生 disabled），
            * "停服前已禁用则延续禁用"约束通过组件自身逻辑依然生效。 */}
          <div className="px-4 py-2 border-t border-gray-200" data-billing-exempt>
            <Pagination
              total={filtered.length}
              current={safePage}
              pageSize={PAGE_SIZE}
              showTotal={(total) => `共 ${total} 条记录`}
              className="w-full justify-between"
              hideOnSinglePage
              onChange={(p) => setPage(p)}
            />
          </div>
        </SurfaceCard>
      </div>

      {/* Detail Dialog */}
      <Dialog open={!!selectedLog} onOpenChange={() => setSelectedLog(null)}>
        <DialogContent className="sm:max-w-[720px]">
          <DialogHeader>
            <DialogTitle>
              消息详情
            </DialogTitle>
          </DialogHeader>
          {selectedLog && (
            <div className="bg-gray-950 rounded-[4px] p-5 font-mono text-sm overflow-auto max-h-96">
              <div className="text-[#A3A3A3] mb-3">{"{"} <span className="text-[#737373] text-xs">{Object.keys(selectedLog.detail).length} items</span></div>
              {Object.entries(selectedLog.detail).map(([key, value]) => (
                <div key={key} className="ml-4 mb-1.5">
                  <span className="text-[#A3A3A3]">"{key}"</span>
                  <span className="text-[#737373]"> : </span>
                  <span className="text-orange-400">"{value}"</span>
                </div>
              ))}
              <div className="text-[#A3A3A3]">{"}"}</div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setSelectedLog(null)}>退出</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
