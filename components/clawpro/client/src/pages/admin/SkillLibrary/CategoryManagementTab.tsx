import { useState } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Plus, Edit2, Trash2 } from 'lucide-react';
import { Category } from './types';
import { DEFAULT_CATEGORIES, MOCK_SKILLS } from './mockData';
import AddCategoryDialog from './AddCategoryDialog';
import EditCategoryDialog from './EditCategoryDialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';

export default function CategoryManagementTab() {
  const [categories, setCategories] = useState<Category[]>(DEFAULT_CATEGORIES);
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [selectedCategory, setSelectedCategory] = useState<Category | null>(null);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);

  // 计算每个分类下的技能数量
  const getSkillCountByCategory = (categoryId: string) => {
    return MOCK_SKILLS.filter((skill: any) => skill.categories.includes(categoryId)).length;
  };

  const handleAddCategory = (newCategory: Category) => {
    setCategories([...categories, newCategory]);
    setAddDialogOpen(false);
  };

  const handleEditCategory = (updatedCategory: Category) => {
    try {
      setCategories(categories.map(cat =>
        cat.id === updatedCategory.id ? updatedCategory : cat
      ));
      setEditDialogOpen(false);
      setSelectedCategory(null);
      toast.success('分类编辑成功');
    } catch (error) {
      toast.error('分类编辑失败');
    }
  };

  const handleDeleteCategory = () => {
    if (selectedCategory) {
      try {
        setCategories(categories.filter(cat => cat.id !== selectedCategory.id));
        setDeleteConfirmOpen(false);
        setSelectedCategory(null);
        toast.success('分类删除成功');
      } catch (error) {
        toast.error('分类删除失败');
      }
    }
  };

  const openEditDialog = (category: Category) => {
    setSelectedCategory(category);
    setEditDialogOpen(true);
  };

  const openDeleteConfirm = (category: Category) => {
    setSelectedCategory(category);
    setDeleteConfirmOpen(true);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-lg font-semibold text-gray-900">分类管理</h2>
        <Button onClick={() => setAddDialogOpen(true)}>
          <Plus className="w-4 h-4 mr-2" />
          新增分类
        </Button>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">序号</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">分类名称</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">描述</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">技能数量</th>
              <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900">操作</th>
            </tr>
          </thead>
          <tbody>
            {categories.map((category, index) => (
              <tr key={category.id} className="border-b border-gray-200 hover:bg-gray-50">
                <td className="px-6 py-4 text-sm text-gray-900">{index + 1}</td>
                <td className="px-6 py-4 text-sm text-gray-900">{category.name}</td>
                <td className="px-6 py-4 text-sm text-gray-600">{category.description}</td>
                <td className="px-6 py-4 text-sm text-gray-600">{getSkillCountByCategory(category.id)}</td>
                <td className="px-6 py-4 text-sm">
                  <div className="flex gap-4">
                    <button
                      onClick={() => openEditDialog(category)}
                      className="text-gray-400 hover:text-blue-600 transition-colors"
                      title="编辑"
                    >
                      <Edit2 className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => openDeleteConfirm(category)}
                      className="text-gray-400 hover:text-red-600 transition-colors"
                      title="删除"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <AddCategoryDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        onConfirm={handleAddCategory}
        existingCategories={categories}
      />

      {selectedCategory && (
        <>
          <EditCategoryDialog
            open={editDialogOpen}
            onOpenChange={setEditDialogOpen}
            category={selectedCategory}
            onConfirm={handleEditCategory}
          />

          {/* 删除确认对话框 */}
          {deleteConfirmOpen && (
            <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
              <div className="bg-white rounded-lg p-6 max-w-md">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">删除分类</h3>
                
                <p className="text-sm text-gray-600 mb-6">
                  确定要删除分类「{selectedCategory?.name}」吗？该分类下共有 {getSkillCountByCategory(selectedCategory?.id || '')} 个技能，删除此分类后对应skill将移除该分类。
                </p>

                <div className="flex justify-end gap-2">
                  <Button
                    variant="outline"
                    onClick={() => {
                      setDeleteConfirmOpen(false);
                    }}
                  >
                    取消
                  </Button>
                  <Button
                    variant="destructive"
                    onClick={handleDeleteCategory}
                  >
                    确认
                  </Button>
                </div>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
