/**
 * SettingEditDialog - 团队设定 新建 / 编辑 弹窗
 *
 * 形态（与企业资产库其它库的编辑弹窗一致，复用 Dialog）：
 *  - 基本信息：名称 + 关联项目（关键信息）+ 一句话描述 + 版本
 *  - 正文：Markdown 编辑器 / 预览（单栏切换，类似 Write / Preview）
 *  - 校验：实时给出「精炼 / 误写细则 / 误写步骤」提示，不阻断保存
 */
import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { AlertTriangle, Info, FileText, Eye } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';
import { SearchableSelect } from '@/components/ui/select';
import { BodyMedium, MetaText } from '@/components/ui/Typography';
import { renderMarkdown } from '@/lib/markdownRenderer';
import {
  type TeamSetting,
  TEAM_SETTING_TEMPLATE,
  RECOMMENDED_MAX_BODY_LENGTH,
  MOCK_PROJECTS,
  getProjectName,
  stripFrontmatter,
  composeContent,
  lintTeamSetting,
} from './settingsLibraryData';

interface SettingEditDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 传入则为编辑，否则为新建 */
  setting?: TeamSetting | null;
  /** 现有设定列表，用于校验「一个项目只维护一份设定」 */
  existingSettings: TeamSetting[];
  onSave: (data: {
    id?: string;
    name: string;
    description: string;
    projectId: string;
    version: string;
    content: string;
  }) => void;
}

function todayStr(): string {
  return new Date().toISOString().slice(0, 10);
}

export default function SettingEditDialog({
  open,
  onOpenChange,
  setting,
  existingSettings,
  onSave,
}: SettingEditDialogProps) {
  const isEdit = !!setting;

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [projectId, setProjectId] = useState('');
  const [body, setBody] = useState('');
  const [view, setView] = useState<'edit' | 'preview'>('edit');

  // 自动计算版本：新建 1.0，编辑时在当前版本基础上 +0.1
  const computedVersion = useMemo(() => {
    if (setting) {
      const current = parseFloat(setting.version) || 0;
      return (current + 0.1).toFixed(1);
    }
    return '1.0';
  }, [setting]);

  // 打开时初始化
  useEffect(() => {
    if (!open) return;
    if (setting) {
      setName(setting.name);
      setDescription(setting.description);
      setProjectId(setting.projectId);
      setBody(stripFrontmatter(setting.content).trim());
    } else {
      setName('');
      setDescription('');
      setProjectId('');
      setBody(stripFrontmatter(TEAM_SETTING_TEMPLATE).trim());
    }
    setView('edit');
  }, [open, setting]);

  // 同一项目下已存在的另一份设定（用于唯一性提示）
  const projectConflict = useMemo(() => {
    if (!projectId) return null;
    return existingSettings.find((s) => s.projectId === projectId && s.id !== setting?.id) || null;
  }, [projectId, existingSettings, setting?.id]);

  const fullContent = useMemo(() => {
    const fm: Record<string, string> = {};
    if (projectId) fm.project = getProjectName(projectId);
    fm.version = computedVersion;
    fm.updated_at = todayStr();
    return composeContent(fm, body);
  }, [projectId, computedVersion, body]);

  const issues = useMemo(() => lintTeamSetting(fullContent), [fullContent]);
  const bodyLength = useMemo(() => stripFrontmatter(fullContent).trim().length, [fullContent]);
  const previewHtml = useMemo(() => renderMarkdown(fullContent), [fullContent]);

  const projectOptions = useMemo(
    () =>
      MOCK_PROJECTS.map((p) => ({
        value: p.id,
        label: p.name,
        searchText: `${p.name} ${p.repo}`,
      })),
    [],
  );

  const handleSave = () => {
    if (!name.trim()) {
      toast.error('请填写设定名称');
      return;
    }
    if (!projectId) {
      toast.error('请选择关联项目');
      return;
    }
    onSave({
      id: setting?.id,
      name: name.trim(),
      description: description.trim(),
      projectId,
      version: computedVersion,
      content: fullContent,
    });
    onOpenChange(false);
  };

  const handleResetTemplate = () => {
    setBody(stripFrontmatter(TEAM_SETTING_TEMPLATE).trim());
    toast.success('已填入标准模板');
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[1040px] max-h-[88vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>{isEdit ? '编辑团队设定' : '新建团队设定'}</DialogTitle>
          <DialogDescription>
            团队设定是 Agent 每次会话自动加载的团队基础上下文。请保持全局、精炼，只确立工作基准，不写编码细则或操作步骤。
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 min-h-0 overflow-y-auto space-y-4 pr-1">
          {/* 基本信息 */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <MetaText as="label" tone="secondary">设定名称 <span className="text-[var(--text-danger)]">*</span></MetaText>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="如：核心交易平台 · 团队设定" />
            </div>
            <div className="space-y-1.5">
              <MetaText as="label" tone="secondary">关联项目 <span className="text-[var(--text-danger)]">*</span></MetaText>
              <SearchableSelect
                options={projectOptions}
                value={projectId}
                onChange={setProjectId}
                placeholder="选择该设定归属的项目"
                searchPlaceholder="搜索项目名称 / 仓库"
                countTemplate="共 {count} 个项目"
                triggerClassName="w-full"
              />
            </div>
          </div>

          {/* 项目唯一性提示 */}
          {projectConflict && (
            <div className="flex items-start gap-2 rounded-[4px] border border-amber-200 bg-amber-50 px-3 py-2">
              <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-600" />
              <MetaText as="p" tone="secondary" className="leading-relaxed">
                该项目已绑定设定「<BodyMedium as="span" tone="primary">{projectConflict.name}</BodyMedium>」。
                一个项目通常只维护一份团队设定，继续保存会导致同一项目存在多份设定，建议改为编辑原设定。
              </MetaText>
            </div>
          )}

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <MetaText as="label" tone="secondary">一句话描述</MetaText>
              <Input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="用于列表展示的简要说明" />
            </div>
            <div className="space-y-1.5">
              <MetaText as="label" tone="secondary">版本号（自动升级）</MetaText>
              <div className="flex h-9 items-center rounded-[4px] border border-[var(--cp-border)] bg-[var(--bg-grey-hover-subtle)] px-3">
                <BodyMedium as="span" tone="primary" className="tabular-nums">{computedVersion}</BodyMedium>
                {setting && (
                  <MetaText as="span" tone="weak" className="ml-2">（{setting.version} → {computedVersion}）</MetaText>
                )}
                {!setting && (
                  <MetaText as="span" tone="weak" className="ml-2">初始版本</MetaText>
                )}
              </div>
            </div>
          </div>

          {/* 编辑器 / 预览 切换 */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <BodyMedium as="span" tone="primary">设定正文（Markdown）</BodyMedium>
              <Button variant="link" size="sm" onClick={handleResetTemplate}>填入标准模板</Button>
            </div>
            <div className="flex items-center gap-3">
              <MetaText as="span" tone={bodyLength > RECOMMENDED_MAX_BODY_LENGTH ? 'danger' : 'weak'} className="tabular-nums">
                {bodyLength} / {RECOMMENDED_MAX_BODY_LENGTH} 字
              </MetaText>
              <SegmentGroup>
                <SegmentOption active={view === 'edit'} onClick={() => setView('edit')} title="编辑">
                  <FileText className="w-3.5 h-3.5 mr-1" />
                  编辑
                </SegmentOption>
                <SegmentOption active={view === 'preview'} onClick={() => setView('preview')} title="预览">
                  <Eye className="w-3.5 h-3.5 mr-1" />
                  预览
                </SegmentOption>
              </SegmentGroup>
            </div>
          </div>

          {/* 正文：编辑 / 预览 单栏切换 */}
          <div style={{ minHeight: 340 }}>
            {view === 'edit' ? (
              <Textarea
                value={body}
                onChange={(e) => setBody(e.target.value)}
                placeholder="在此编写团队设定正文（建议分区：团队身份 / 核心价值观与工作准则 / 协作与沟通风格 / 全局质量底线 / 引用入口）"
                className="h-full min-h-[340px] font-mono text-[13px] leading-relaxed resize-none"
              />
            ) : (
              <div className="h-full min-h-[340px] overflow-y-auto rounded-[4px] border border-[var(--cp-border)] bg-white p-4">
                <div
                  className="markdown-body prose prose-sm max-w-none"
                  // 预览内容由本地受控 markdown-it 渲染，输入来自管理员自己编辑的设定文档；
                  // markdownRenderer 已对代码块做转义，此处为 demo 预览用途。
                  dangerouslySetInnerHTML={{ __html: previewHtml }}
                />
              </div>
            )}
          </div>

          {/* 规范校验提示 */}
          {issues.length > 0 && (
            <div className="space-y-2">
              {issues.map((issue, idx) => {
                const isWarning = issue.level === 'warning';
                return (
                  <div
                    key={idx}
                    className={`flex items-start gap-2 rounded-[4px] border px-3 py-2 ${
                      isWarning
                        ? 'border-amber-200 bg-amber-50'
                        : 'border-[#C7D7FE] bg-[#E8ECFE]'
                    }`}
                  >
                    {isWarning ? (
                      <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-600" />
                    ) : (
                      <Info className="mt-0.5 size-4 shrink-0 text-[#1447E6]" />
                    )}
                    <div className="space-y-0.5">
                      <BodyMedium as="p" tone="primary">{issue.title}</BodyMedium>
                      <MetaText as="p" tone="secondary" className="leading-relaxed">{issue.detail}</MetaText>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="claw-outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button variant="dialog-confirm" onClick={handleSave}>
            {isEdit ? '保存' : '创建'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
