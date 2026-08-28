import { useRef } from 'react';
import { Button } from '@/components/ui/button';
import { AlertCircle, ChevronDown, ChevronRight, FileText, Loader, Trash2 } from 'lucide-react';
import { MetaText, BodyText, HelperText, CardTitle, MetaMedium } from '@/components/ui/Typography';
import type { UploadedFile } from './types';

type UploadFileCardVariant = 'skill' | 'plugin' | 'standard';

interface UploadFileCardProps {
  file: UploadedFile | null;
  expanded?: boolean;
  onToggleExpand?: () => void;
  onZipUpload: (files: FileList) => void;
  onFolderUpload?: (files: FileList) => void;
  onRemove: () => void;
  variant?: UploadFileCardVariant;
  accept?: string;
  uploadHint?: string;
  uploadButtonLabel?: string;
}

export default function UploadFileCard({
  file,
  expanded = false,
  onToggleExpand,
  onZipUpload,
  onFolderUpload,
  onRemove,
  variant = 'skill',
  accept,
  uploadHint,
  uploadButtonLabel,
}: UploadFileCardProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);
  const isPlugin = variant === 'plugin';
  const isStandard = variant === 'standard';
  const showFileDetails = !isStandard && !!file?.files;

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    const files = e.dataTransfer.files;
    if (files && files.length > 0) {
      onZipUpload(files);
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      onZipUpload(e.target.files);
    }
    e.target.value = '';
  };

  const handleFolderChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      onFolderUpload?.(e.target.files);
    }
    e.target.value = '';
  };

  // 未上传状态
  if (!file) {
    return (
      <div
        onDragOver={handleDragOver}
        onDrop={handleDrop}
        className="border border-dashed rounded-[4px] p-4 text-center transition-colors border-gray-200 hover:border-blue-500"
      >
        <HelperText className="mb-3">
          {uploadHint || (isStandard ? '点击或拖拽 Markdown 文件上传' : '点击或拖拽 ZIP 文件上传')}
        </HelperText>

        <div className="flex gap-3 justify-center">
          <Button
            variant="outline"
            size="sm"
            onClick={() => fileInputRef.current?.click()}
          >
            {uploadButtonLabel || (isStandard ? '上传 Markdown' : '上传 ZIP')}
          </Button>
          {!isPlugin && !isStandard && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => folderInputRef.current?.click()}
            >
              选择文件夹
            </Button>
          )}
        </div>

        <input
          ref={fileInputRef}
          type="file"
          accept={accept || (isStandard ? '.md' : '.zip')}
          multiple
          onChange={handleFileChange}
          className="hidden"
        />
        {!isPlugin && !isStandard && (
          <input
            ref={folderInputRef}
            type="file"
            multiple
            onChange={handleFolderChange}
            className="hidden"
            {...({ webkitdirectory: '' } as any)}
          />
        )}
      </div>
    );
  }

  // 解析中状态
  if (file.status === 'parsing') {
    return (
      <div className="border border-gray-200 rounded-[4px] bg-white overflow-hidden px-3 py-3">
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <Loader className="w-5 h-5 text-[#355EF1] animate-spin shrink-0" />
          {file.name === '文件夹上传' ? (
            <CardTitle as="span">{file.name}</CardTitle>
          ) : (
            <MetaText as="span">{file.name}</MetaText>
          )}
          <HelperText as="span">正在解析...</HelperText>
        </div>
      </div>
    );
  }

  // 报错状态
  if (file.status === 'error') {
    return (
      <div className="border border-gray-200 rounded-[4px] bg-white overflow-hidden">
        <div className="flex items-center justify-between px-3 py-3">
          <div className="flex items-center gap-2 flex-1 min-w-0">
            <AlertCircle className="w-5 h-5 text-red-600 shrink-0" />
            <div className="flex items-center gap-2 min-w-0 truncate">
              {file.name === '文件夹上传' ? (
                <CardTitle as="span">{file.name}</CardTitle>
              ) : (
                <MetaText as="span">{file.name}</MetaText>
              )}
              <MetaText as="span" tone="danger" className="truncate">
                {file.error}
              </MetaText>
            </div>
          </div>
          <div className="flex items-center gap-1 shrink-0">
            <Button
              variant="ghost"
              size="sm"
              onClick={onRemove}
              className="h-7 w-7 p-0 hover:bg-red-50 hover:text-red-500"
            >
              <Trash2 className="w-4 h-4" />
            </Button>
          </div>
        </div>
      </div>
    );
  }

  // 已上传状态
  return (
    <div className="border border-gray-200 rounded-[4px] bg-white overflow-hidden">
      <div
        className="flex items-center justify-between px-3 py-3 cursor-pointer hover:bg-[#FAFAFA] transition-colors"
        onClick={() => onToggleExpand?.()}
      >
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <span className="w-7 h-7 rounded-full bg-[#F5F5F5] flex items-center justify-center shrink-0">
            <FileText className="w-4 h-4 text-[#525252]" />
          </span>
          {file.name === '文件夹上传' ? (
            <CardTitle as="span" className="truncate">{file.name}</CardTitle>
          ) : (
            <BodyText as="span" className="truncate">{file.name}</BodyText>
          )}
          {showFileDetails && (
            <HelperText as="span" className="shrink-0">
              包含 {file.files?.length || 0} 个文件
            </HelperText>
          )}
        </div>

        <div className="flex items-center gap-1 shrink-0">
          <Button
            variant="ghost"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              onRemove();
            }}
            className="h-7 w-7 p-0 hover:bg-red-50 hover:text-red-500"
          >
            <Trash2 className="w-4 h-4" />
          </Button>
          {showFileDetails && (
            <span className="h-7 w-7 flex items-center justify-center">
              {expanded ? (
                <ChevronDown className="w-4 h-4 text-[var(--text-muted)]" />
              ) : (
                <ChevronRight className="w-4 h-4 text-[var(--text-muted)]" />
              )}
            </span>
          )}
        </div>
      </div>

      {/* 文件详情展开 */}
      {expanded && showFileDetails && file.files && (
        <div className="border-t border-[var(--border)] bg-[var(--card)] p-3 space-y-2">
          <HelperText>文件列表</HelperText>
          <div className="space-y-1 max-h-48 overflow-y-auto">
            {file.files?.map((f) => (
              <div key={f.name} className="flex justify-between">
                <MetaText>{f.name}</MetaText>
                <MetaText>{(f.size / 1024).toFixed(2)} KB</MetaText>
              </div>
            ))}
          </div>

          {/* Skill 场景：显示 SKILL.md 校验结果 */}
          {!isPlugin && file.skillmdParsed && (
            <div className="mt-3 pt-3 border-t border-[var(--border)]">
              <HelperText className="mb-2 text-[var(--text-success)]">SKILL.md 校验通过</HelperText>
              <div className="space-y-1">
                {file.skillmdParsed.name && (
                  <div className="flex gap-1">
                    <MetaMedium as="span" className="text-[var(--text-success)]">name:</MetaMedium>
                    <MetaText>{file.skillmdParsed.name}</MetaText>
                  </div>
                )}
                {file.skillmdParsed.description && (
                  <div className="space-y-0.5">
                    <div>
                      <MetaMedium as="span" className="text-[var(--text-success)]">description:</MetaMedium>
                    </div>
                    <MetaText>{file.skillmdParsed.description}</MetaText>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Plugin 场景：显示 agent.plugin.json 和 package.json 校验结果 */}
          {isPlugin && (file.pluginJsonFound || file.packageJsonFound) && (
            <div className="mt-3 pt-3 border-t border-[var(--border)] space-y-1.5">
              {file.pluginJsonFound && (
                <HelperText className="text-[var(--text-success)]">agent.plugin.json 校验通过</HelperText>
              )}
              {file.packageJsonFound && (
                <HelperText className="text-[var(--text-success)]">package.json 校验通过</HelperText>
              )}
              {file.pluginJsonParsed && (
                <div className="space-y-1 mt-1">
                  {file.pluginJsonParsed.name && (
                    <div className="flex gap-1">
                      <MetaMedium as="span" className="text-[var(--text-success)]">name:</MetaMedium>
                      <MetaText>{file.pluginJsonParsed.name}</MetaText>
                    </div>
                  )}
                  {file.pluginJsonParsed.description && (
                    <div className="space-y-0.5">
                      <div>
                        <MetaMedium as="span" className="text-[var(--text-success)]">description:</MetaMedium>
                      </div>
                      <MetaText>{file.pluginJsonParsed.description}</MetaText>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
