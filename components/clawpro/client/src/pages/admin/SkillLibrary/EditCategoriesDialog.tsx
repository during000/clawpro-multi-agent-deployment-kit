import { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Category } from './types';

interface EditCategoriesDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  categories: Category[];
  selectedCategoryIds: string[];
  skillName?: string;
  onConfirm: (selectedCategoryIds: string[]) => void;
}

export default function EditCategoriesDialog({
  open,
  onOpenChange,
  categories,
  selectedCategoryIds,
  skillName,
  onConfirm,
}: EditCategoriesDialogProps) {
  const [selected, setSelected] = useState<string[]>([]);

  useEffect(() => {
    if (open) {
      setSelected(selectedCategoryIds);
    }
  }, [open, selectedCategoryIds]);

  const handleToggleCategory = (categoryId: string) => {
    setSelected(prev =>
      prev.includes(categoryId)
        ? prev.filter(id => id !== categoryId)
        : [...prev, categoryId]
    );
  };

  const handleConfirm = () => {
    onConfirm(selected);
    onOpenChange(false);
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleCancel}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>修改分类</DialogTitle>
          {skillName && (
            <p className="text-sm text-gray-600 mt-2">请选择 {skillName} Skill 的分类</p>
          )}
        </DialogHeader>

        <div className="space-y-3">
          <div className="flex flex-wrap gap-1.5">
            {categories.map((cat) => (
              <button
                key={cat.id}
                onClick={() => handleToggleCategory(cat.id)}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-all border ${
                  selected.includes(cat.id)
                    ? 'text-white border-transparent'
                    : 'bg-white text-gray-600 border-gray-200 hover:border-gray-300 hover:shadow-sm'
                }`}
                style={selected.includes(cat.id) ? { backgroundColor: '#007AFF', borderColor: '#007AFF' } : undefined}
              >
                {cat.name}
              </button>
            ))}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleCancel}>
            取消
          </Button>
          <Button onClick={handleConfirm}>
            确认
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
