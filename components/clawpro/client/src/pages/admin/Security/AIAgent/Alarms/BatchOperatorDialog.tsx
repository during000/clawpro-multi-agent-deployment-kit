import React, { useState, useEffect } from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Table, TableBody, TableRow, TableCell } from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';

interface BatchOperatorDialogProps {
  onOk: () => void;
  title?: string;
  data?: any[];
  okText?: string;
  cancelText?: string;
  onCancel: () => void;
  renderItem: (item: any) => any;
  size?: string | number;
  visible: boolean;
  content?: string | React.ReactNode;
  disabled?: boolean;
}

export default function BatchOperatorDialog({
  onOk,
  title = '操作',
  data = [],
  okText = '确定',
  cancelText = '取消',
  onCancel,
  renderItem,
  visible,
  content = null,
  disabled = false,
}: BatchOperatorDialogProps) {
  const [show, setShow] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const handleChangeTableShow = () => {
    setShow(!show);
  };

  useEffect(() => {
    if (visible) {
      setShow(false);
      setIsLoading(false);
    }
  }, [visible]);

  return (
    <Dialog open={visible} onOpenChange={open => { if (!open) onCancel(); }}>
      <DialogContent className="sm:max-w-[460px]">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>

        <div className="space-y-3">
          {content && <div className="text-sm text-[#525252] leading-relaxed">{content}</div>}

          <div className="text-sm text-[#0A0A0A]">
            您已经选择 <span className="font-medium text-[#1447E6]">{data?.length}</span> 条数据，
            <button
              onClick={handleChangeTableShow}
              className="text-[#1447E6] hover:underline inline-flex items-center gap-0.5"
            >
              查看详情
              {show ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
            </button>
          </div>

          {show && data?.length > 0 && (
            <div className="border border-[#E5E5E5] rounded-[4px] overflow-hidden">
              <Table density="compact" containerClassName="max-h-[130px] overflow-y-auto">
                <TableBody>
                  {data.map((item, idx) => (
                    <TableRow key={idx}>
                      <TableCell className="w-10 text-center text-gray-400">{idx + 1}</TableCell>
                      <TableCell className="text-gray-600">{renderItem(item)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="claw-outline" size="claw-sm" onClick={onCancel}>
            {cancelText}
          </Button>
          <Button
            variant="dialog-confirm"
            size="claw-sm"
            onClick={() => {
              setIsLoading(true);
              onOk?.();
            }}
            disabled={disabled || isLoading || !data?.length}
          >
            {okText}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
