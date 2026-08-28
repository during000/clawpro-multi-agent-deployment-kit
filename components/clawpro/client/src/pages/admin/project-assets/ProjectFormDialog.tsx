import { useEffect, useMemo, useState } from 'react';
import { ShieldCheck } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { MetaText } from '@/components/ui/Typography';
import type { UserGroup } from '../MemberManagement/types';

export interface ProjectFormValues {
  name: string;
  description: string;
  goal: string;
  allowMemberEdit: boolean;
}

export interface ProjectFormTarget extends ProjectFormValues {
  id: string;
}

interface ProjectFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: 'create' | 'edit';
  target?: ProjectFormTarget | null;
  projects: UserGroup[];
  onConfirm: (values: ProjectFormValues) => void;
}

export default function ProjectFormDialog({
  open,
  onOpenChange,
  mode,
  target,
  projects,
  onConfirm,
}: ProjectFormDialogProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [goal, setGoal] = useState('');
  const [allowMemberEdit, setAllowMemberEdit] = useState(true);

  useEffect(() => {
    if (!open) return;
    setName(mode === 'edit' ? target?.name ?? '' : '');
    setDescription(mode === 'edit' ? target?.description ?? '' : '');
    setGoal(mode === 'edit' ? target?.goal ?? '' : '');
    setAllowMemberEdit(mode === 'edit' ? target?.allowMemberEdit ?? true : true);
  }, [mode, open, target]);

  const duplicate = useMemo(
    () => projects.some((project) => (
      project.source === 'project'
      && project.name.trim() === name.trim()
      && project.id !== target?.id
    )),
    [name, projects, target?.id],
  );
  const valid = name.trim().length > 0 && !duplicate;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{mode === 'create' ? '新建项目' : '编辑项目'}</DialogTitle>
          <DialogDescription>
            {mode === 'create'
              ? '填写项目基本信息并配置成员编辑项目的权限。'
              : '修改项目基本信息与成员编辑项目的权限。'}
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="px-6 space-y-3 max-h-[60vh] overflow-y-auto">
          <div className="space-y-2">
            <Label htmlFor="admin-project-name">项目名称</Label>
            <Input
              id="admin-project-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="输入项目名称"
              maxLength={40}
              aria-invalid={duplicate || undefined}
              autoFocus
            />
            {duplicate && <MetaText tone="danger">项目名称已存在</MetaText>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="admin-project-description">项目描述</Label>
            <Textarea
              id="admin-project-description"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              placeholder="说明项目背景与协作范围"
              className="min-h-16"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="admin-project-goal">项目目标</Label>
            <Textarea
              id="admin-project-goal"
              value={goal}
              onChange={(event) => setGoal(event.target.value)}
              placeholder="填写项目希望达成的目标"
              className="min-h-16"
            />
          </div>

          <div className="flex items-start justify-between gap-3 rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 py-2.5">
            <div className="min-w-0">
              <div className="flex items-center gap-1.5 text-sm font-medium text-[var(--text-body)]">
                <ShieldCheck className="h-4 w-4 text-[var(--cp-brand-blue)]" />
                允许编辑项目
              </div>
              <MetaText className="mt-0.5 block">
                开启后项目成员可修改项目信息及资产，关闭后项目保持只读。
              </MetaText>
            </div>
            <Switch
              checked={allowMemberEdit}
              onCheckedChange={setAllowMemberEdit}
              aria-label="允许项目成员编辑项目"
            />
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="claw-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="claw-primary"
            disabled={!valid}
            onClick={() => {
              if (!valid) return;
              onConfirm({
                name: name.trim(),
                description: description.trim(),
                goal: goal.trim(),
                allowMemberEdit,
              });
            }}
          >
            {mode === 'create' ? '创建' : '保存'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
