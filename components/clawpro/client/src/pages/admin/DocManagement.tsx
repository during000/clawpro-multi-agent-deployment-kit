/**
 * DocManagement - 管控端文档管理页
 */
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { SurfaceCard } from "@/components/ui/Surface";
import { StatusTag } from "@/components/ui/status-tag";
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell,
} from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import { toast } from "sonner";
import { Plus, Trash2, FileText, Upload, ExternalLink } from "lucide-react";
import { AdminPageHeader } from "@/components/ui/admin-page-header";

const DEFAULT_DOCS = [
  { id: "1", title: "Agent 概念介绍", addTime: "2025-01-01", addBy: "系统", visible: true, isDefault: true },
  { id: "2", title: "ClawPro平台的功能与特色", addTime: "2025-01-01", addBy: "系统", visible: true, isDefault: true },
  { id: "3", title: "部署 Agent 指引", addTime: "2025-01-01", addBy: "系统", visible: true, isDefault: true },
  { id: "4", title: "Agent 进阶玩法", addTime: "2025-01-01", addBy: "系统", visible: true, isDefault: true },
];

export default function DocManagement() {
  const [docs, setDocs] = useState(DEFAULT_DOCS);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [newDoc, setNewDoc] = useState({ title: "", file: null as File | null });

  const handleAdd = () => {
    if (!newDoc.title.trim()) { toast.error("请输入文档标题"); return; }
    if (!newDoc.file) { toast.error("请上传 Markdown 文件"); return; }
    setDocs([...docs, {
      id: String(Date.now()), title: newDoc.title,
      addTime: new Date().toISOString().slice(0, 10),
      addBy: "alice@acompany.com", visible: true, isDefault: false,
    }]);
    setShowAddDialog(false);
    setNewDoc({ title: "", file: null });
    toast.success("文档已添加");
  };

  return (
    <>
      <div className="page-enter">
        <AdminPageHeader
          title="帮助文档"
          description="此处配置的文档将展示在企业用户看到的「帮助文档」中。默认包含 4 篇通用文档，管理员可自行添加或删除文档。"
          actions={
            <Button onClick={() => setShowAddDialog(true)}>
              <Plus className="w-4 h-4 mr-1.5" />
              添加文档
            </Button>
          }
          className="mb-8"
        />

        <SurfaceCard className="overflow-hidden">
          <Table variant="white">
            <TableHeader>
              <TableRow>
                <TableHead>文档标题</TableHead>
                <TableHead>添加时间</TableHead>
                <TableHead>添加人</TableHead>
                <TableHead>展示状态</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {docs.map((doc) => (
                <TableRow key={doc.id}>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <FileText className="w-4 h-4 text-[#355EF1] flex-shrink-0" />
                      <button
                        onClick={() => toast.info("跳转到用户端文档页面")}
                        className="text-sm font-medium text-[#0A0A0A] hover:text-[#355EF1] transition-colors flex items-center gap-1 underline underline-offset-2 decoration-transparent hover:decoration-blue-600"
                      >
                        {doc.title}
                        <ExternalLink className="w-3 h-3 text-[#A3A3A3] group-hover:text-[#355EF1]" />
                      </button>
                      {doc.isDefault && (
                        <Badge variant="outline" className="text-xs border-blue-200 text-[#355EF1] bg-blue-50">默认</Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="text-[#737373]">{doc.addTime}</TableCell>
                  <TableCell className="text-[#737373]">{doc.addBy}</TableCell>
                  <TableCell>
                    {doc.visible ? (
                      <StatusTag mode="fill" variant="green">展示中</StatusTag>
                    ) : (
                      <StatusTag mode="fill" variant="gray">已隐藏</StatusTag>
                    )}
                  </TableCell>
                  <TableActionCell>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-[#A3A3A3]">用户可见</span>
                      <Switch
                        checked={doc.visible}
                        onCheckedChange={(v) => {
                          setDocs(docs.map((d) => d.id === doc.id ? { ...d, visible: v } : d));
                          toast.success(v ? "文档已展示" : "文档已隐藏");
                        }}
                      />
                    </div>
                    <button
                      onClick={() => { setDocs(docs.filter((d) => d.id !== doc.id)); toast.success("文档已删除"); }}
                      className="text-[#A3A3A3] hover:text-red-500 transition-colors">
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </TableActionCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <div className="px-6 py-3 border-t border-gray-50 text-xs text-[#A3A3A3]">
            共 {docs.length} 篇文档
          </div>
        </SurfaceCard>
      </div>

      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>添加文档</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label>文档标题</Label>
              <Input
                placeholder="请输入文档标题"
                value={newDoc.title}
                onChange={(e) => setNewDoc({ ...newDoc, title: e.target.value })}
                className="bg-white"
              />
            </div>
            <div className="space-y-2">
              <Label>上传文档</Label>
              <p className="text-xs text-[#A3A3A3]">仅支持上传 .md 格式的 Markdown 文件</p>
              <label className="flex flex-col items-center justify-center w-full h-28 border-2 border-dashed border-border rounded-[4px] cursor-pointer hover:border-[var(--cp-brand-blue)] hover:bg-[var(--accent)] transition-colors">
                {newDoc.file ? (
                  <div className="text-center">
                    <FileText className="w-6 h-6 text-[#355EF1] mx-auto mb-1" />
                    <span className="text-sm text-[#355EF1]">{newDoc.file.name}</span>
                  </div>
                ) : (
                  <>
                    <Upload className="w-6 h-6 text-[#A3A3A3] mb-2" />
                    <span className="text-sm text-[#737373]">点击上传 .md 文件</span>
                  </>
                )}
                <input type="file" accept=".md" className="hidden"
                  onChange={(e) => { if (e.target.files?.[0]) setNewDoc({ ...newDoc, file: e.target.files[0] }); }} />
              </label>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAddDialog(false)}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleAdd}>确认添加</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
