import React, { useState, useEffect } from 'react';
import { Copy } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Pagination } from '@/components/ui/pagination';

import { DescribeLicenseBindSchedule } from '@/pages/admin/Security/api';

const CopyBtn = ({ text }: { text?: string }) => {
  if (!text) return null;
  return (
    <Copy
      className="inline-block h-3.5 w-3.5 ml-1 cursor-pointer text-muted-foreground hover:text-foreground align-middle"
      onClick={(e: React.MouseEvent) => {
        e.stopPropagation();
        navigator.clipboard.writeText(text);
      }}
    />
  );
};

const BindErrModal = (props: any) => {
  const { initErrData, total, visible, setVisible, bindTaskId, setDetailOutClickClose = undefined, aiAgentHostList } = props;

  const [bindErrList, setBindErrList] = useState([]);
  const [bindErrTotal, setBindErrTotal] = useState(0);
  const [bindErrQuery, setBindErrQuery] = useState({
    Offset: 0,
    Limit: 10,
  });

  const onPageChange = async (page: number) => {
    const offset = (page - 1) * bindErrQuery.Limit;
    setBindErrQuery(prev => ({ ...prev, Offset: offset }));
    const data: any = await DescribeLicenseBindSchedule({
      TaskId: bindTaskId,
      Offset: offset,
      Limit: bindErrQuery.Limit,
      Filters: [{ Name: 'Status', Values: ['2'] }],
    });
    if (data) {
      setBindErrTotal(data?.TotalCount ?? 0);
      setBindErrList(data?.List ?? []);
    }
  };

  useEffect(() => {
    setBindErrTotal(total);
    setBindErrList(initErrData);
  }, []);

  return (
    <Dialog
      open={visible}
      onOpenChange={(open) => {
        if (!open) {
          setDetailOutClickClose?.(true);
          setVisible?.(false);
          setBindErrQuery({ Offset: 0, Limit: 10 });
        }
      }}
    >
      <DialogContent className="sm:max-w-[920px]">
        <DialogHeader>
          <DialogTitle>
            <span>以下 </span>
            <span className="text-[#DC2626] mx-1">
              {bindErrTotal}
            </span>
            <span> 台OpenClaw绑定失败</span>
          </DialogTitle>
        </DialogHeader>
        <div>
          <div className="mb-4 text-sm text-[#525252] leading-relaxed">
            绑定失败OpenClaw资产及失败原因如下，请根据解决方案进行调整后进行重新绑定：
          </div>
          <ScrollArea className="max-h-[400px]">
            <Table>
              <TableHeader>
                <TableRow>
                  {/* <TableHead>OpenClaw QUUID</TableHead> */}
                  <TableHead>Agent名称</TableHead>
                  {/* <TableHead style={{ width: 140 }}>IP地址</TableHead> */}
                  <TableHead>失败原因</TableHead>
                  <TableHead>解决方案</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(bindErrList || []).map((item: any, index: number) => (
                  <TableRow key={item?.Quuid || index}>
                    <TableCell>{aiAgentHostList?.find?.((d: any) => d?.InstanceID === (item?.MachineExtraInfo?.InstanceID || item?.InstanceID || item?.InstanceId))?.OpenClawName || '-'}</TableCell>
                    {/* <TableCell>
                      <div>
                        <div title={item?.MachineExtraInfo?.HostName} className="machineName-btn-textOverflow">
                          {aiAgentHostList?.find?.((d: any) => d?.InstanceID === (item?.MachineExtraInfo?.InstanceID || item?.InstanceID || item?.InstanceId))?.OpenClawName || '-'}
                        </div>
                        <div>
                          {item?.MachineExtraInfo?.InstanceID || item?.InstanceID || item?.InstanceId || '--'}
                          {(item?.MachineExtraInfo?.InstanceID || item?.InstanceID || item?.InstanceId) && (
                            <CopyBtn text={item?.MachineExtraInfo?.InstanceID || item?.InstanceID || item?.InstanceId} />
                          )}
                        </div>
                      </div>
                    </TableCell> */}
                    {/* <TableCell>
                      <div>
                        <div>
                          <span className="newbuy-ip-label">公</span>
                          <span className="newbuy-table-text">
                            {item?.MachineExtraInfo?.WanIP || '--'}
                            {item?.MachineExtraInfo?.WanIP && <CopyBtn text={item?.MachineExtraInfo?.WanIP} />}
                          </span>
                        </div>
                        <div>
                          <span className="newbuy-ip-label">内</span>
                          <span className="newbuy-table-text">
                            {item?.MachineExtraInfo?.PrivateIP || item?.Hostip || item?.HostIp || '--'}
                            {(item?.MachineExtraInfo?.PrivateIP || item?.Hostip || item?.HostIp) && (
                              <CopyBtn text={item?.MachineExtraInfo?.PrivateIP || item?.Hostip || item?.HostIp} />
                            )}
                          </span>
                        </div>
                      </div>
                    </TableCell> */}
                    <TableCell>{item?.ErrMsg || '-'}</TableCell>
                    <TableCell>{item?.FixMessage || '-'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </ScrollArea>
          <div className="mt-4">
            <Pagination
              total={bindErrTotal}
              current={bindErrQuery.Offset / bindErrQuery.Limit + 1}
              pageSize={10}
              size="small"
              showTotal={(t) => `共 ${t} 条记录`}
              className="w-full justify-between"
              onChange={(p) => onPageChange(p)}
            />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default BindErrModal;
