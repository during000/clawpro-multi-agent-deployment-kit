/* eslint-disable  */
 
 
import React, { useState, useEffect, useRef } from 'react';
import { X, Loader2, Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select';

import { DescribeTags } from '@/pages/admin/Security/api';

import { requestApi } from './requestApi';

/**
 * CvmKeySelectComponent
 *
 * NOTE: This component originally used TagSearchBox which is a complex
 * multi-attribute search box. Since there's no direct equivalent in shadcn/ui,
 * we provide a simplified tag filter interface using native Select components.
 * The component maintains the same data flow and callback interface.
 */
const CvmKeySelectComponent = (props:any) => {
  const {
    fetchMachine = () => {},
    isCvmSelected = true,
    deviceArea = '',
    setIsCvmTagKey = () => false,
    openSwitch = true,
    tagValues = undefined,
  } = props;

  const [allTag, setAllTag] = useState([] as any);
  const [allTagKeys, setAllTagKeys] = useState([]);
  const [listLoading, setListLoading] = useState(false);
  const [cvmValueSelectKey, setCvmValueSelectKey] = useState<string | null>(null);
  const [cvmValueSelectVal, setCvmValueSelectVal] = useState<(string | null)[]>([null]);
  const [selectedCwpTags, setSelectedCwpTags] = useState<string[]>([]);
  const [selectedKeyTags, setSelectedKeyTags] = useState<string[]>([]);
  const [cvmTagValuesSelection, setCvmTagValuesSelection] = useState<any[]>([]);

  const cvmTagValuesSelectionData:any = useRef(null);

  const fetchTagList = async () => {
    const tagLists: any = await DescribeTags();
    setAllTag(tagLists?.List ?? []);
  };

  const fetchTagKeys = async () => {
    setListLoading(true);
    const res = await requestApi({
      cmd: 'DescribeTagKeys',
      data: { Limit: 1000, Offset: 0 },
      regionId: 1,
      serviceType: 'tag',
      version: '2018-08-13',
    });
    setAllTagKeys(res?.Tags || []);
    setListLoading(false);
  };

  const fetchTagValues = async (key: string) => {
    setListLoading(true);
    const res = await requestApi({
      cmd: 'DescribeTagValues',
      data: { Limit: 1000, Offset: 0, TagKeys: Array.isArray(key) ? key : [key] },
      regionId: 1,
      serviceType: 'tag',
      version: '2018-08-13',
    });
    const tempData = res?.Tags?.map?.((item: { TagValue: any; }) => ({
      text: item?.TagValue ? item?.TagValue : '空值',
      value: item?.TagValue ?? '',
    }));
    cvmTagValuesSelectionData.current = tempData || [];
    setCvmTagValuesSelection(cvmTagValuesSelectionData.current);
    setListLoading(false);
  };

  useEffect(() => {
    // fetchTagList();
    // fetchTagKeys();
    cvmTagValuesSelectionData.current = [];
  }, []);

  useEffect(() => {
    if (tagValues) {
      setSelectedCwpTags(tagValues?.cwpTagIds || []);
      setSelectedKeyTags(tagValues?.keyTags || []);
      const cvmKey = tagValues?.valTags?.map?.((d: string) => d?.split?.('$')?.[0])?.[0];
      setCvmValueSelectKey(cvmKey || null);
      setCvmValueSelectVal(tagValues?.valTags?.map?.((d: string) => d?.split?.('$')?.[1] || '') || [null]);
      if (cvmKey) {
        fetchTagValues(cvmKey);
      }
    } else {
      setSelectedCwpTags([]);
      setSelectedKeyTags([]);
    }
  }, [deviceArea, isCvmSelected, allTag, allTagKeys, tagValues]);

  const buildAndFireTags = (cwpIds: string[], keyTagsArr: string[], valKey: string | null, valArr: (string | null)[]) => {
    const values: any[] = [];
    cwpIds.forEach(id => values.push({ key: '$1', val: id }));
    keyTagsArr.forEach(k => values.push({ key: '$2', val: k }));
    if (valKey) {
      valArr.filter(v => v != null).forEach(v => values.push({ key: valKey, val: v }));
    }
    const hasTags = values.length > 0;
    if (isCvmSelected) {
      setIsCvmTagKey(hasTags);
    }
    fetchMachine?.(hasTags ? values : null, null);
  };

  const handleCwpTagChange = (value: string) => {
    const next = selectedCwpTags.includes(value)
      ? selectedCwpTags.filter(v => v !== value)
      : [...selectedCwpTags, value];
    setSelectedCwpTags(next);
    buildAndFireTags(next, selectedKeyTags, cvmValueSelectKey, cvmValueSelectVal);
  };

  const handleApplyKeyValue = () => {
    buildAndFireTags(selectedCwpTags, selectedKeyTags, cvmValueSelectKey, cvmValueSelectVal);
  };

  return (
    <div className="flex flex-col gap-2">
      {/* 主机安全标签 */}
      <div className="flex items-center gap-2 flex-wrap">
        <Label className="text-xs text-muted-foreground shrink-0">OpenClaw标签:</Label>
        <Select
          disabled={!openSwitch}
          onValueChange={handleCwpTagChange}
        >
          <SelectTrigger className="w-[200px] h-8">
            <SelectValue placeholder={selectedCwpTags.length > 0
              ? `已选${selectedCwpTags.length}个标签`
              : '选择标签'} />
          </SelectTrigger>
          <SelectContent>
            {allTag.filter((item: any) => item?.Id != null && String(item.Id) !== '').map((item: any) => (
              <SelectItem key={String(item?.Id)} value={String(item.Id)}>
                {item?.Name ?? '空值'}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {selectedCwpTags.length > 0 && (
          <Button variant="ghost" size="sm" className="h-6 px-2 text-xs" onClick={() => {
            setSelectedCwpTags([]);
            buildAndFireTags([], selectedKeyTags, cvmValueSelectKey, cvmValueSelectVal);
          }}>
            清除
          </Button>
        )}
      </div>

      {/* 腾讯云标签键值 */}
      {isCvmSelected && (
        <div className="flex items-start gap-2 flex-wrap">
          <Label className="text-xs text-muted-foreground shrink-0 mt-2">腾讯云标签:</Label>
          <div className="flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <Select
                disabled={!openSwitch}
                value={cvmValueSelectKey || undefined}
                onValueChange={value => {
                  setCvmValueSelectKey(value);
                  setCvmValueSelectVal([null]);
                  fetchTagValues(value);
                }}
              >
                <SelectTrigger className="w-[180px] h-8">
                  <SelectValue placeholder="选择标签键" />
                </SelectTrigger>
                <SelectContent>
                  {listLoading && (
                    <div className="flex items-center justify-center py-2">
                      <Loader2 className="h-4 w-4 animate-spin" />
                    </div>
                  )}
                  {(allTagKeys as string[]).filter(item => item != null && item !== '').map(item => (
                    <SelectItem key={item} value={item}>
                      {item}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              {cvmValueSelectVal.map((val, index) => (
                <div key={index} className="flex items-center gap-1">
                  <Select
                    disabled={!openSwitch || !cvmValueSelectKey}
                    value={val || undefined}
                    onValueChange={value => {
                      const newVals = [...cvmValueSelectVal];
                      newVals[index] = value;
                      setCvmValueSelectVal(newVals);
                    }}
                  >
                    <SelectTrigger className="w-[180px] h-8">
                      <SelectValue placeholder="选择标签值" />
                    </SelectTrigger>
                    <SelectContent>
                      {cvmTagValuesSelection.map((opt: any) => (
                        <SelectItem key={opt.value || '__empty__'} value={opt.value || '__empty__'}>
                          {opt.text}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {cvmValueSelectVal.length > 1 && (
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6"
                      onClick={() => {
                        const newVals = cvmValueSelectVal.filter((_, i) => i !== index);
                        setCvmValueSelectVal(newVals);
                      }}
                    >
                      <X className="h-3 w-3" />
                    </Button>
                  )}
                </div>
              ))}

              <Button
                variant="ghost"
                size="sm"
                className="h-8 px-2 text-xs"
                disabled={!cvmValueSelectKey}
                onClick={() => setCvmValueSelectVal([...cvmValueSelectVal, null])}
              >
                <Plus className="h-3 w-3 mr-1" />
                添加
              </Button>
            </div>
            <div className="flex gap-2">
              <Button
                variant="claw-primary"
                size="claw-sm"
                disabled={!cvmValueSelectKey}
                onClick={handleApplyKeyValue}
              >
                确定
              </Button>
              <Button
                variant="claw-outline"
                size="claw-sm"
                onClick={() => {
                  setCvmValueSelectKey(null);
                  setCvmValueSelectVal([null]);
                  buildAndFireTags(selectedCwpTags, selectedKeyTags, null, []);
                }}
              >
                清除
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default CvmKeySelectComponent;
