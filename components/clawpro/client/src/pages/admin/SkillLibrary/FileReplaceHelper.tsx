import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { HelperText } from '@/components/ui/Typography';
import { UploadRequirementsCard } from './UploadRequirementsCard';

interface FileReplaceHelperProps {
  show: boolean;
  /** 提示文字，默认"如需替换，请先删除当前文件" */
  children?: React.ReactNode;
  /** 上传要求卡片变体 */
  variant?: 'skill' | 'plugin-upload' | 'plugin-update' | 'standard';
  /** 是否显示"查看上传要求"按钮，默认 true */
  showRequirements?: boolean;
}

export default function FileReplaceHelper({
  show,
  children,
  variant = 'skill',
  showRequirements = true,
}: FileReplaceHelperProps) {
  if (!show) return null;

  return (
    <HelperText>
      {children ?? '如需替换，请先删除当前文件'}
      {showRequirements && (
        <Popover>
          <PopoverTrigger asChild>
            <button type="button" className="ml-2 text-[var(--text-brand)] hover:underline cursor-pointer">
              查看上传要求
            </button>
          </PopoverTrigger>
          <PopoverContent className="w-[360px] p-4" align="start" side="bottom">
            <UploadRequirementsCard variant={variant} borderless />
          </PopoverContent>
        </Popover>
      )}
    </HelperText>
  );
}
