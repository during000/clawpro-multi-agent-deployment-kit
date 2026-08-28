/*  */
 
 
import React from 'react';
import { Tag } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from '@/components/ui/tooltip';


// 抹平各接口tag列表字段差异，返回标签列表
export const transformTags = (item:any) => {
  if (item?.MachineTag && Array.isArray(item?.MachineTag)) {
    return item?.MachineTag.map((tag: { Name: any; }) => ({
      ...tag,
      Name: tag?.Name || '',
    }));
  }
  if (item?.Tag && Array.isArray(item?.Tag)) {
    return item?.Tag.map((tag: { Name: any; }) => ({
      ...tag,
      Name: tag?.Name || '',
    }));
  }
  if (item?.Tags && Array.isArray(item?.Tags)) {
    return item?.Tags.map((tag: any) => ({
      Name: tag || '',
    }));
  }
  if (item?.TagList && Array.isArray(item?.TagList)) {
    return item?.TagList.map((tag: any) => ({
      Name: tag || '',
    }));
  }
  return [];
};

export const renderCloudTags = (item: any) => (
  <div>
    {transformTags(item)?.length + (item?.CloudTags ?? [])?.length > 0 && (
      <Tooltip>
        <TooltipTrigger asChild>
          <Badge variant="outline" className="cursor-pointer gap-1 px-1 py-0">
            <Tag className="h-3 w-3" />
            <span>{transformTags(item)?.length + (item?.CloudTags ?? [])?.length === 1 ? '标签' : '多个'}</span>
            <span className="text-[#1447E6]">({transformTags(item)?.length + (item?.CloudTags ?? [])?.length})</span>
          </Badge>
        </TooltipTrigger>
        <TooltipContent side="top" className="max-h-[500px] overflow-y-auto">
          <p className="font-bold">腾讯云标签</p>
          {item?.CloudTags?.map?.((data: { TagKey: any; TagValue: any; }, index: React.Key | null | undefined) => (
            <div key={index} className="mt-0.5 max-w-[300px] break-all">
              {`${data?.TagKey}:${data?.TagValue}`}
            </div>
          ))}
          {!item?.CloudTags?.length && <span className="text-muted-foreground">暂无腾讯云标签</span>}
          <p className="mt-2.5 mb-1 font-bold">OpenClaw标签</p>
          {transformTags(item).map((data: { Name: any }, index: React.Key | null | undefined) => (
            <div key={index} className="mt-0.5 max-w-[300px] break-all">
              {data?.Name}
            </div>
          ))}
          {!transformTags(item)?.length && <span className="text-muted-foreground">暂无OpenClaw标签</span>}
        </TooltipContent>
      </Tooltip>
    )}
    {!transformTags(item)?.length && !item?.CloudTags?.length && <span className="text-muted-foreground">暂无标签</span>}
  </div>
);
