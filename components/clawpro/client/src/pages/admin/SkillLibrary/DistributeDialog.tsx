import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Search } from 'lucide-react';
import { type DistributionStatus, DISTRIBUTION_STATUS_MAP } from './types';

interface AgentInstance {
  id: string;
  name: string;
  createdBy: string;
  distributionStatus?: DistributionStatus;
}

interface DistributeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  skillName: string;
  instances: AgentInstance[];
  onDistribute: (selectedInstanceIds: string[]) => void;
  onViewProgress: () => void;
}

export default function DistributeDialog({
  open,
  onOpenChange,
  skillName,
  instances,
  onDistribute,
  onViewProgress,
}: DistributeDialogProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedInstances, setSelectedInstances] = useState<string[]>([]);
  const [distributionFilter, setDistributionFilter] = useState<'all' | DistributionStatus>('all');
  const [isDistributing, setIsDistributing] = useState(false);
  const [showSuccessMessage, setShowSuccessMessage] = useState(false);

  const filteredInstances = instances.filter(instance => {
    const matchesSearch = instance.name.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesFilter = 
      distributionFilter === 'all' ||
      instance.distributionStatus === distributionFilter;
    return matchesSearch && matchesFilter;
  });

  const handleSelectAll = () => {
    if (selectedInstances.length === filteredInstances.length) {
      setSelectedInstances([]);
    } else {
      setSelectedInstances(filteredInstances.map(i => i.id));
    }
  };

  const handleSelectInstance = (id: string) => {
    setSelectedInstances(prev =>
      prev.includes(id) ? prev.filter(i => i !== id) : [...prev, id]
    );
  };

  const handleStartDistribute = () => {
    setIsDistributing(true);
    onDistribute(selectedInstances);
    
    setTimeout(() => {
      setShowSuccessMessage(true);
    }, 500);
  };

  const handleConfirm = () => {
    setShowSuccessMessage(false);
    setIsDistributing(false);
    setSelectedInstances([]);
    setSearchQuery('');
    setDistributionFilter('all');
    onOpenChange(false);
  };

  const getStatusDisplay = (status?: DistributionStatus) => {
    const s = status || 'not_distributed';
    const { label, color } = DISTRIBUTION_STATUS_MAP[s];
    return <span className={`font-medium text-xs ${color.split(' ')[0]}`}>{label}</span>;
  };

  if (showSuccessMessage) {
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>下发 {skillName}</DialogTitle>
          </DialogHeader>
          <div className="py-6">
            <div className="flex items-start gap-3">
              <div className="text-green-600 mt-1">
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
              </div>
              <div>
                <p className="font-medium text-[#0A0A0A]">已开始安装流程</p>
                <p className="text-sm text-[#737373] mt-1">
                  已向 {selectedInstances.length} 个 Agent 实例下发 {skillName}
                </p>
              </div>
            </div>
          </div>
          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={handleConfirm}>
              确认
            </Button>
            <Button variant="dialog-confirm" onClick={onViewProgress}>
              查看进度
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[720px]">
        <DialogHeader>
          <DialogTitle>下发 {skillName}</DialogTitle>
          <DialogDescription>
            选择要下发该 Skill 的 Agent 实例
          </DialogDescription>
        </DialogHeader>

        {/* 搜索框 + 筛选 */}
        <div className="flex gap-2 mb-4">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#A3A3A3]" />
            <Input
              placeholder="搜索 Agent 云服务器..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10"
            />
          </div>
          <Select value={distributionFilter} onValueChange={(value: any) => setDistributionFilter(value)}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部</SelectItem>
              <SelectItem value="not_distributed">未下发</SelectItem>
              <SelectItem value="success">成功</SelectItem>
              <SelectItem value="failed">失败</SelectItem>
              <SelectItem value="distributing">下发中</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* 实例列表 */}
        <div className="border border-gray-200 rounded-xl overflow-hidden">
          {/* 全选复选框 — 固定在滚动区外部 */}
          <div className="flex items-center gap-3 p-2 border-b border-gray-200 bg-gray-50">
            <Checkbox
              checked={selectedInstances.length === filteredInstances.length && filteredInstances.length > 0}
              onCheckedChange={handleSelectAll}
            />
            <span className="text-sm font-medium text-[#0A0A0A]">
              全选 ({selectedInstances.length}/{filteredInstances.length})
            </span>
          </div>

          {/* 实例列表滚动区 */}
          <div className="max-h-52 overflow-y-auto">
            {filteredInstances.map(instance => (
              <div
                key={instance.id}
                className="flex items-center gap-3 p-2 border-b border-gray-200 last:border-b-0 hover:bg-gray-50"
              >
                <Checkbox
                  checked={selectedInstances.includes(instance.id)}
                  onCheckedChange={() => handleSelectInstance(instance.id)}
                />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-[#0A0A0A] truncate">{instance.name}</p>
                </div>
                <div className="flex-shrink-0">
                  {getStatusDisplay(instance.distributionStatus)}
                </div>
              </div>
            ))}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="dialog-confirm"
            onClick={handleStartDistribute}
            disabled={selectedInstances.length === 0 || isDistributing}
          >
            {isDistributing ? '下发中...' : '确认下发'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
