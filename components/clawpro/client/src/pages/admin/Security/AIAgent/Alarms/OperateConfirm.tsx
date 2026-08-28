/* eslint-disable  */
 
import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from '@/components/ui/popover';

interface OperateConfirmProps {
  title: string;
  message: React.ReactNode;
  customBtn?: React.ReactNode;
  btnType?: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link';
  btnText?: string;
  confirmText?: string;
  cancelText?: string;
  btnStyle?: any;
  btnClickHandle?: () => void;
  operateHandle: () => void;
  cancelHandle?: () => void;
}

export default function OperateConfirm({
  title,
  message,
  customBtn = null,
  btnType = 'link',
  btnText,
  btnStyle = {},
  confirmText = '确定',
  cancelText = '取消',
  btnClickHandle = () => {},
  operateHandle = () => {},
  cancelHandle = () => {},
}: OperateConfirmProps) {
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        {customBtn || (
          <Button
            variant={btnType as any}
            className="px-0 h-auto"
            style={{ ...btnStyle }}
            onClick={() => {
              btnClickHandle?.();
            }}
          >
            <span>{btnText}</span>
          </Button>
        )}
      </PopoverTrigger>
      <PopoverContent align="end" side="bottom" className="w-72 p-4">
        <div className="space-y-2">
          <div className="text-sm font-medium">{title}</div>
          <div className="text-sm text-muted-foreground">{message}</div>
          <div className="flex gap-2 pt-2">
            <Button
              variant="link"
              size="sm"
              disabled={loading}
              onClick={async () => {
                setLoading(true);
                setOpen(false);
                await operateHandle?.();
                setLoading(false);
              }}
            >
              <span>{confirmText}</span>
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={async () => {
                setOpen(false);
                await cancelHandle?.();
              }}
            >
              <span>{cancelText}</span>
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
