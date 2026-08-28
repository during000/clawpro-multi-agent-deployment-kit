import React, { useEffect, useState, useCallback } from 'react';
import { Copy, Loader2, ExternalLink as ExternalLinkIcon, ChevronDown, X } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { PanelTitle } from '@/components/ui/Typography';
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerDescription,
  DrawerClose,
  DrawerBody,
} from '@/components/ui/drawer';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu';
import {
  DescribeMalwareInfo,
  DescribeSkillInfo,
  DescribeMachineInfo,
  DescribeBashEventsNew,
  DescribeRiskDnsEventList,
  DescribeMalWareList,
} from '@/pages/admin/Security/api';

import { SKILL_STATUS_VAL_MAP, SKILL_LEVEL_MAP, SKILL_LEVEL_THEME_MAP, RISK_TYPE_SKILL, CONSOLE_URL } from '../constants';
import { modifyEventsStatus } from '../Common/CommonRiskHandleFunc';
import { renderAgentItem } from '../Assets/AgentAssetsList';

interface SkillDetailDrawerProps {
  visible: boolean;
  record: any;
  aiAgentInfo: any;
  onClose: () => void;
  onRefresh?: () => void;
}

const TAB_KEYS = ['summary', 'detail', 'scope', 'suggestion'] as const;
const TAB_LABELS: Record<string, string> = {
  summary: '安全摘要',
  detail: '告警详情',
  scope: '影响范围',
  suggestion: '处置建议',
};

const InfoItem = ({
  label,
  children,
  className: extraClassName,
}: {
  label: string;
  children: React.ReactNode;
  className?: string;
}) => (
  <div className={`flex mb-3 leading-[22px] ${extraClassName || ''}`}>
    <span className="w-[100px] shrink-0 text-left mr-3 text-[#737373] text-sm">{label}</span>
    <div className="flex-1 break-all text-sm">{children}</div>
  </div>
);

const SectionTitle = ({ id, title }: { id: string; title: string }) => (
  <div
    id={`skill-detail-${id}`}
    className="text-sm font-semibold mb-4 pt-1"
  >
    {title}
  </div>
);

const handleCopyText = (text: string) => {
  navigator.clipboard.writeText(text);
};

export default function SkillDetailDrawer({
  visible,
  record,
  aiAgentInfo,
  onClose,
  onRefresh,
}: SkillDetailDrawerProps) {
  const [malwareInfo, setMalwareInfo] = useState<any>(null);
  const [skillInfo, setSkillInfo] = useState<any>(null);
  const [machineInfo, setMachineInfo] = useState<any>(null);
  const [relatedInfo, setRelatedInfo] = useState<any>(null);
  const [activeTab, setActiveTab] = useState<string>('summary');
  const [loading, setLoading] = useState(false);
  const [confirmType, setConfirmType] = useState<string>('');
  const [killProcess, setKillProcess] = useState(true);

  const fetchDetail = useCallback(async () => {
    if (!visible || !record?.Id) return;
    setLoading(true);
    try {
      const [malwareRes, skillRes, bashInfo, dnsInfo, malWareInfo]: any[] = await Promise.all([
        DescribeMalwareInfo({ Id: Number(record.Id) }).catch(() => null),
        DescribeSkillInfo({ Ids: [Number(record.Id)] }).catch(() => null),
        DescribeBashEventsNew({
          Limit: 10,
          Offset: 0,
          Filters: [
            { Name: 'RuleLevel', Values: ['1', '2', '3'], ExactMatch: true },
            { Name: 'Status', Values: ['0'], ExactMatch: false },
            { Name: 'InstanceID', Values: [record?.MachineExtraInfo?.InstanceID], ExactMatch: false },
          ],
        }),
        DescribeRiskDnsEventList({
          Limit: 10,
          Offset: 0,
          Filters: [
            { Name: 'HandleStatus', Values: ['0'] },
            { Name: 'InstanceID', Values: [record?.MachineExtraInfo?.InstanceID] },
          ],
        }),
        DescribeMalWareList({
          Limit: 10,
          Offset: 0,
          Filters: [
            {
              Name: 'VirusType',
              Values: ['AgentSkill'],
            },
            { Name: 'InstanceID', Values: [record?.MachineExtraInfo?.InstanceID] },
          ],
        }),
      ]);
      const info = malwareRes?.MalwareInfo || malwareRes || {};
      setMalwareInfo(info);
      setSkillInfo(skillRes?.SkillInfoList?.[0] || {});
      setRelatedInfo({
        bashCount: bashInfo?.TotalCount || 0,
        dnsCount: dnsInfo?.TotalCount || 0,
        skillCount: malWareInfo?.TotalCount || 0,
      });

      if (info?.Uuid) {
        const machineRes = await DescribeMachineInfo({ Uuid: info.Uuid }).catch(() => null);
        setMachineInfo(machineRes);
      }
    } finally {
      setLoading(false);
    }
  }, [visible, record?.Id]);

  const scrollContainerRef = React.useRef<HTMLDivElement>(null);

  const handleScroll = useCallback(() => {
    const body = scrollContainerRef.current;
    if (!body) return;
    const bodyRect = body.getBoundingClientRect();
    let current = 'summary';
    for (const key of TAB_KEYS) {
      const el = document.getElementById(`skill-detail-${key}`);
      if (el) {
        const elRect = el.getBoundingClientRect();
        if (elRect.top - bodyRect.top <= 20) {
          current = key;
        }
      }
    }
    setActiveTab(current);
  }, []);

  useEffect(() => {
    if (visible && record?.Id) {
      fetchDetail();
      setActiveTab('summary');
    } else {
      setMalwareInfo(null);
      setSkillInfo(null);
      setMachineInfo(null);
    }
  }, [visible, record?.Id]);

  useEffect(() => {
    if (!visible) return undefined;
    const timer = setTimeout(() => {
      const body = scrollContainerRef.current;
      if (!body) return;
      body.addEventListener('scroll', handleScroll);
    }, 100);
    return () => {
      clearTimeout(timer);
      scrollContainerRef.current?.removeEventListener('scroll', handleScroll);
    };
  }, [visible, handleScroll]);

  const scrollToSection = (key: string) => {
    setActiveTab(key);
    const el = document.getElementById(`skill-detail-${key}`);
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  };

  const status = String(record?.Status);
  const statusObj = SKILL_STATUS_VAL_MAP[status];
  const levelStr = String(record?.Level ?? malwareInfo?.Level ?? '');
  const levelTheme = SKILL_LEVEL_THEME_MAP[levelStr];

  const handleOperate = (type: string) => {
    modifyEventsStatus(RISK_TYPE_SKILL, type, record?.Id, () => {
      onRefresh?.();
      fetchDetail();
    });
  };

  const confirmModalConfig: Record<string, { title: string; message: React.ReactNode }> = {
    separate: {
      title: '确认将此告警隔离？',
      message: (
        <div className="space-y-3">
          <p className="text-sm text-[#525252] leading-relaxed">
            隔离此病毒文件，让黑客无法再次启动它，便于您定位病毒文件位置，对其进行查杀。（注意：windows系统下，若该文件正在运行中，会导致隔离失败）
          </p>
          <div className="flex items-center gap-2">
            <Checkbox
              id="kill-process-detail"
              checked={killProcess}
              onCheckedChange={(val) => setKillProcess(!!val)}
            />
            <label htmlFor="kill-process-detail" className="text-sm text-[#525252] cursor-pointer">
              隔离并杀掉该文件相关进程，建议勾选。
            </label>
          </div>
        </div>
      ),
    },
    ignore: {
      title: '确认将此告警标记为已忽略？',
      message: (
        <p className="text-sm text-[#525252] leading-relaxed">
          确认后，此告警的处理状态将变更为已忽略，该资产当天若再命中该告警策略，则不告警，处置状态仍为&quot;已忽略&quot;；第二天若触发将重新告警（新增一条）
        </p>
      ),
    },
    mark: {
      title: '确认将此告警标记为已处置？',
      message: (
        <p className="text-sm text-[#525252] leading-relaxed">
          确认后，此告警的处理状态将变更为已处置，该资产当天若再命中该告警策略，则不告警，处置状态仍为&quot;已处置&quot;；第二天若触发将重新告警（新增一条）
        </p>
      ),
    },
    del: {
      title: '确认删除此告警？',
      message: (
        <p className="text-sm text-[#525252] leading-relaxed">
          删除该告警记录，控制台将不再显示，无法恢复记录，请慎重操作。
        </p>
      ),
    },
  };

  const renderConfirmModal = () => {
    const config = confirmModalConfig[confirmType];
    if (!config) return null;
    return (
      <Dialog open onOpenChange={() => setConfirmType('')}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle>{config.title}</DialogTitle>
          </DialogHeader>
          <div className="py-2 text-sm text-[#525252] leading-relaxed">{config.message}</div>
          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={() => setConfirmType('')}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={() => {
                setConfirmType('');
                handleOperate(confirmType);
              }}
            >
              确定
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  };

  const renderExtra = () => {
    if (status !== '4') return null;
    return (
      <div className="flex items-center gap-2">
        <Button
          variant="claw-primary"
          size="claw-sm"
          onClick={() => setConfirmType('mark')}
        >
          标记处置
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="claw-outline" size="claw-sm">
              更多
              <ChevronDown className="w-3 h-3 ml-0.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => setConfirmType('ignore')}>
              标记忽略
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setConfirmType('separate')}>
              隔离文件
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setConfirmType('del')}>
              删除记录
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    );
  };

  const renderTabs = () => (
    <div className="flex gap-6 text-xs border-b border-[#E5E5E5] mt-3">
      {TAB_KEYS.map(key => (
        <div
          key={key}
          onClick={() => scrollToSection(key)}
          className={`pb-2 cursor-pointer transition-colors ${
            activeTab === key
              ? 'font-semibold text-[#1447E6] border-b-2 border-blue-600'
              : 'text-[#737373] hover:text-[#525252] border-b-2 border-transparent'
          }`}
        >
          {TAB_LABELS[key]}
        </div>
      ))}
    </div>
  );

  const renderSummary = () => {
    const description = skillInfo?.RiskDesc || '-';
    return (
      <div>
        <SectionTitle id="summary" title="安全摘要" />
        <div
          className="rounded-[4px] border border-[#E5E5E5] p-4"
          style={{
            background:
              'linear-gradient(135deg, rgba(180, 44, 63, 0.06) 0%, rgba(180, 44, 63, 0) 30%, #fff 30%), #fff',
          }}
        >
          <div className="mb-3">
            <span className="font-semibold text-sm">危害描述</span>
            <span className="text-sm">：{description}</span>
          </div>
          <div className="flex gap-4">
            <div className="flex-1 border border-[#E5E5E5] rounded p-4">
              <span className="block text-[#737373] text-xs mb-2">威胁等级</span>
              {levelTheme ? (
                <span
                  className="inline-flex items-center gap-1.5 text-sm font-medium whitespace-nowrap"
                  style={{ color: levelTheme.color }}
                >
                  <span
                    className="w-1.5 h-1.5 rounded-full flex-shrink-0"
                    style={{ background: (levelTheme as any).dot || levelTheme.color }}
                  />
                  {SKILL_LEVEL_MAP[levelStr] || '无'}
                </span>
              ) : (
                <span className="text-sm">{SKILL_LEVEL_MAP[levelStr] || '-'}</span>
              )}
            </div>
            <div className="flex-1 border border-[#E5E5E5] rounded p-4">
              <span className="block text-[#737373] text-xs mb-2">受影响资产（ID/名称）</span>
              <div className="flex items-center gap-2">
                <a
                  href={`${CONSOLE_URL}/cwp/machine?Uuid=${malwareInfo?.Uuid}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[#1447E6] hover:underline text-sm"
                >
                  {record?.MachineExtraInfo?.InstanceID
                    || malwareInfo?.MachineExtraInfo?.InstanceID
                    || record?.InstanceId
                    || '-'}
                </a>
                <span className="text-sm ml-2">{malwareInfo?.ServersName || record?.Alias || '-'}</span>
              </div>
            </div>
          </div>
          <div className="mt-3 border border-[#E5E5E5] rounded p-3 pr-10 relative">
            <div className="font-semibold text-sm mb-2">证据</div>
            <InfoItem label="文件路径">
              <span className="text-sm">{malwareInfo?.FilePath || record?.FilePath || '-'}</span>
            </InfoItem>
            <InfoItem label="文件名">{malwareInfo?.FileName || '-'}</InfoItem>
            <InfoItem label="证据">
              <span className="whitespace-pre-wrap break-all text-sm leading-[22px]">
                {skillInfo?.Evidence || '-'}
              </span>
            </InfoItem>
          </div>
        </div>
      </div>
    );
  };

  const renderDetail = () => {
    const fileSize = malwareInfo?.FileSize ? `${(malwareInfo.FileSize / 1024).toFixed(2)}KB` : '-';

    return (
      <div>
        <SectionTitle id="detail" title="告警详情" />
        <div className="rounded-[4px] border border-[#E5E5E5] p-4">
          <InfoItem label="Skill名称">{skillInfo?.SkillName || '-'}</InfoItem>
          <InfoItem label="Skill描述">{skillInfo?.SkillDesc || '-'}</InfoItem>
          <InfoItem label="Skill来源">{skillInfo?.SkillSource || '-'}</InfoItem>
          <hr className="border-[#E5E5E5] my-3" />
          <InfoItem label="标签特征">{skillInfo?.Tags?.length ? skillInfo.Tags.join('、') : '--'}</InfoItem>
          <InfoItem label="文件MD5">
            <span className="inline-flex items-center gap-1">
              <span className="text-sm">{malwareInfo?.MD5 || '-'}</span>
              {malwareInfo?.MD5 && (
                <Copy
                  className="w-3.5 h-3.5 text-[#A3A3A3] hover:text-[#1447E6] cursor-pointer"
                  onClick={() => handleCopyText(malwareInfo.MD5)}
                />
              )}
            </span>
          </InfoItem>
          <InfoItem label="文件大小">{fileSize}</InfoItem>
          <InfoItem label="首次访问时间">
            {malwareInfo?.StrFileAccessTime || malwareInfo?.FileCreateTime || '-'}
          </InfoItem>
          <InfoItem label="最近修改时间">
            {malwareInfo?.FileModifierTime || malwareInfo?.ModifyTime || '-'}
          </InfoItem>
        </div>
      </div>
    );
  };

  const renderScope = () => {
    const instanceId = record?.MachineExtraInfo?.InstanceID || malwareInfo?.MachineExtraInfo?.InstanceID || record?.InstanceId || '';
    const publicIp = malwareInfo?.MachineExtraInfo?.WanIP || machineInfo?.MachineWanIp || '';
    const privateIp = malwareInfo?.MachineExtraInfo?.PrivateIP || machineInfo?.MachineIp || '';
    const hostName = malwareInfo?.MachineExtraInfo?.HostName || machineInfo?.MachineName || record?.Alias || '';

    return (
      <div>
        <SectionTitle id="scope" title="影响范围" />
        <div className="rounded-[4px] border border-[#E5E5E5] p-4">
          <div className="border border-[#E5E5E5] rounded p-4 mb-4">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 flex items-center justify-center rounded bg-gray-50" />
              <div>
                <div className="flex items-center gap-2">
                  {instanceId ? (
                    <a
                      href={`${CONSOLE_URL}/cwp/machine?Uuid=${malwareInfo?.Uuid}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-[#1447E6] hover:underline text-sm inline-flex items-center gap-1"
                    >
                      {instanceId}
                      <ExternalLinkIcon className="w-3 h-3" />
                    </a>
                  ) : (
                    <span className="text-sm">-</span>
                  )}
                </div>
                <div className="flex items-center gap-2 mt-1 text-xs text-[#737373]">
                  <span>资产类型：<span className="text-[#525252]">{machineInfo?.MachineType || '-'}</span></span>
                  <span className="w-px h-3.5 bg-gray-200" />
                  <span>资产名称：<span className="text-[#525252]">{hostName || '-'}</span></span>
                  <span className="w-px h-3.5 bg-gray-200" />
                  <span>
                    IP地址：
                    <span className="text-[#525252]">公：{publicIp || '-'}</span>
                    <span className="mx-2 text-gray-300">|</span>
                    <span className="text-[#525252]">内：{privateIp || '-'}</span>
                  </span>
                </div>
              </div>
            </div>
          </div>

          <InfoItem label="AI Agent" className="items-center">
            <span className="inline-flex items-center gap-2">
              {renderAgentItem(aiAgentInfo, undefined)}
            </span>
          </InfoItem>
          <InfoItem label="关联告警">
            <div className="flex items-center gap-2">
              <span className="inline-flex items-center gap-1 text-sm">
                高危命令
                <Badge variant={relatedInfo?.bashCount ? 'destructive' : 'secondary'} className="text-xs">
                  {relatedInfo?.bashCount || 0}
                </Badge>
              </span>
              <span className="w-px h-3.5 bg-gray-200" />
              <span className="inline-flex items-center gap-1 text-sm">
                恶意请求
                <Badge variant={relatedInfo?.dnsCount ? 'destructive' : 'secondary'} className="text-xs">
                  {relatedInfo?.dnsCount || 0}
                </Badge>
              </span>
            </div>
          </InfoItem>
          <InfoItem label="恶意Skills">
            <span className="inline-flex items-center gap-1 text-sm">
              恶意Skills
              <Badge variant={relatedInfo?.skillCount ? 'destructive' : 'secondary'} className="text-xs">
                {relatedInfo?.skillCount || 0}
              </Badge>
            </span>
          </InfoItem>
          {/* <InfoItem label="matedata识别">{renderMeta(aiAgentInfo)}</InfoItem> */}
        </div>
      </div>
    );
  };

  const renderSuggestion = () => {
    const suggestion = malwareInfo?.SuggestScheme || '-';
    return (
      <div>
        <SectionTitle id="suggestion" title="处置建议" />
        <div className="rounded-[4px] border border-[#E5E5E5] p-4">
          <InfoItem label="建议方案" className="mb-0">
            <span className="text-sm leading-[22px]">{suggestion}</span>
          </InfoItem>
        </div>
      </div>
    );
  };

  if (!record) return null;

  const virusName = malwareInfo?.VirusName || record?.VirusName || '-';
  const eventId = record?.Id ? `event_${record.Id}` : '-';

  return (
    <>
      <Drawer open={visible} onOpenChange={open => { if (!open) onClose(); }} direction="right">
        <DrawerContent className="data-[vaul-drawer-direction=right]:w-[760px] data-[vaul-drawer-direction=right]:sm:max-w-none max-w-[calc(100vw-24px)] h-full rounded-none bg-background p-0">
          <DrawerHeader className="flex flex-row items-start justify-between gap-4 px-6 pt-5 pb-3 bg-background text-left">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <DrawerTitle asChild>
                  <PanelTitle as="h2" className="truncate max-w-[360px]">
                    {virusName}
                  </PanelTitle>
                </DrawerTitle>
                {statusObj && (() => {
                  const styleMap: Record<string, { bg: string; color: string; dot: string }> = {
                    error: { bg: '#FEF2F2', color: '#DC2626', dot: '#DC2626' },
                    success: { bg: '#F0FDF4', color: '#16A34A', dot: '#16A34A' },
                    warning: { bg: '#FFFBEB', color: '#D97706', dot: '#D97706' },
                    info: { bg: '#F3F4F6', color: '#6B7280', dot: '#6B7280' },
                  };
                  const s = styleMap[statusObj.theme] || styleMap.info;
                  return statusObj.icon === 'loading' ? (
                    <span
                      className="inline-flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium rounded-full whitespace-nowrap"
                      style={{ background: s.bg, color: s.color }}
                    >
                      <Loader2 className="w-3 h-3 animate-spin" />
                      {statusObj.text}
                    </span>
                  ) : (
                    <span
                      className="inline-flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium rounded-full whitespace-nowrap"
                      style={{ background: s.bg, color: s.color }}
                    >
                      <span
                        className="w-1.5 h-1.5 rounded-full flex-shrink-0"
                        style={{ background: s.dot }}
                      />
                      {statusObj.text}
                    </span>
                  );
                })()}
              </div>
              <DrawerDescription className="text-xs text-[#A3A3A3]">
                <span className="mr-4">类型：恶意Skills</span>
                <span>ID：{eventId}</span>
              </DrawerDescription>
              {renderTabs()}
            </div>
            <div className="flex items-start gap-1 shrink-0">
              {renderExtra()}
              <DrawerClose asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 w-7 p-0 text-gray-900 hover:text-gray-950 ml-1"
                  aria-label="关闭"
                >
                  <X className="w-4 h-4" />
                </Button>
              </DrawerClose>
            </div>
          </DrawerHeader>

          <DrawerBody>
          <div
            ref={scrollContainerRef}
            className="px-6 pb-6"
          >
            {loading ? (
              <div className="flex items-center justify-center py-10">
                <Loader2 className="w-5 h-5 animate-spin text-[#1447E6] mr-2" />
                <span className="text-sm text-[#737373]">加载中...</span>
              </div>
            ) : (
              <div className="space-y-6 pt-4">
                {renderSummary()}
                {renderDetail()}
                {renderScope()}
                {renderSuggestion()}
                <div className="h-10" />
              </div>
            )}
          </div>
          </DrawerBody>
        </DrawerContent>
      </Drawer>
      {renderConfirmModal()}
    </>
  );
}
