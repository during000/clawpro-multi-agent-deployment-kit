import { useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Category } from './types';

interface AddCategoryDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (category: Category) => void;
  existingCategories: Category[];
}

export default function AddCategoryDialog({
  open,
  onOpenChange,
  onConfirm,
  existingCategories,
}: AddCategoryDialogProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState('');

  const handleConfirm = () => {
    setError('');

    if (!name.trim()) {
      setError('分类名称不能为空');
      return;
    }

    if (existingCategories.some(cat => cat.name === name)) {
      setError('该分类已存在');
      return;
    }

    const newCategory: Category = {
      id: `cat-${Date.now()}`,
      name,
      description,
    };

    onConfirm(newCategory);
    resetForm();
  };

  const resetForm = () => {
    setName('');
    setDescription('');
    setError('');
  };

  const handleClose = () => {
    resetForm();
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>新增分类</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-semibold text-gray-900 mb-2">
              分类名称 <span className="text-red-500">*</span>
            </label>
            <Input
              placeholder="请输入分类名称"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full"
            />
          </div>

          <div>
            <label className="block text-sm font-semibold text-gray-900 mb-2">
              描述（非必填）
            </label>
            <Textarea
              placeholder="请输入分类定位或覆盖范围"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full"
              rows={3}
            />
          </div>

          {error && (
            <p className="text-sm text-red-600">{error}</p>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            取消
          </Button>
          <Button onClick={handleConfirm}>
            确认添加
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
