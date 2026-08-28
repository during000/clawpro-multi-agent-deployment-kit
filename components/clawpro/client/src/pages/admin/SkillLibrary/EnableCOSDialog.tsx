import { useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Switch } from '@/components/ui/switch';
import { Input } from '@/components/ui/input';
import { AlertCircle, Info } from 'lucide-react';

interface EnableCOSDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}

export default function EnableCOSDialog({ open, onOpenChange, onConfirm }: EnableCOSDialogProps) {
  const [bucketName, setBucketName] = useState('clawpro-skills-1');
  const [multiAZ, setMultiAZ] = useState(false);
  const [agreed, setAgreed] = useState(false);

  const handleConfirm = () => {
    if (agreed) {
      onConfirm();
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>开启服务</DialogTitle>
        </DialogHeader>

        <div className="space-y-6">
          {/* 说明文字 */}
          <div className="text-sm text-gray-700">
            <p>开启后，将会在您的账号下的 广州 地域创建一个存储桶用于存放上传的 Skill 文件，更安全可控；会根据实际使用收取<span className="font-bold text-blue-600">存储费和上传下载流量费</span>。</p>
          </div>

          {/* COS 桶名称 */}
          <div>
            <label className="block text-sm font-semibold text-gray-900 mb-2">
              COS 桶名称
            </label>
            <Input
              value={bucketName}
              onChange={(e) => setBucketName(e.target.value)}
              placeholder="输入桶名称"
              className="w-full"
            />
            <p className="text-xs text-gray-500 mt-1">格式：{bucketName}-appid</p>
          </div>

          {/* 多 AZ 功能 */}
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <label className="text-sm font-semibold text-gray-900">多 AZ 特性</label>
              <Switch
                checked={multiAZ}
                onCheckedChange={setMultiAZ}
              />
            </div>

            {/* 多 AZ 说明 */}
            <div className="space-y-3 text-xs text-gray-600">
              <div className="flex gap-2">
                <Info className="w-4 h-4 text-blue-500 shrink-0 mt-0.5" />
                <div>
                  <p className="font-semibold text-gray-900">【说明】</p>
                  <p>多 AZ 特性允许用户将数据存储在同地理区域内的不同物理位置，提供同城容灾功能，推荐开启。
                    <a href="https://cloud.tencent.com/document/product/436/多-az-存储和单-az-存储对比" target="_blank" rel="noopener noreferrer" className="text-blue-600 underline ml-1">
                      了解更多
                    </a>
                  </p>
                </div>
              </div>

              <div className="flex gap-2">
                <AlertCircle className="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
                <div>
                  <p className="font-semibold text-gray-900">【计费】</p>
                  <p>多 AZ 配置会导致 存储容量费用相比单 AZ 有增加，且目前 暂无多 AZ 资源包，详情请参考该地域的
                    <a href="https://buy.cloud.tencent.com/price/cos/overview" target="_blank" rel="noopener noreferrer" className="text-blue-600 underline ml-1">
                      产品价格
                    </a>
                  </p>
                </div>
              </div>

              <div className="flex gap-2">
                <AlertCircle className="w-4 h-4 text-red-500 shrink-0 mt-0.5" />
                <div>
                  <p className="font-semibold text-gray-900">【注意】</p>
                  <p>多 AZ 特性 开启后无法关闭，数据将存储为多 AZ 类型。若关闭，将存储为单 AZ 类型，请根据业务需求谨慎选择，避免后续产生迁移成本。多 AZ 和单 AZ 存储的对比请见
                    <a href="https://cloud.tencent.com/document/product/436/40548#.E5.A4.9A-az-.E7.9A.84.E4.BC.98.E5.8A.BF" target="_blank" rel="noopener noreferrer" className="text-blue-600 underline ml-1">
                      文档
                    </a>
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* 同意条款 */}
          <div className="flex items-start gap-3">
            <Checkbox
              id="agree"
              checked={agreed}
              onCheckedChange={(checked) => setAgreed(checked === true)}
            />
            <label htmlFor="agree" className="text-sm text-gray-700 cursor-pointer">
              我已阅读并同意
            </label>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button onClick={handleConfirm} disabled={!agreed}>
            确认
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
