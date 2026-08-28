import { Field, FieldGroup } from '@/components/ui/field';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { SurfaceInner } from '@/components/ui/Surface';
import { Textarea } from '@/components/ui/textarea';
import { MetaMedium, MetaText } from '@/components/ui/Typography';
import { ChevronRight } from 'lucide-react';

export interface HookFormValue {
  event: string;
  command: string;
}

export const EMPTY_HOOK_FORM: HookFormValue = {
  event: '',
  command: '',
};

const HOOK_EVENT_OPTIONS = [
  {
    value: 'SessionStart',
    label: 'Agent 启动或新会话开始时（SessionStart）',
    description: '用户启动 Agent 或开始一个新会话时触发。',
  },
  {
    value: 'UserPromptSubmit',
    label: '用户发送消息后（UserPromptSubmit）',
    description: '用户每次发送新消息时触发。',
  },
  {
    value: 'PreToolUse',
    label: 'Agent 调用任意工具前（PreToolUse）',
    description: 'Agent 每次调用终端、文件、MCP 等任意工具前触发。',
  },
  {
    value: 'PostToolUse',
    label: 'Agent 调用任意工具后（PostToolUse）',
    description: 'Agent 每次调用终端、文件、MCP 等任意工具完成后触发。',
  },
  {
    value: 'Stop',
    label: 'Agent 每轮回复结束时（Stop）',
    description: 'Agent 每完成一轮回复时触发；同一会话可能触发多次。',
  },
] as const;

export const getHookFormError = (value: HookFormValue) => {
  if (!value.event.trim()) return '请选择 Hook 触发时机';
  if (!value.command.trim()) return '请填写触发后执行的本地命令';
  return '';
};

interface HookManifestMetadata {
  id: string;
  description: string;
}

const yamlString = (value: string) => JSON.stringify(value);

const inferHookTools = (command: string) => {
  if (/\bCODEBUDDY_PROJECT_DIR\b/.test(command)) {
    return ['codebuddy'];
  }
  return [];
};

export const buildHookManifestYaml = (
  value: HookFormValue,
  metadata: HookManifestMetadata,
) => {
  const tools = inferHookTools(value.command);
  const commandLines = value.command.trim().split('\n').map((line) => `      ${line}`);
  return [
    'hooks:',
    `  - id: ${metadata.id.trim() || 'hook-id'}`,
    `    description: ${yamlString(metadata.description.trim() || 'Hook configuration')}`,
    `    event: ${value.event.trim() || 'EventName'}`,
    ...(tools.length > 0 ? [`    tools: [${tools.join(', ')}]`] : []),
    '    command: |',
    ...commandLines,
  ].join('\n');
};

interface HookSettingsFormProps {
  value: HookFormValue;
  onChange: (value: HookFormValue) => void;
  manifestId: string;
  manifestDescription: string;
}

export default function HookSettingsForm({
  value,
  onChange,
  manifestId,
  manifestDescription,
}: HookSettingsFormProps) {
  const update = <K extends keyof HookFormValue>(field: K, nextValue: HookFormValue[K]) => {
    onChange({ ...value, [field]: nextValue });
  };
  const preview = value.event.trim()
    ? buildHookManifestYaml(value, { id: manifestId, description: manifestDescription })
    : '';
  const selectedEvent = HOOK_EVENT_OPTIONS.find((event) => event.value === value.event);

  return (
    <div className="space-y-4">
      <div>
        <MetaMedium as="h4" tone="secondary">Hook 配置</MetaMedium>
        <MetaText as="p" tone="weak" className="mt-1">
          选择 Hook 什么时候触发，以及触发后在本地执行什么命令；其余字段由系统生成。
        </MetaText>
      </div>

      <FieldGroup className="gap-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field className="gap-2">
            <MetaMedium as="label" tone="secondary" htmlFor="hook-create-event">
              触发时机<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
            </MetaMedium>
            <Select value={value.event || undefined} onValueChange={(event) => update('event', event)}>
              <SelectTrigger id="hook-create-event" aria-label="触发时机">
                <SelectValue placeholder="请选择 Hook 触发时机" />
              </SelectTrigger>
              <SelectContent>
                {HOOK_EVENT_OPTIONS.map((event) => (
                  <SelectItem key={event.value} value={event.value}>
                    {event.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <MetaText as="p" tone="weak">
              {selectedEvent?.description || '选择用户或 Agent 的哪个动作发生时运行 Hook。'}
            </MetaText>
          </Field>
        </div>

        <Field className="gap-2">
          <MetaMedium as="label" tone="secondary" htmlFor="hook-create-command">
            触发后执行的本地命令<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
          </MetaMedium>
          <Textarea
            id="hook-create-command"
            value={value.command}
            onChange={(event) => update('command', event.target.value)}
            placeholder="例如 ruby scripts/feedback.rb"
            rows={3}
          />
          <MetaText as="p" tone="weak">
            粘贴触发时要执行的命令；请确保命令和引用的脚本在目标项目中可用。
          </MetaText>
        </Field>
      </FieldGroup>

      {preview && (
        <Collapsible>
          <CollapsibleTrigger className="group flex items-center gap-1.5 text-xs font-medium text-[var(--text-brand)] hover:underline">
            <ChevronRight className="size-3.5 transition-transform group-data-[state=open]:rotate-90" />
            查看生成配置
          </CollapsibleTrigger>
          <CollapsibleContent className="mt-2">
            <SurfaceInner className="overflow-hidden">
              <div className="border-b border-[var(--cp-border)] px-3 py-2">
                <MetaMedium as="span" tone="secondary">hooks.yaml</MetaMedium>
              </div>
              <pre className="max-h-48 overflow-auto px-3 py-2 text-xs leading-5 text-[var(--text-secondary)]">
                <code>{preview}</code>
              </pre>
            </SurfaceInner>
          </CollapsibleContent>
        </Collapsible>
      )}
    </div>
  );
}
