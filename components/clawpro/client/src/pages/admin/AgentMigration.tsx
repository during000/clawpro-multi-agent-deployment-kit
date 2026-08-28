/**
 * MigrationManagement - 智能体迁移（单页分区布局）
 * 三个区域：① 源端配置 ② 迁移映射 ③ 执行 & 进度
 */
import { useState, useCallback, useMemo } from "react";
import { useLocation } from "wouter";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Switch } from "@/components/ui/switch";
import { Checkbox } from "@/components/ui/checkbox";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  Tooltip, TooltipContent, TooltipProvider, TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Collapsible, CollapsibleContent, CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { toast } from "sonner";
import {
  Server, Upload, ArrowRight, CheckCircle2, XCircle,
  Loader2, Copy, Terminal, Trash2, Plus, Download, Eye, EyeOff,
  HelpCircle, Clock, RefreshCw, ExternalLink, CircleAlert,
  ChevronRight, ChevronDown, Box, Play, History, Info, Cloud, Database, Search,
  ArrowRightLeft, FileSpreadsheet, ArrowLeft,
} from "lucide-react";
import { AdminPageHeader } from "@/components/ui/admin-page-header";

// ==================== Types ====================

type SourceType = "lighthouse" | "eks" | "idc" | "aliyun" | "volcengine" | "huaweicloud";
type TaskStatus = "pending" | "uploading" | "restoring" | "success" | "failed";
type EditMode = "table" | "csv";

interface MappingRow {
  id: string;
  srcId: string;
  targetUser: string;
  targetInstance: string;
  targetInstanceId: string;
  autoFilled: boolean; // 是否自动填充了实例
}

interface MigrationTask {
  srcId: string;
  targetUser: string;
  targetInstanceName: string;
  targetInstanceId: string;
  status: TaskStatus;
  message?: string;
}

interface MigrationHistory {
  batchId: string;
  scene: string;
  createdAt: string;
  total: number;
  success: number;
  failed: number;
}

// ==================== Mock Data ====================

const MOCK_USERS = [
  { id: "u1", name: "张三", email: "zhangsan@company.com" },
  { id: "u2", name: "李四", email: "lisi@company.com" },
  { id: "u3", name: "王五", email: "wangwu@company.com" },
  { id: "u4", name: "赵六", email: "zhaoliu@company.com" },
  { id: "u5", name: "Alice", email: "alice@acompany.com" },
  { id: "u6", name: "Bob", email: "bob@acompany.com" },
];

const MOCK_USER_INSTANCES: Record<string, { id: string; instanceId: string; name: string }[]> = {
  u1: [{ id: "c1", instanceId: "ins-g83c6wvc", name: "张三的助手" }, { id: "c2", instanceId: "ins-h92d7xwe", name: "张三的写作助手" }],
  u2: [{ id: "c3", instanceId: "ins-j14e8yvf", name: "李四的研究助手" }], // 只有1台→自动填充
  u3: [{ id: "c4", instanceId: "ins-k25f9zwg", name: "王五的代码助手" }], // 只有1台→自动填充
  u4: [{ id: "c6", instanceId: "ins-m47h1byi", name: "赵六的数据助手" }], // 只有1台→自动填充
  u5: [{ id: "c7", instanceId: "ins-n58i2czj", name: "Alice的助手" }],
  u6: [{ id: "c8", instanceId: "ins-o69j3dak", name: "Bob工作助手" }],
};

const MOCK_SRC_INSTANCES = [
  { instanceId: "lhins-abc001", name: "Agent-张三", region: "ap-guangzhou", status: "running" },
  { instanceId: "lhins-abc002", name: "Agent-李四", region: "ap-guangzhou", status: "running" },
  { instanceId: "lhins-abc003", name: "Agent-王五", region: "ap-beijing", status: "running" },
  { instanceId: "lhins-abc004", name: "Agent-赵六", region: "ap-beijing", status: "shutdown" },
  { instanceId: "lhins-abc005", name: "Agent-Alice", region: "ap-guangzhou", status: "running" },
  { instanceId: "lhins-abc006", name: "Agent-Bob", region: "ap-shanghai", status: "running" },
];

const MOCK_HISTORY: MigrationHistory[] = [
  { batchId: "20260328-f7a2b1", scene: "Lighthouse", createdAt: "2026-03-28 14:30", total: 20, success: 18, failed: 2 },
  { batchId: "20260325-c3d4e5", scene: "EKS 容器", createdAt: "2026-03-25 09:15", total: 5, success: 5, failed: 0 },
];

const SOURCE_TYPES: { value: SourceType; label: string; enabled: boolean }[] = [
  { value: "lighthouse", label: "腾讯云 Lighthouse", enabled: true },
  { value: "eks", label: "EKS / K8s 容器", enabled: true },
  { value: "idc", label: "自建服务器 / IDC", enabled: true },
  { value: "aliyun", label: "阿里云 ECS", enabled: false },
  { value: "volcengine", label: "火山引擎 ECS", enabled: false },
  { value: "huaweicloud", label: "华为云 ECS", enabled: false },
];

// ==================== Helpers ====================

function SecretInput({ value, onChange, placeholder }: { value: string; onChange: (v: string) => void; placeholder: string }) {
  const [vis, setVis] = useState(false);
  return (
    <div className="relative">
      <Input type={vis ? "text" : "password"} value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} className="pr-8 font-mono text-xs" />
      <button type="button" onClick={() => setVis(!vis)} className="absolute right-2 top-1/2 -translate-y-1/2 text-[#A3A3A3] hover:text-[#737373]">
        {vis ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
      </button>
    </div>
  );
}

function CopyBtn({ text }: { text: string }) {
  const [ok, setOk] = useState(false);
  return (
    <button onClick={() => { navigator.clipboard.writeText(text); setOk(true); toast.success("已复制"); setTimeout(() => setOk(false), 2000); }}
      className="p-1 rounded hover:bg-[#f5f5f5] text-[#A3A3A3] hover:text-[#737373] transition-colors">
      {ok ? <CheckCircle2 className="w-3.5 h-3.5 text-green-500" /> : <Copy className="w-3.5 h-3.5" />}
    </button>
  );
}

/** 查找用户，如果该用户只有1个实例，自动返回实例信息 */
function resolveUser(userName: string): { userId: string; autoInstance?: { id: string; instanceId: string; name: string } } | null {
  const user = MOCK_USERS.find((u) => u.name === userName || u.email === userName || u.id === userName);
  if (!user) return null;
  const instances = MOCK_USER_INSTANCES[user.id] || [];
  return { userId: user.id, autoInstance: instances.length === 1 ? instances[0] : undefined };
}

function makeCsvBlob(header: string, rows: string[]): Blob {
  return new Blob(["\uFEFF" + header + "\n" + rows.join("\n")], { type: "text/csv;charset=utf-8" });
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url; a.download = filename; a.click();
  URL.revokeObjectURL(url);
}

// ==================== Main Component ====================

export default function AgentMigration() {
  const [, setLocation] = useLocation();
  // --- Source config ---
  const [sourceType, setSourceType] = useState<SourceType>("lighthouse");
  const [srcAk, setSrcAk] = useState("");
  const [srcSk, setSrcSk] = useState("");
  const [srcRegion, setSrcRegion] = useState("all");
  const [srcLoading, setSrcLoading] = useState(false);
  const [srcLoaded, setSrcLoaded] = useState(false);
  const [stopGateway, setStopGateway] = useState(true);
  const [containerDirs, setContainerDirs] = useState("/root/.agent");

  // EKS/IDC: COS upload state
  const [cosLoaded, setCosLoaded] = useState(false);
  const [cosFiles, setCosFiles] = useState<string[]>([]);

  // --- Mapping ---
  const [editMode, setEditMode] = useState<EditMode>("table");
  const [rows, setRows] = useState<MappingRow[]>([]);
  const [srcSearch, setSrcSearch] = useState("");

  // --- Execution ---
  const [batchId] = useState(() => {
    const d = new Date();
    return `${d.getFullYear()}${String(d.getMonth() + 1).padStart(2, "0")}${String(d.getDate()).padStart(2, "0")}-${Math.random().toString(36).substring(2, 8)}`;
  });
  const cosBucket = "clawpro-migrate-1302061491";
  const [tasks, setTasks] = useState<MigrationTask[]>([]);
  const [isRunning, setIsRunning] = useState(false);
  const [progress, setProgress] = useState(0);
  const [jumpboxId, setJumpboxId] = useState("");
  const [jumpboxReady, setJumpboxReady] = useState(false);
  const [showCommand, setShowCommand] = useState(false);

  // --- History ---
  const [showHistory, setShowHistory] = useState(false);

  // --- Derived ---
  const isAuto = sourceType === "lighthouse";
  const isManual = sourceType === "eks" || sourceType === "idc";
  const validRows = rows.filter((r) => r.srcId && r.targetUser && r.targetInstance);
  const isDone = !isRunning && tasks.length > 0;

  // ==================== Source Instance Fetch (Lighthouse) ====================

  const fetchInstances = () => {
    if (!srcAk || !srcSk) { toast.error("请先输入源端密钥"); return; }
    setSrcLoading(true);
    setTimeout(() => {
      const newRows: MappingRow[] = MOCK_SRC_INSTANCES
        .filter((i) => i.status === "running")
        .map((i, idx) => ({ id: String(idx), srcId: i.instanceId, targetUser: "", targetInstance: "", targetInstanceId: "", autoFilled: false }));
      setRows(newRows);
      setSrcLoaded(true);
      setSrcLoading(false);
      toast.success(`获取到 ${newRows.length} 个可迁移实例`);
    }, 1200);
  };

  // ==================== COS File Detection (EKS/IDC) ====================

  const fetchCosFiles = () => {
    setTimeout(() => {
      const files = ["pod-frontend-001.tgz", "pod-backend-002.tgz", "server-10.0.1.50.tgz"];
      setCosFiles(files);
      setCosLoaded(true);
      const newRows: MappingRow[] = files.map((f, i) => ({
        id: String(i), srcId: f.replace(".tgz", ""), targetUser: "", targetInstance: "", targetInstanceId: "", autoFilled: false,
      }));
      setRows(newRows);
      toast.success(`检测到 ${files.length} 个数据包`);
    }, 800);
  };

  // ==================== Mapping Row Ops ====================

  const updateRow = useCallback((id: string, field: "srcId" | "targetUser" | "targetInstance", value: string) => {
    setRows((prev) => prev.map((r) => {
      if (r.id !== id) return r;
      const updated = { ...r, [field]: value, autoFilled: false };
      if (field === "targetUser") {
        // 自动填充：如果该用户只有一个实例
        const instances = MOCK_USER_INSTANCES[value] || [];
        if (instances.length === 1) {
          updated.targetInstance = instances[0].id;
          updated.targetInstanceId = instances[0].instanceId;
          updated.autoFilled = true;
        } else {
          updated.targetInstance = "";
          updated.targetInstanceId = "";
        }
      }
      if (field === "targetInstance") {
        const instances = MOCK_USER_INSTANCES[updated.targetUser] || [];
        const inst = instances.find((i) => i.id === value);
        updated.targetInstanceId = inst?.instanceId || "";
      }
      return updated;
    }));
  }, []);

  const addRow = () => setRows((prev) => [...prev, { id: String(Date.now()), srcId: "", targetUser: "", targetInstance: "", targetInstanceId: "", autoFilled: false }]);
  const removeRow = (id: string) => { if (rows.length > 1) setRows((prev) => prev.filter((r) => r.id !== id)); };

  // ==================== CSV Import/Export ====================

  const downloadSrcCsv = () => {
    const header = "源端标识,ClawPro用户名,Agent名";
    const data = rows.map((r) => `${r.srcId},,`);
    downloadBlob(makeCsvBlob(header, data), `migration_${batchId}_template.csv`);
    toast.success("模板已下载，请补充第 2、3 列后上传");
  };

  const downloadCosCsv = () => {
    const header = "COS文件名,ClawPro用户名,Agent名";
    const data = cosFiles.map((f) => `${f},,`);
    downloadBlob(makeCsvBlob(header, data), `cos_files_${batchId}_template.csv`);
    toast.success("COS 文件列表模板已下载");
  };

  const handleCsvUpload = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      const text = ev.target?.result as string;
      const lines = text.split("\n").filter((l) => l.trim());
      if (lines.length <= 1) { toast.error("CSV 无数据行"); return; }
      let imported = 0;
      let autoCount = 0;
      const newRows: MappingRow[] = [];
      for (let i = 1; i < lines.length; i++) {
        const cols = lines[i].split(",").map((c) => c.trim().replace(/^"|"$/g, ""));
        if (!cols[0]) continue;
        const srcId = cols[0];
        const userName = cols[1] || "";
        const instName = cols[2] || "";

        let targetUser = "";
        let targetInstance = "";
        let targetInstanceId = "";
        let autoFilled = false;

        if (userName) {
          const resolved = resolveUser(userName);
          if (resolved) {
            targetUser = resolved.userId;
            if (instName) {
              // 精确匹配实例名
              const instances = MOCK_USER_INSTANCES[resolved.userId] || [];
              const inst = instances.find((ins) => ins.name === instName || ins.instanceId === instName);
              if (inst) { targetInstance = inst.id; targetInstanceId = inst.instanceId; }
            } else if (resolved.autoInstance) {
              // 实例名为空，但该用户只有1台，自动填充
              targetInstance = resolved.autoInstance.id;
              targetInstanceId = resolved.autoInstance.instanceId;
              autoFilled = true;
              autoCount++;
            }
          }
        }
        newRows.push({ id: String(Date.now() + i), srcId, targetUser, targetInstance, targetInstanceId, autoFilled });
        imported++;
      }
      setRows(newRows);
      setEditMode("table");
      toast.success(`导入 ${imported} 条${autoCount > 0 ? `，${autoCount} 条已自动匹配单实例用户` : ""}`);
    };
    reader.readAsText(file);
    e.target.value = "";
  }, []);

  // ==================== Upload Script (EKS/IDC) ====================

  const uploadScript = useMemo(() => {
    const dirs = containerDirs || "/root/.agent";
    const stop = stopGateway ? 'echo "停止 Gateway..."\nagent gateway stop 2>/dev/null || true\n' : "";
    const packCmd = sourceType === "eks"
      ? `tar -czf /tmp/openclaw-state.tgz ${dirs.split(",").map((d) => d.trim()).join(" ")}`
      : `tar -czf /tmp/openclaw-state.tgz -C /root .agent`;
    return `#!/bin/bash
# 批次: ${batchId} | 桶: ${cosBucket}
COS_BUCKET="${cosBucket}"
COS_REGION="ap-guangzhou"
BATCH="${batchId}"
TMP_AK="<平台自动填充>"
TMP_SK="<平台自动填充>"
TMP_TOKEN="<平台自动填充>"

SRC_ID=\${POD_NAME:-\$(hostname)}
echo "源端: \${SRC_ID}"
${stop}echo "打包中..."
${packCmd}
echo "上传到 COS..."
curl -X PUT -H "x-cos-security-token: \${TMP_TOKEN}" \\
  --upload-file /tmp/openclaw-state.tgz \\
  "https://\${COS_BUCKET}.cos.\${COS_REGION}.myqcloud.com/\${BATCH}/\${SRC_ID}.tgz"
echo "✅ 完成: cos://\${COS_BUCKET}/\${BATCH}/\${SRC_ID}.tgz"
rm -f /tmp/openclaw-state.tgz`;
  }, [batchId, cosBucket, containerDirs, stopGateway, sourceType]);

  // ==================== Migration Command ====================

  const migrationCommand = useMemo(() => {
    const count = validRows.length;
    if (isAuto) {
      return `python3 /opt/migrate/lighthouse2clawpro.py \\
  --src-secret-id "${srcAk.substring(0, 8)}****" --src-secret-key "****" \\
  --cos-bucket "${cosBucket}" --cos-region "ap-guangzhou" \\
  --batch-id "${batchId}" ${stopGateway ? "--stop-gateway " : ""}\\
  --manifest /opt/migrate/manifest.json
# ${count} 台: 打包→上传COS→TAT恢复→restart`;
    }
    return `python3 /opt/migrate/restore_from_cos.py \\
  --cos-bucket "${cosBucket}" --cos-region "ap-guangzhou" \\
  --batch-id "${batchId}" \\
  --manifest /opt/migrate/manifest.json
# ${count} 台: COS下载→解压→${sourceType === "eks" ? "修复路径→" : ""}restart`;
  }, [validRows.length, isAuto, srcAk, cosBucket, batchId, stopGateway, sourceType]);

  // ==================== Execute ====================

  const startMigration = () => {
    if (validRows.length === 0) { toast.error("请至少配置一条有效映射"); return; }
    const taskList: MigrationTask[] = validRows.map((r) => {
      const user = MOCK_USERS.find((u) => u.id === r.targetUser);
      const insts = user ? MOCK_USER_INSTANCES[user.id] || [] : [];
      const inst = insts.find((i) => i.id === r.targetInstance);
      return { srcId: r.srcId, targetUser: user?.name || "", targetInstanceName: inst?.name || "", targetInstanceId: r.targetInstanceId, status: "pending" as TaskStatus };
    });
    setTasks(taskList);
    setIsRunning(true);
    setJumpboxReady(false);
    setJumpboxId("[迁移操作台] " + batchId);
    setTimeout(() => setJumpboxReady(true), 2500);

    let done = 0;
    const total = taskList.length;
    const iv = setInterval(() => {
      done++;
      setProgress(Math.round((done / total) * 100));
      setTasks((prev) => prev.map((t, i) => {
        if (i < done - 1) return t;
        if (i === done - 1) { const fail = Math.random() < 0.08; return { ...t, status: fail ? "failed" : "success", message: fail ? "TAT 超时" : undefined }; }
        if (i === done) return { ...t, status: isAuto ? "uploading" : "restoring" };
        return t;
      }));
      if (done >= total) { clearInterval(iv); setIsRunning(false); toast.success("迁移完成"); }
    }, 1500);
  };

  const resetAll = () => {
    setRows([]); setTasks([]); setProgress(0); setIsRunning(false);
    setJumpboxReady(false); setSrcLoaded(false); setCosLoaded(false); setCosFiles([]);
  };

  // ==================== Render ====================

  const filteredRows = rows.filter((r) => !srcSearch || r.srcId.includes(srcSearch));
  const successCount = tasks.filter((t) => t.status === "success").length;
  const failedCount = tasks.filter((t) => t.status === "failed").length;

  return (
    <div className="page-enter">
      {/* Header */}
      <div className="mb-7">
        <button onClick={() => setLocation("/admin/openclaw-monitor")} className="flex items-center gap-1 text-xs text-[#525252] hover:text-[#355EF1] mb-2 transition-colors">
          <ArrowLeft className="w-3.5 h-3.5" /> 返回 Agent 列表
        </button>
        <AdminPageHeader
          title="智能体迁移"
          description="批量迁移其他平台的 Agent 智能体到 ClawPro，数据通过 COS 安全中转。单台迁移可在用户端 Agent 详情页操作。"
          className="mb-0"
        />
      </div>

      {/* ==================== Section 1: 源端配置 ==================== */}
      <Card className="p-5 mb-5 rounded-[4px]">
        <div className="flex items-center gap-2.5 mb-4">
          <span className="font-din text-[18px] font-bold text-[#1447E6] leading-none tabular-nums shrink-0">01</span>
          <h2 className="text-[14px] font-semibold text-[#0A0A0A]">源端环境</h2>
          <span className="text-[12px] text-[#A3A3A3]">— 选择源端类型，配置密钥和迁移选项</span>
        </div>

        <div className="flex items-end gap-4">
          {/* Source Type */}
          <div className="w-48 flex-shrink-0">
            <Label className="text-xs text-[#737373] mb-1 block">源端类型</Label>
            <Select value={sourceType} onValueChange={(v) => { setSourceType(v as SourceType); setRows([]); setSrcLoaded(false); setCosLoaded(false); }}>
              <SelectTrigger className="h-9 text-[14px]"><SelectValue /></SelectTrigger>
              <SelectContent>
                {SOURCE_TYPES.map((s) => (
                  <SelectItem key={s.value} value={s.value} disabled={!s.enabled}>
                    {s.label}{!s.enabled && <span className="text-[#A3A3A3] ml-1 text-xs">即将支持</span>}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Lighthouse: AK/SK + Fetch */}
          {isAuto && (
            <>
              <div className="flex-1 min-w-0">
                <Label className="text-xs text-[#737373] mb-1 block">
                  SecretId
                  <a href="https://console.cloud.tencent.com/cam/capi" target="_blank" rel="noopener noreferrer"
                    className="ml-1.5 text-[#355EF1] hover:text-[#355EF1] hover:underline inline-flex items-center gap-0.5">
                    获取密钥 <ExternalLink className="w-2.5 h-2.5" />
                  </a>
                </Label>
                <SecretInput value={srcAk} onChange={setSrcAk} placeholder="源端 SecretId" />
              </div>
              <div className="flex-1 min-w-0">
                <Label className="text-xs text-[#737373] mb-1 block">SecretKey</Label>
                <SecretInput value={srcSk} onChange={setSrcSk} placeholder="源端 SecretKey" />
              </div>
              <div className="flex-shrink-0">
                <Button variant="dialog-confirm" onClick={fetchInstances} disabled={!srcAk || !srcSk || srcLoading}>
                  {srcLoading ? <Loader2 className="w-3.5 h-3.5 animate-spin mr-1" /> : <Search className="w-3.5 h-3.5 mr-1" />}
                  获取实例
                </Button>
              </div>
            </>
          )}

          {/* EKS: Container dirs */}
          {sourceType === "eks" && (
            <div className="flex-1">
              <Label className="text-xs text-[#737373] mb-1 block">待迁移目录（逗号分隔）</Label>
              <Input value={containerDirs} onChange={(e) => setContainerDirs(e.target.value)} placeholder="/root/.agent" className="text-xs font-mono" />
            </div>
          )}
        </div>

        {/* Options row */}
        <div className="flex items-center gap-6 mt-4 pt-4 border-t border-gray-200">
          <div className="flex items-center gap-2">
            <Switch checked={stopGateway} onCheckedChange={setStopGateway} className="scale-90" />
            <span className="text-xs text-[#737373]">迁移前停止 Gateway</span>
            <TooltipProvider><Tooltip><TooltipTrigger><HelpCircle className="w-3 h-3 text-[#A3A3A3]" /></TooltipTrigger>
              <TooltipContent><p className="text-xs">建议开启，保证数据一致性</p></TooltipContent></Tooltip></TooltipProvider>
          </div>
          <div className="text-xs text-[#A3A3A3] leading-relaxed">
            <Database className="w-3 h-3 inline mr-1 -mt-0.5" />
            系统自动创建临时 COS 桶 <code className="bg-[#f5f5f5] px-1 rounded font-mono">{cosBucket}/{batchId}/</code> 作为数据中转，迁移数据保留 24 小时后自动清理。COS
            按量计费，<a href="https://cloud.tencent.com/document/product/436/53482#.E5.AD.98.E5.82.A8.E5.AE.B9.E9.87.8F.E5.AE.9A.E4.BB.B7" target="_blank" rel="noopener noreferrer"
              className="text-[#355EF1] hover:text-[#355EF1] hover:underline inline-flex items-center gap-0.5">
              费用详见腾讯云定价文档 <ExternalLink className="w-2.5 h-2.5" />
            </a>
          </div>
        </div>

        {/* Manual scene: Upload Script */}
        {isManual && (
          <Collapsible className="mt-4 pt-4 border-t border-gray-200">
            <CollapsibleTrigger className="flex items-center gap-1.5 text-xs font-medium text-[#355EF1] hover:text-[#355EF1]">
              <Terminal className="w-3.5 h-3.5" />
              查看上传脚本（请在源端执行）
              <ChevronRight className="w-3 h-3" />
            </CollapsibleTrigger>
            <CollapsibleContent className="mt-2">
              {sourceType === "eks" && (
                <p className="text-xs text-amber-600 mb-2 flex items-center gap-1">
                  <CircleAlert className="w-3 h-3" />
                  通过 <code className="bg-amber-50 px-1 rounded">kubectl exec &lt;pod&gt; -- bash /tmp/migrate.sh</code> 执行
                </p>
              )}
              <div className="relative bg-gray-900 rounded-[4px] p-3 overflow-x-auto">
                <div className="absolute top-2 right-2"><CopyBtn text={uploadScript} /></div>
                <pre className="text-xs text-green-400 font-mono whitespace-pre leading-relaxed">{uploadScript}</pre>
              </div>
              <div className="flex items-center gap-3 mt-2">
                <span className="text-xs text-[#A3A3A3] flex items-center gap-1"><Clock className="w-3 h-3" /> 临时密钥有效 4 小时</span>
              </div>
            </CollapsibleContent>
          </Collapsible>
        )}

        {/* Manual scene: COS detection */}
        {isManual && (
          <div className="mt-4 pt-4 border-t border-gray-200 flex items-center gap-3">
            <Button variant="outline" size="sm" onClick={fetchCosFiles} className="h-7 text-xs">
              <RefreshCw className="w-3 h-3 mr-1" /> {cosLoaded ? "刷新 COS 文件" : "检测已上传文件"}
            </Button>
            {cosLoaded && (
              <>
                <span className="text-xs text-green-600 flex items-center gap-1"><CheckCircle2 className="w-3 h-3" /> 检测到 {cosFiles.length} 个文件</span>
                <Button variant="ghost" size="sm" onClick={downloadCosCsv} className="h-7 text-xs text-[#355EF1]">
                  <Download className="w-3 h-3 mr-1" /> 下载文件列表 CSV
                </Button>
              </>
            )}
          </div>
        )}
      </Card>

      {/* ==================== Section 2: 迁移映射 ==================== */}
      <Card className="p-5 mb-5 rounded-[4px]">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2.5">
            <span className="font-din text-[18px] font-bold text-[#1447E6] leading-none tabular-nums shrink-0">02</span>
            <h2 className="text-[14px] font-semibold text-[#0A0A0A]">迁移映射</h2>
            <span className="text-[12px] text-[#A3A3A3]">— 建立源端与 ClawPro 用户/实例的对应关系</span>
            <span className="text-[12px] text-[#A3A3A3] ml-auto mr-2">{validRows.length} / {rows.length} 有效</span>
          </div>
          <Tabs value={editMode} onValueChange={(v) => setEditMode(v as EditMode)}>
            <TabsList className="h-8 bg-[#F5F5F5] p-1 rounded-[4px]">
              <TabsTrigger value="table" className="text-xs px-3 h-6 rounded-[3px] data-[state=active]:bg-white data-[state=active]:text-[#0A0A0A] data-[state=active]:font-medium data-[state=active]:shadow-[var(--shadow-segment)]">在线编辑</TabsTrigger>
              <TabsTrigger value="csv" className="text-xs px-3 h-6 rounded-[3px] data-[state=active]:bg-white data-[state=active]:text-[#0A0A0A] data-[state=active]:font-medium data-[state=active]:shadow-[var(--shadow-segment)]">CSV 批量导入</TabsTrigger>
            </TabsList>
          </Tabs>
        </div>

        {editMode === "csv" ? (
          /* ===== CSV Tab ===== */
          <div className="space-y-3">
            <div className="bg-[#F5F5F5] border border-gray-200 rounded-[4px] p-3">
              <p className="text-xs text-[#737373] leading-relaxed">
                <FileSpreadsheet className="w-3.5 h-3.5 inline mr-1 -mt-0.5" />
                <strong>批量迁移推荐使用 CSV：</strong>下载映射模板 → 在 Excel 中填写源端标识与目标用户/实例的对应关系 → 上传 CSV 自动导入。
              </p>
            </div>
            <div className="flex items-center gap-3 flex-wrap">
              {(isAuto && srcLoaded) && (
                <Button variant="outline" size="sm" onClick={downloadSrcCsv} className="h-7 text-xs">
                  <Download className="w-3 h-3 mr-1" /> 下载源端实例模板
                </Button>
              )}
              {(isManual && cosLoaded) && (
                <Button variant="outline" size="sm" onClick={downloadCosCsv} className="h-7 text-xs">
                  <Download className="w-3 h-3 mr-1" /> 下载 COS 文件列表模板
                </Button>
              )}
              <Button variant="outline" size="sm" onClick={() => {
                const header = "ClawPro用户名,用户邮箱,Agent数,实例列表(实例名:实例ID)";
                const data = MOCK_USERS.map((u) => {
                  const insts = MOCK_USER_INSTANCES[u.id] || [];
                  const instList = insts.map((i) => `${i.name}:${i.instanceId}`).join("|");
                  return `${u.name},${u.email},${insts.length},${instList}`;
                });
                downloadBlob(makeCsvBlob(header, data), `clawpro_users_instances.csv`);
                toast.success("用户与实例列表已下载，供填写映射时参考");
              }} className="h-7 text-xs">
                <Download className="w-3 h-3 mr-1" /> 下载 ClawPro 用户与实例列表
              </Button>
              <label className="cursor-pointer">
                <input type="file" accept=".csv" className="hidden" onChange={handleCsvUpload} />
                <span className="inline-flex items-center gap-1 px-3 py-1.5 rounded-[4px] border border-gray-200 bg-white text-xs font-medium text-[#355EF1] hover:border-blue-500 cursor-pointer transition-colors">
                  <Upload className="w-3 h-3" /> 上传 CSV 文件
                </span>
              </label>
            </div>
            <div className="text-xs text-[#A3A3A3]">
              CSV 格式：<code className="bg-[#f5f5f5] px-1 rounded">源端标识,ClawPro用户名,Agent名</code>（第一行为表头）
            </div>
            {rows.length > 0 && (
              <div className="bg-green-50 border border-green-200 rounded-[4px] p-2.5">
                <p className="text-xs text-green-700 flex items-center gap-1">
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  已导入 {rows.length} 条映射，其中 {validRows.length} 条有效。可切换到「在线编辑」查看和调整。
                </p>
              </div>
            )}
          </div>
        ) : (
          /* ===== Table Tab ===== */
          <>
            {rows.length === 0 ? (
              <div className="text-center py-8 text-sm text-[#A3A3A3]">
                {isAuto ? "请先获取源端实例列表" : "请先检测 COS 文件或手动添加行"}
              </div>
            ) : (
              <>
                <div className="flex items-center gap-2 mb-2">
                  {rows.length > 10 && (
                    <Input value={srcSearch} onChange={(e) => setSrcSearch(e.target.value)} placeholder="搜索源端标识..." className="h-7 text-xs w-48" />
                  )}
                </div>
                <div className="border border-gray-200 rounded-[4px] overflow-hidden max-h-[380px] overflow-y-auto">
                  <Table>
                    <TableHeader>
                      <TableRow className="sticky top-0 z-10">
                        <TableHead className="w-[200px]">
                          {isAuto ? "源端实例 ID" : "COS 文件 / 源端标识"}
                        </TableHead>
                        <TableHead className="w-[30px]"></TableHead>
                        <TableHead>ClawPro 用户</TableHead>
                        <TableHead>Agent 实例</TableHead>
                        <TableHead className="w-[40px]"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {filteredRows.map((row) => {
                        const userInst = row.targetUser ? MOCK_USER_INSTANCES[row.targetUser] || [] : [];
                        const singleInst = userInst.length === 1;
                        return (
                          <TableRow key={row.id} className={row.targetInstance ? "bg-green-50/30" : ""}>
                            <TableCell className="py-1.5">
                              {isAuto ? (
                                <span className="text-xs font-mono">{row.srcId}</span>
                              ) : (
                                <Input value={row.srcId} onChange={(e) => updateRow(row.id, "srcId", e.target.value)}
                                  placeholder="文件名或标识" className="h-7 text-xs font-mono" />
                              )}
                            </TableCell>
                            <TableCell className="py-1.5"><ArrowRight className="w-3 h-3 text-[#A3A3A3]" /></TableCell>
                            <TableCell className="py-1.5">
                              <Select value={row.targetUser} onValueChange={(v) => updateRow(row.id, "targetUser", v)}>
                                <SelectTrigger className="h-7 text-xs"><SelectValue placeholder="选择用户" /></SelectTrigger>
                                <SelectContent>
                                  {MOCK_USERS.map((u) => {
                                    const instCount = (MOCK_USER_INSTANCES[u.id] || []).length;
                                    return (
                                    <SelectItem key={u.id} value={u.id}>
                                      <span className="flex items-center gap-1">
                                        {u.name}
                                        <span className="text-[#A3A3A3] text-xs">（{instCount}台）</span>
                                      </span>
                                    </SelectItem>
                                    );
                                  })}
                                </SelectContent>
                              </Select>
                            </TableCell>
                            <TableCell className="py-1.5">
                              {singleInst && row.autoFilled ? (
                                <div className="flex items-center gap-1 text-xs text-green-600 px-2 h-7">
                                  <CheckCircle2 className="w-3 h-3" />
                                  {userInst[0].name}
                                  <span className="text-[#A3A3A3] font-mono">{userInst[0].instanceId}</span>
                                </div>
                              ) : (
                                <Select value={row.targetInstance} disabled={!row.targetUser}
                                  onValueChange={(v) => updateRow(row.id, "targetInstance", v)}>
                                  <SelectTrigger className="h-7 text-xs"><SelectValue placeholder={row.targetUser ? "选择实例" : "—"} /></SelectTrigger>
                                  <SelectContent>
                                    {userInst.map((inst) => (
                                      <SelectItem key={inst.id} value={inst.id}>{inst.name} <span className="text-[#A3A3A3] text-xs">{inst.instanceId}</span></SelectItem>
                                    ))}
                                  </SelectContent>
                                </Select>
                              )}
                            </TableCell>
                            <TableCell className="py-1.5">
                              {!isAuto && (
                                <button onClick={() => removeRow(row.id)} className="p-1 text-[#A3A3A3] hover:text-red-500">
                                  <Trash2 className="w-3 h-3" />
                                </button>
                              )}
                            </TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </div>
                {!isAuto && (
                  <Button variant="ghost" size="sm" onClick={addRow} className="mt-2 h-6 text-xs text-[#737373]">
                    <Plus className="w-3 h-3 mr-1" /> 添加行
                  </Button>
                )}
              </>
            )}
          </>
        )}
      </Card>

      {/* ==================== Section 3: 执行 ==================== */}
      <Card className="p-5 mb-5 rounded-[4px]">
        <div className="flex items-center gap-2.5 mb-4">
          <span className="font-din text-[18px] font-bold text-[#1447E6] leading-none tabular-nums shrink-0">03</span>
          <h2 className="text-[14px] font-semibold text-[#0A0A0A]">执行迁移</h2>
          <span className="text-[12px] text-[#A3A3A3]">— 确认无误后一键执行，平台自动创建操作台并运行迁移脚本</span>
        </div>

        {tasks.length === 0 ? (
          <>
            {/* Pre-execution info */}
            <div className="flex items-center gap-4 mb-4 text-xs text-[#737373]">
              <span className="flex items-center gap-1"><Database className="w-3 h-3" /> COS 中转桶: <code className="bg-[#f5f5f5] px-1 rounded font-mono">{cosBucket}/{batchId}/</code></span>
              <span className="flex items-center gap-1"><Clock className="w-3 h-3" /> 临时数据保留 24 小时后自动清理</span>
            </div>

            {/* Command preview */}
            <Collapsible open={showCommand} onOpenChange={setShowCommand}>
              <CollapsibleTrigger className="flex items-center gap-1.5 text-xs font-medium text-[#355EF1] hover:text-[#355EF1] mb-2">
                <Terminal className="w-3.5 h-3.5" />
                {showCommand ? "收起" : "查看"}迁移命令
                {showCommand ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
              </CollapsibleTrigger>
              <CollapsibleContent>
                <div className="relative bg-gray-900 rounded-[4px] p-3 overflow-x-auto mb-3">
                  <div className="absolute top-2 right-2"><CopyBtn text={migrationCommand} /></div>
                  <pre className="text-xs text-green-400 font-mono whitespace-pre leading-relaxed">{migrationCommand}</pre>
                </div>
              </CollapsibleContent>
            </Collapsible>

            <div className="flex items-center gap-3">
              <Button variant="dialog-confirm" onClick={startMigration} disabled={validRows.length === 0} className="px-6">
                <Play className="w-4 h-4 mr-1.5" />
                开始迁移（{validRows.length} 台）
              </Button>
              <span className="text-xs text-[#A3A3A3]">将创建临时实例「<strong className="text-[#737373]">[迁移操作台] {batchId}</strong>」，完成后请在 Agent 列表删除</span>
            </div>
          </>
        ) : (
          <>
            {/* Jumpbox + Progress */}
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                {jumpboxReady
                  ? <CheckCircle2 className="w-4 h-4 text-green-500" />
                  : <Loader2 className="w-4 h-4 text-[#355EF1] animate-spin" />}
                <span className="text-xs text-[#737373]">操作台 {jumpboxReady ? jumpboxId : "创建中..."}</span>
                {jumpboxReady && (
                  <button onClick={() => toast.info(`跳转到终端: ${jumpboxId}`)} className="text-xs text-[#355EF1] hover:underline flex items-center gap-0.5">
                    <Terminal className="w-3 h-3" /> 终端 <ExternalLink className="w-2.5 h-2.5" />
                  </button>
                )}
              </div>
              <span className="text-xs text-[#737373]">
                {successCount + failedCount}/{tasks.length}
                {failedCount > 0 && <span className="text-red-500 ml-1">({failedCount} 失败)</span>}
              </span>
            </div>

            <Progress value={progress} className="h-1.5 mb-3" />

            {/* Task list */}
            <div className="space-y-1 max-h-[260px] overflow-y-auto">
              {tasks.map((t, i) => (
                <div key={i} className={`flex items-center justify-between px-3 py-2 rounded-[4px] text-xs ${
                  t.status === "success" ? "bg-green-50" :
                  t.status === "failed" ? "bg-red-50" :
                  t.status === "pending" ? "bg-[#fafafa]" : "bg-[#eff4ff]"
                }`}>
                  <div className="flex items-center gap-2 flex-1 min-w-0">
                    {t.status === "success" && <CheckCircle2 className="w-3.5 h-3.5 text-green-500" />}
                    {t.status === "failed" && <XCircle className="w-3.5 h-3.5 text-red-500" />}
                    {t.status === "pending" && <Clock className="w-3.5 h-3.5 text-[#A3A3A3]" />}
                    {(t.status === "uploading" || t.status === "restoring") && <Loader2 className="w-3.5 h-3.5 text-[#355EF1] animate-spin" />}
                    <span className="font-mono truncate">{t.srcId}</span>
                    <ArrowRight className="w-3 h-3 text-[#A3A3A3] flex-shrink-0" />
                    <span className="truncate">{t.targetUser}/{t.targetInstanceName}</span>
                    {t.message && <span className="text-red-500 ml-1">({t.message})</span>}
                  </div>
                  <span className={`flex-shrink-0 ${
                    t.status === "success" ? "text-green-600" :
                    t.status === "failed" ? "text-red-600" :
                    t.status === "pending" ? "text-[#A3A3A3]" : "text-[#355EF1]"
                  }`}>
                    {{ success: "✓", failed: "✗", pending: "等待", uploading: "上传中", restoring: "恢复中" }[t.status]}
                  </span>
                </div>
              ))}
            </div>

            {/* Done */}
            {isDone && (
              <div className="flex items-center justify-between mt-3 pt-3 border-t border-gray-200">
                <div className="text-xs text-[#737373]">
                  <Database className="w-3 h-3 inline mr-1" />
                  COS 临时数据 24h 后自动清理 | 请前往列表验证并删除临时操作台
                </div>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" onClick={() => toast.info("跳转到列表")} className="h-7 text-xs">
                    <ExternalLink className="w-3 h-3 mr-1" /> Agent 列表
                  </Button>
                  <Button size="sm" onClick={resetAll} className="h-7 text-xs">
                    <RefreshCw className="w-3 h-3 mr-1" /> 新建迁移
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </Card>

      {/* ==================== History ==================== */}
      <Collapsible open={showHistory} onOpenChange={setShowHistory}>
        <CollapsibleTrigger className="flex items-center gap-1.5 text-xs font-medium text-[#737373] hover:text-[#334155] mb-2">
          <History className="w-3.5 h-3.5" /> 迁移历史
          {showHistory ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
        </CollapsibleTrigger>
        <CollapsibleContent>
          <div className="border border-gray-200 rounded-[4px] overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>批次</TableHead>
                  <TableHead>场景</TableHead>
                  <TableHead>时间</TableHead>
                  <TableHead>结果</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {MOCK_HISTORY.map((h) => (
                  <TableRow key={h.batchId}>
                    <TableCell className="font-mono">{h.batchId}</TableCell>
                    <TableCell>{h.scene}</TableCell>
                    <TableCell className="text-[#737373]">{h.createdAt}</TableCell>
                    <TableCell>
                      <span className="text-green-600">{h.success}✓</span>
                      {h.failed > 0 && <span className="text-red-500 ml-1">{h.failed}✗</span>}
                      <span className="text-[#A3A3A3] ml-1">/ {h.total}</span>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}
