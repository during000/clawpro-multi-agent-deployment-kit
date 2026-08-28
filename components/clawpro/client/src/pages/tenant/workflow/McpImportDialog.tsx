/**
 * McpImportDialog —— agent 经 MCP 回传工作流的指引与接收入口（P3）
 *
 * 体现 ClawPro 自身使用也是 AI-Native：提供可复制的「让 agent 画工作流」指引/schema，
 * 并接收粘贴的工作流 JSON，经 mcpWorkflowSchema 严格安全校验后落库为流水线模板。
 */
import { useEffect, useState } from 'react';
import { toast } from 'sonner';
import { Copy, Check, Bot, ShieldCheck } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogBody,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { MetaText } from '@/components/ui/Typography';
import {
  parseMcpWorkflow,
  toTemplateNodes,
  MCP_WORKFLOW_EXAMPLE,
} from './mcpWorkflowSchema';
import type { TenantPipelineTemplateNode } from '../tenantProjectStore';

export interface McpImportDialogProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onReceive: (
    nodes: TenantPipelineTemplateNode[],
    meta: { name: string; description: string },
  ) => void;
}

const MCP_GUIDE = `工作流上传格式：JSON（唯一受信格式）。你可以在本地 agent 里按下方规范生成工作流 JSON，再通过 MCP 工具 receive_workflow 回传，或直接粘贴到下方文本框导入。

【顶层结构】
{
  "name":        必填，字符串，工作流名称（1–200 字符）
  "description": 选填，字符串，工作流说明（≤200 字符）
  "nodes":       必填，数组，1–50 个节点，构成一张 DAG
}

【每个 node 结构】
{
  "id":        必填，节点唯一标识；仅允许字母/数字/-/_，≤64 字符
  "title":     必填，节点标题（1–200 字符）
  "agentRole": 选填，执行该节点的 agent 角色（如产品/前端/后端/测试），默认「成员」
  "dependsOn": 选填，字符串数组，上游节点 id；全部上游确认后本节点才启动
}

【硬性约束】（不满足将拒绝导入，不会落库）
- 节点全部派发给 agent 执行，不支持派发给人；
- 节点数 1–50，id 不可重复；
- dependsOn 只能引用已存在的 id，不能自依赖；
- 整图不得存在循环依赖（必须是 DAG）；
- 禁止 __proto__ / prototype / constructor 等危险键。`;

export default function McpImportDialog({
  open,
  onOpenChange,
  onReceive,
}: McpImportDialogProps) {
  const [text, setText] = useState('');
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (open) {
      setText('');
      setCopied(false);
    }
  }, [open]);

  const copyExample = () => {
    navigator.clipboard?.writeText(MCP_WORKFLOW_EXAMPLE).then(
      () => {
        setCopied(true);
        setTimeout(() => setCopied(false), 1600);
      },
      () => toast.error('复制失败，请手动选择复制'),
    );
  };

  const handleImport = () => {
    const result = parseMcpWorkflow(text);
    if (!result.ok) {
      toast.error(result.error);
      return;
    }
    onReceive(toTemplateNodes(result.data), {
      name: result.data.name,
      description: result.data.description,
    });
    toast.success(`已接收工作流「${result.data.name}」并落库为流水线模板`);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-[640px] flex flex-col"
        style={{ maxHeight: 'min(90vh, 760px)' }}
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Bot className="w-4 h-4 text-[var(--cp-brand-blue)]" />
            agent 经 MCP 回传工作流
          </DialogTitle>
          <DialogDescription>
            让本地 agent 按 JSON 规范「画」好工作流并回传，ClawPro校验后自动落库为流水线模板——节点全部由 agent 执行。
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="px-6 space-y-4">
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <Label>回传指引 & schema 示例</Label>
              <Button variant="tenant-outline" size="sm" onClick={copyExample}>
                {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
                {copied ? '已复制' : '复制示例'}
              </Button>
            </div>
            <pre className="rounded-[4px] bg-[var(--color-gray-100)] p-3 text-[11px] leading-relaxed text-[var(--text-secondary)] whitespace-pre-wrap">
              {MCP_GUIDE}
            </pre>
            <pre className="mt-2 max-h-[200px] overflow-auto rounded-[4px] bg-[#1D2129] p-3 text-[11px] leading-relaxed text-[#E5E7EB] font-mono">
              {MCP_WORKFLOW_EXAMPLE}
            </pre>
          </div>

          <div className="space-y-1.5">
            <Label>粘贴 agent 回传的工作流 JSON</Label>
            <Textarea
              value={text}
              onChange={(e) => setText(e.target.value)}
              rows={6}
              placeholder="在此粘贴工作流 JSON…"
              className="font-mono text-xs"
            />
            <MetaText tone="weak" className="flex items-center gap-1">
              <ShieldCheck className="w-3.5 h-3.5" />
              导入前会严格校验字段、类型、依赖合法性与环，校验失败不会落库。
            </MetaText>
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button variant="tenant-primary" disabled={!text.trim()} onClick={handleImport}>
            校验并落库为模板
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
