/**
 * ServerManagement - 管控端云服务器管理页
 * 包含：镜像管理 Tab + 安全组管理 Tab
 */
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell,
} from "@/components/ui/table";
import { StatusTag } from "@/components/ui/status-tag";
import { toast } from "sonner";
import { Plus, Trash2, Pencil, Download, Server, Shield } from "lucide-react";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { MetaMedium } from "@/components/ui/Typography";

// Mock data
const MOCK_IMAGES = [
  { id: "img-001", name: "openclaw-base-v2.1", status: "available", disk: "系统盘 150GiB", os: "CentOS 7.9 64位", createTime: "2025-12-01", active: true },
  { id: "img-002", name: "openclaw-base-v2.0", status: "available", disk: "系统盘 100GiB", os: "CentOS 7.9 64位", createTime: "2025-09-15", active: false },
  { id: "img-003", name: "openclaw-dev-v1.5", status: "creating", disk: "系统盘 200GiB", os: "Ubuntu 22.04 64位", createTime: "2026-03-01", active: false },
];

const DEFAULT_INBOUND = [
  { id: "1", source: "0.0.0.0/0", protocol: "ICMP", port: "ALL", policy: "允许", remark: "放通Ping服务" },
  { id: "2", source: "::/0", protocol: "ICMPv6", port: "ALL", policy: "允许", remark: "放通Ping服务" },
  { id: "3", source: "0.0.0.0/0", protocol: "TCP", port: "22", policy: "允许", remark: "放通Linux SSH登录" },
  { id: "4", source: "::/0", protocol: "TCP", port: "22", policy: "允许", remark: "放通Linux SSH登录" },
  { id: "5", source: "0.0.0.0/0", protocol: "TCP", port: "3389", policy: "允许", remark: "放通Windows远程登录" },
  { id: "6", source: "::/0", protocol: "TCP", port: "3389", policy: "允许", remark: "放通Windows远程登录" },
  { id: "7", source: "10.0.0.0/8", protocol: "ALL", port: "ALL", policy: "允许", remark: "放通内网（云私有网络）" },
  { id: "8", source: "172.16.0.0/12", protocol: "ALL", port: "ALL", policy: "允许", remark: "放通内网（云私有网络）" },
  { id: "9", source: "192.168.0.0/16", protocol: "ALL", port: "ALL", policy: "允许", remark: "放通内网（云私有网络）" },
  { id: "10", source: "0.0.0.0/0", protocol: "TCP", port: "80", policy: "允许", remark: "Web服务HTTP(80)，如Apache、Nginx" },
  { id: "11", source: "0.0.0.0/0", protocol: "TCP", port: "443", policy: "允许", remark: "Web服务HTTPS(443)，如Apache、Nginx" },
  { id: "12", source: "0.0.0.0/0", protocol: "TCP", port: "18789", policy: "允许", remark: "" },
  { id: "13", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "" },
];

const DEFAULT_OUTBOUND = [
  { id: "1", source: "-", protocol: "ALL", port: "ALL", policy: "允许", remark: "" },
  { id: "2", source: "0.0.0.0/0", protocol: "ALL", port: "ALL", policy: "拒绝", remark: "" },
];

export default function ServerManagement() {
  const [images, setImages] = useState(MOCK_IMAGES);
  const [inboundRules, setInboundRules] = useState(DEFAULT_INBOUND);
  const [outboundRules, setOutboundRules] = useState(DEFAULT_OUTBOUND);
  const [showImportDialog, setShowImportDialog] = useState(false);
  const [showAddRuleDialog, setShowAddRuleDialog] = useState(false);
  const [ruleType, setRuleType] = useState<"inbound" | "outbound">("inbound");
  const [editRule, setEditRule] = useState<any>(null);
  const [newRule, setNewRule] = useState({ source: "", protocol: "TCP", port: "", policy: "允许", remark: "" });

  const handleSaveRule = () => {
    if (!newRule.source && ruleType === "inbound") { toast.error("请填写来源"); return; }
    const rule = { ...newRule, id: String(Date.now()) };
    if (editRule) {
      if (ruleType === "inbound") setInboundRules(inboundRules.map((r) => r.id === editRule.id ? { ...rule, id: editRule.id } : r));
      else setOutboundRules(outboundRules.map((r) => r.id === editRule.id ? { ...rule, id: editRule.id } : r));
      toast.success("规则已更新");
    } else {
      if (ruleType === "inbound") setInboundRules([...inboundRules, rule]);
      else setOutboundRules([...outboundRules, rule]);
      toast.success("规则已添加");
    }
    setShowAddRuleDialog(false);
    setEditRule(null);
    setNewRule({ source: "", protocol: "TCP", port: "", policy: "允许", remark: "" });
  };

  const openAddRule = (type: "inbound" | "outbound") => {
    setRuleType(type);
    setEditRule(null);
    setNewRule({ source: "", protocol: "TCP", port: "", policy: "允许", remark: "" });
    setShowAddRuleDialog(true);
  };

  const openEditRule = (rule: any, type: "inbound" | "outbound") => {
    setRuleType(type);
    setEditRule(rule);
    setNewRule({ source: rule.source || rule.target || "", protocol: rule.protocol, port: rule.port, policy: rule.policy, remark: rule.remark });
    setShowAddRuleDialog(true);
  };

  return (
    <>
      <div className="page-enter">
        <div className="mb-8">
          <AdminPageHeader title="云服务器管理" description="管理企业版 Agent 所使用的云服务器镜像和安全组策略。" />
        </div>

        <Tabs defaultValue="images">
          <TabsList className="mb-6">
            <TabsTrigger value="images" className="flex items-center gap-1.5">
              <Server className="w-3.5 h-3.5" />
              镜像管理
            </TabsTrigger>
            <TabsTrigger value="security" className="flex items-center gap-1.5">
              <Shield className="w-3.5 h-3.5" />
              安全组管理
            </TabsTrigger>
          </TabsList>

          {/* 镜像管理 */}
          <TabsContent value="images">
            <div className="bg-white rounded-[4px] border border-gray-200 overflow-hidden"
             >
              <div className="flex items-center justify-between px-6 py-5 border-b border-gray-50">
                <h2 className="font-semibold text-[#0A0A0A]">镜像列表</h2>
                <Button size="sm" onClick={() => setShowImportDialog(true)}
                 >
                  <Download className="w-3.5 h-3.5 mr-1.5" />
                  导入镜像
                </Button>
              </div>
              <Table variant="white">
                <TableHeader>
                  <TableRow>
                    <TableHead>镜像 ID / 名称</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>硬盘</TableHead>
                    <TableHead>操作系统</TableHead>
                    <TableHead>创建时间</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {images.map((img) => (
                    <TableRow key={img.id}>
                      <TableCell>
                        <p className="text-sm font-medium text-[#0A0A0A]">{img.name}</p>
                        <p className="text-xs text-[#A3A3A3] font-mono">{img.id}</p>
                      </TableCell>
                      <TableCell>
                        {img.status === "available" ? (
                          <StatusTag mode="fill" variant="green">可用</StatusTag>
                        ) : (
                          <StatusTag mode="fill" variant="gray">创建中</StatusTag>
                        )}
                      </TableCell>
                      <TableCell className="text-[#737373]">{img.disk}</TableCell>
                      <TableCell className="text-[#737373]">{img.os}</TableCell>
                      <TableCell className="text-[#737373]">{img.createTime}</TableCell>
                      <TableActionCell actionsClassName="gap-3">
                        <div className="flex items-center gap-2">
                          <span className="text-xs text-[#A3A3A3]">生效</span>
                          <Switch
                            checked={img.active}
                            onCheckedChange={(v) => {
                              setImages(images.map((i) => ({ ...i, active: i.id === img.id ? v : false })));
                              if (v) toast.success(`镜像 ${img.name} 已设为生效`);
                            }}
                          />
                        </div>
                        <button
                          onClick={() => { setImages(images.filter((i) => i.id !== img.id)); toast.success("镜像已删除"); }}
                          className="text-[#A3A3A3] hover:text-red-500 transition-colors">
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </TableActionCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </TabsContent>

          {/* 安全组管理 */}
          <TabsContent value="security">
            <Tabs defaultValue="inbound">
              <TabsList className="mb-4">
                <TabsTrigger value="inbound">入站规则</TabsTrigger>
                <TabsTrigger value="outbound">出站规则</TabsTrigger>
              </TabsList>

              <TabsContent value="inbound">
                <div className="bg-white rounded-[4px] border border-gray-200 overflow-hidden"
                 >
                  <div className="flex items-center justify-between px-6 py-4 border-b border-gray-50">
                    <span className="text-sm font-medium text-[#334155]">入站规则</span>
                    <Button size="sm" variant="outline" onClick={() => openAddRule("inbound")}>
                      <Plus className="w-3.5 h-3.5 mr-1" />
                      添加规则
                    </Button>
                  </div>
                  <Table density="compact">
                    <TableHeader>
                      <TableRow>
                        <TableHead>来源</TableHead>
                        <TableHead>协议端口</TableHead>
                        <TableHead>端口</TableHead>
                        <TableHead>策略</TableHead>
                        <TableHead>备注</TableHead>
                        <TableHead>操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {inboundRules.map((rule) => (
                        <TableRow key={rule.id}>
                          <TableCell className="text-[#0A0A0A] font-mono">{rule.source}</TableCell>
                          <TableCell className="text-[#737373]">{rule.protocol}</TableCell>
                          <TableCell className="text-[#737373]">{rule.port}</TableCell>
                          <TableCell>
                            <StatusTag mode="text" variant={rule.policy === "允许" ? "green" : "red"}>
                              {rule.policy}
                            </StatusTag>
                          </TableCell>
                          <TableCell className="text-[#A3A3A3]">{rule.remark || "-"}</TableCell>
                          <TableActionCell>
                            <Button variant="link" onClick={() => openEditRule(rule, "inbound")}>编辑</Button>
                            <Button variant="link" onClick={() => { setInboundRules(inboundRules.filter((r) => r.id !== rule.id)); toast.success("规则已删除"); }}>
                              删除
                            </Button>
                          </TableActionCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </TabsContent>

              <TabsContent value="outbound">
                <div className="bg-white rounded-[4px] border border-gray-200 overflow-hidden"
                 >
                  <div className="flex items-center justify-between px-6 py-4 border-b border-gray-50">
                    <span className="text-sm font-medium text-[#334155]">出站规则</span>
                    <Button size="sm" variant="outline" onClick={() => openAddRule("outbound")}>
                      <Plus className="w-3.5 h-3.5 mr-1" />
                      添加规则
                    </Button>
                  </div>
                  <Table density="compact">
                    <TableHeader>
                      <TableRow>
                        <TableHead>目标</TableHead>
                        <TableHead>协议端口</TableHead>
                        <TableHead>端口</TableHead>
                        <TableHead>策略</TableHead>
                        <TableHead>备注</TableHead>
                        <TableHead>操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {outboundRules.map((rule) => (
                        <TableRow key={rule.id}>
                          <TableCell className="text-[#0A0A0A] font-mono">{rule.source}</TableCell>
                          <TableCell className="text-[#737373]">{rule.protocol}</TableCell>
                          <TableCell className="text-[#737373]">{rule.port}</TableCell>
                          <TableCell>
                            <StatusTag mode="text" variant={rule.policy === "允许" ? "green" : "red"}>
                              {rule.policy}
                            </StatusTag>
                          </TableCell>
                          <TableCell className="text-[#A3A3A3]">{rule.remark || "-"}</TableCell>
                          <TableActionCell>
                            <Button variant="link" onClick={() => openEditRule(rule, "outbound")}>编辑</Button>
                            <Button variant="link" onClick={() => { setOutboundRules(outboundRules.filter((r) => r.id !== rule.id)); toast.success("规则已删除"); }}>
                              删除
                            </Button>
                          </TableActionCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </TabsContent>
            </Tabs>
          </TabsContent>
        </Tabs>
      </div>

      {/* Import Image Dialog */}
      <Dialog open={showImportDialog} onOpenChange={setShowImportDialog}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader><DialogTitle>导入镜像</DialogTitle></DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <MetaMedium as="label" tone="secondary">镜像名称</MetaMedium>
              <Input placeholder="请输入镜像名称" className="bg-white" />
            </div>
            <div className="space-y-2">
              <MetaMedium as="label" tone="secondary">镜像 ID</MetaMedium>
              <Input placeholder="请输入镜像 ID" className="bg-white font-mono" />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowImportDialog(false)}>取消</Button>
            <Button variant="dialog-confirm" onClick={() => { setShowImportDialog(false); toast.success("镜像导入任务已提交"); }}
             >
              确认导入
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add/Edit Rule Dialog */}
      <Dialog open={showAddRuleDialog} onOpenChange={setShowAddRuleDialog}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>{editRule ? "编辑规则" : `添加${ruleType === "inbound" ? "入站" : "出站"}规则`}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <MetaMedium as="label" tone="secondary">{ruleType === "inbound" ? "来源" : "目标"}</MetaMedium>
              <Input
                placeholder={ruleType === "inbound" ? "例如 0.0.0.0/0" : "例如 0.0.0.0/0"}
                value={newRule.source}
                onChange={(e) => setNewRule({ ...newRule, source: e.target.value })}
                className="bg-white"
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary">协议</MetaMedium>
                <Select value={newRule.protocol} onValueChange={(v) => setNewRule({ ...newRule, protocol: v })}>
                  <SelectTrigger className="bg-white"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {["TCP", "UDP", "ICMP", "ICMPv6", "ALL"].map((p) => (
                      <SelectItem key={p} value={p}>{p}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <MetaMedium as="label" tone="secondary">端口</MetaMedium>
                <Input
                  placeholder="例如 80 或 ALL"
                  value={newRule.port}
                  onChange={(e) => setNewRule({ ...newRule, port: e.target.value })}
                  className="bg-white"
                />
              </div>
            </div>
            <div className="space-y-2">
              <MetaMedium as="label" tone="secondary">策略</MetaMedium>
              <Select value={newRule.policy} onValueChange={(v) => setNewRule({ ...newRule, policy: v })}>
                <SelectTrigger className="bg-white"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="允许">允许</SelectItem>
                  <SelectItem value="拒绝">拒绝</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <MetaMedium as="label" tone="secondary">备注（可选）</MetaMedium>
              <Input
                placeholder="规则备注"
                value={newRule.remark}
                onChange={(e) => setNewRule({ ...newRule, remark: e.target.value })}
                className="bg-white"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddRuleDialog(false)}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleSaveRule}>
              {editRule ? "保存" : "确认添加"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
