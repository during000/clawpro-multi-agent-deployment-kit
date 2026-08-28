import React, { useState, useEffect, useReducer, useCallback } from "react";
import _ from "@/vendor/lodash";
import { Info, X, Search, Loader2, Copy, ChevronDown } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Label } from "@/components/ui/label";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import { Pagination } from "@/components/ui/pagination";

import {
  DescribeMachineRegionList,
  DescribeMachines,
  DescribeLicenseGeneral,
} from "@/pages/admin/Security/api";

import {
  checkMachineIsWindows,
  AUTHORIZE_ROUTE,
  MIX_MACHINE_TYPES,
  buyUrl,
  machineTypes,
  VUL_DEFEND_HOST_TYPES_FLAGSHIP,
  VUL_DEFEND_HOST_TYPES_ANTIEXTORT,
  VUL_DEFEND_HOST_TYPES_ANTIEXTORT_PAYED,
} from "../constants";

import { fixFilters, requestApi } from "./requestApi";
// import CvmKeySelectComponent from "./CvmKeySelectComponent";
import { requestAllPromises } from "./CommonRiskHandleFunc";
import { renderCloudTags } from "./CommonCloudTags";

const fixedLabelWidth = "6.5em";

interface ColumnDef {
  header: string;
  key: string;
  width?: number;
  render?: (item: any) => React.ReactNode;
}

interface CvmSelectComponentProps {
  onChange?: any;
  isEnable?: (record: any) => boolean;
  QuuidList?: string[];
  Uuids?: string[];
  renderColumns?: ColumnDef[];
  renderLeftColumns?: ColumnDef[];
  renderRightColumns?: ColumnDef[];
  showLeftTagColumns?: boolean;
  showRightTagColumns?: boolean;
  showProjectFilter?: boolean;
  showTagFilter?: boolean;
  leftTitle?: React.ReactNode;
  rightTitle?: React.ReactNode;
  beforeChangeMachineList?: (machineList: any[]) => Promise<any[]>;
  cmd?: (params: any) => Promise<any>;
  beforeOnChange?: (keys: any, rows: any) => Promise<any>;
  filter?: any;
  selectedRows?: any[];
  selectedKeys?: any[];
  renderDisabledContent?: (record?: any) => React.ReactNode;
  openSwitch?: boolean;
  layout?: "default" | "fixed";
  getSelectedRowsExtends?: (item?: any) => any;
  isVulDefend?: boolean;
  isNewDnsBlock?: boolean;
  isBlockMode?: string;
  isQrcodeSetting?: boolean;
  isCVM?: boolean;
  setFetchLoading?: any;
  isShowFilterActions?: any;
  projectIds?: any;
  isAllMachineSelectable?: boolean;
  aiAgentHostList: any;
  /**
   * 仅供设计走查/演示使用：
   *   传入后，将跳过真实接口（DescribeMachines / DescribeMachineRegionList / DescribeLicenseGeneral 等）
   *   直接把 mockMachines 作为左侧候选列表渲染。
   *   每一项需要至少包含 Quuid（recordKey）、OpenClawName、ProtectType 等基础字段。
   */
  mockMachines?: any[];
}

const CopyBtn = ({ text }: { text: string }) => (
  <Tooltip>
    <TooltipTrigger asChild>
      <button
        type="button"
        className="inline-flex items-center justify-center ml-1 p-0.5 rounded hover:bg-muted"
        onClick={e => {
          e.stopPropagation();
          navigator.clipboard.writeText(text).then(() => {
            toast.success("复制成功");
          });
        }}
      >
        <Copy className="h-3 w-3 text-muted-foreground" />
      </button>
    </TooltipTrigger>
    <TooltipContent>复制</TooltipContent>
  </Tooltip>
);

export const getNewRows = (
  keys: string | any[],
  selectRows: any[],
  recordKey: string,
  allDevice: any[]
) => {
  let newRows = [];
  const ids = selectRows.map((item: { [x: string]: any }) => item[recordKey]);
  if (keys.length > ids.length) {
    const diff = _.difference(Array.isArray(keys) ? keys : [keys], ids);
    const addList = allDevice.filter((item: { [x: string]: any }) =>
      diff.includes(item[recordKey])
    );
    newRows = selectRows.concat(addList);
  } else {
    newRows = selectRows.filter((item: { [x: string]: any }) =>
      keys?.includes?.(item[recordKey])
    );
  }
  return newRows;
};

export const renderStatusInfo = (
  record: { MachineStatus: string; Uuid: any; InstanceState: string },
  resultStatus = {}
) => {
  let resStatus: any = { ...resultStatus };
  if (Object.keys(resStatus).length === 0) {
    resStatus = {
      isUninstalled: record?.MachineStatus === "UNINSTALLED",
      isOffline: record?.MachineStatus === "OFFLINE",
      isStopped: record?.MachineStatus === "SHUTDOWN",
    };
  }
  if (!record?.Uuid || resStatus.isUninstalled) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <Info className="inline h-3.5 w-3.5 ml-1 -mt-0.5 text-muted-foreground cursor-pointer" />
        </TooltipTrigger>
        <TooltipContent>未安装OpenClaw客户端</TooltipContent>
      </Tooltip>
    );
  }
  if (record?.InstanceState === "TERMINATED_PRO_VERSION") {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <Info className="inline h-3.5 w-3.5 ml-1 -mt-0.5 text-muted-foreground cursor-pointer" />
        </TooltipTrigger>
        <TooltipContent>OpenClaw已销毁</TooltipContent>
      </Tooltip>
    );
  }
  if (resStatus.isOffline) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <Info className="inline h-3.5 w-3.5 ml-1 -mt-0.5 text-muted-foreground cursor-pointer" />
        </TooltipTrigger>
        <TooltipContent>OpenClaw已离线</TooltipContent>
      </Tooltip>
    );
  }
  if (resStatus.isStopped) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <Info className="inline h-3.5 w-3.5 ml-1 -mt-0.5 text-muted-foreground cursor-pointer" />
        </TooltipTrigger>
        <TooltipContent>OpenClaw已关机</TooltipContent>
      </Tooltip>
    );
  }
  return null;
};

export const getRenderColumns = (
  showTagColumns: boolean,
  renderColumns: ColumnDef[],
  renderDisabledContent: any,
  resultStatus: any,
  cmd = DescribeMachines
): ColumnDef[] => {
  const columns: ColumnDef[] = [
    {
      header: "Agent名称/ID",
      key: "OpenClawName",
      render: item => (
        <div>
          <div>{item?.OpenClawName || "-"}</div>
          <div>{item?.MachineExtraInfo?.InstanceID || "-"}</div>
        </div>
      ),
      // render: item => (
      //   <div>
      //     <div
      //       className="machineName-btn-textOverflow"
      //       title={item?.MachineName || item?.InstanceName}
      //     >
      //       {item?.MachineName || item?.InstanceName || "未命名"}
      //     </div>
      //     <div>
      //       {item?.InstanceId ||
      //         item?.MachineExtraInfo?.InstanceID ||
      //         item?.InstanceID ||
      //         "--"}
      //       {(item?.InstanceId ||
      //         item?.MachineExtraInfo?.InstanceID ||
      //         item?.InstanceID) && (
      //           <CopyBtn
      //             text={
      //               item?.InstanceId ||
      //               item?.MachineExtraInfo?.InstanceID ||
      //               item?.InstanceID
      //             }
      //           />
      //         )}
      //       {renderDisabledContent
      //         ? renderDisabledContent(item)
      //         : renderStatusInfo(item, resultStatus(item, cmd))}
      //     </div>
      //   </div>
      // ),
    },
    // {
    //   header: "IP地址",
    //   key: "MachineIp",
    //   width: 135,
    //   render: item => (
    //     <div>
    //       <div>
    //         <span className="newbuy-ip-label">{"公"}</span>
    //         <span className="newbuy-table-text">
    //           {item?.MachineExtraInfo?.WanIP || item?.MachineWanIp || "--"}
    //         </span>
    //       </div>
    //       <div>
    //         <span className="newbuy-ip-label">{"内"}</span>
    //         <span className="newbuy-table-text">
    //           {item?.MachineExtraInfo?.PrivateIP || item?.MachineIp || "--"}
    //         </span>
    //       </div>
    //     </div>
    //   ),
    // },
  ];
  if (showTagColumns) {
    columns.push({
      header: "标签",
      key: "Tag",
      render: item => renderCloudTags(item),
    });
  }
  return [...columns, ...renderColumns];
};

const defaultIsEnable = () => true;

export const getProjectList = (resp: { PermProjects: never[] }) => {
  const list = resp?.PermProjects ?? [];
  const tmplist: { value: any; text: any; tooltip: any }[] = [];
  list.map((item: { ProjectId: any; Name: any }) => {
    tmplist.push({
      value: item?.ProjectId,
      text: item?.Name,
      tooltip: item?.Name,
    });
  });
  return tmplist;
};

export const getSelectedRowsAndKeys = (
  response: any,
  getSelectedRowsExtends: ((item?: any) => any) | undefined
) => {
  const result: any = response || [];
  const newArrRows = result?.map?.((item: any) => {
    let extend = {};
    if (getSelectedRowsExtends) {
      extend = getSelectedRowsExtends(item);
    }
    return {
      ...item,
      MachineIp: item.HostIp,
      MachineName: item.AliasName,
      MachineWanIp: item?.MachineWanIp ?? item?.PublicIp ?? "",
      Tag: item.TagList
        ? item.TagList.map((subItem: any, subIndex: any) => ({
            Rid: subIndex,
            Name: subItem,
            TagId: subIndex,
          }))
        : [],
      ...extend,
    };
  });
  const newArrKeys = result.map((item: any) => item.Quuid);
  return {
    Keys: newArrKeys,
    Rows: newArrRows,
  };
};

/** 搜索框组件 */
const SearchInput = ({
  placeholder,
  disabled,
  onSearch,
  onClear,
  onChange: onChangeProp,
}: {
  placeholder?: string;
  disabled?: boolean;
  onSearch?: (val: string) => void;
  onClear?: () => void;
  onChange?: (val: string) => void;
}) => {
  const [val, setVal] = useState("");
  return (
    <div className="relative">
      <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
      <Input
        className="pl-8 pr-8 h-8"
        placeholder={placeholder}
        disabled={disabled}
        value={val}
        onChange={e => {
          const v = e.target.value;
          setVal(v);
          onChangeProp?.(v);
          if (!v) {
            onClear?.();
          }
        }}
        onKeyDown={e => {
          if (e.key === "Enter") {
            onSearch?.(val);
          }
        }}
      />
      {val && (
        <button
          type="button"
          className="absolute right-2 top-1/2 -translate-y-1/2"
          onClick={() => {
            setVal("");
            onClear?.();
          }}
        >
          <X className="h-3.5 w-3.5 text-muted-foreground" />
        </button>
      )}
    </div>
  );
};

/** 版本筛选下拉 */
const PROTECT_TYPE_ALL = "__all__";

const ProtectTypeFilter = ({
  value,
  options,
  onChange,
}: {
  value: string;
  options: { value: string; text: string }[];
  onChange: (value: string) => void;
}) => (
  <Select
    value={value || PROTECT_TYPE_ALL}
    onValueChange={v => onChange(v === PROTECT_TYPE_ALL ? "" : v)}
  >
    <SelectTrigger className="h-7 w-auto min-w-[80px] text-xs">
      <SelectValue placeholder="全部" />
    </SelectTrigger>
    <SelectContent>
      {options?.map?.((opt: any) => (
        <SelectItem
          key={opt.value || PROTECT_TYPE_ALL}
          value={opt.value || PROTECT_TYPE_ALL}
        >
          {opt.text}
        </SelectItem>
      ))}
    </SelectContent>
  </Select>
);

const CvmSelectComponent = (props: CvmSelectComponentProps) => {
  const {
    layout = "default",
    onChange,
    isEnable = defaultIsEnable,
    QuuidList = undefined,
    Uuids = undefined,
    renderColumns = [],
    showProjectFilter = false,
    showTagFilter = true,
    leftTitle = "",
    rightTitle = "",
    renderLeftColumns = [],
    renderRightColumns = [],
    showLeftTagColumns = true,
    showRightTagColumns = true,
    beforeChangeMachineList = null,
    cmd = DescribeMachines,
    beforeOnChange = null,
    filter = {},
    isNewDnsBlock = undefined,
    isBlockMode = undefined,
    openSwitch = true,
    isVulDefend = false,
    selectedKeys = undefined,
    selectedRows = undefined,
    isQrcodeSetting = undefined,
    isCVM = undefined,
    renderDisabledContent = undefined,
    getSelectedRowsExtends,
    setFetchLoading = undefined,
    isShowFilterActions = true,
    isAllMachineSelectable = false,
    projectIds = [],
    aiAgentHostList,
    mockMachines,
  } = props;
  const recordKey = "Quuid";
  const isMock = !!(mockMachines && mockMachines.length);

  const [deviceArea, setDeviceArea] = useState(
    isQrcodeSetting ? "LH" : isCVM ? "CVM" : "ALL"
  );
  const [deviceRegion, setDeviceRegin] = useState("all-regions");
  const [deviceProjectId, setDeviceProjectId] = useState("");
  const [projectList, setProjectList] = useState([]);
  const [allDevice, setAllDevice] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [selectKeys, setSelectKeys] = useState([] as any);
  const [selectRows, setSelectRows] = useState([] as any);
  const [isCvmTagKey, setIsCvmTagKey] = useState(false);
  const [filterParams, setFilterParams] = useState({} as any);
  const [protectType, setProtectType] = useState(
    isAllMachineSelectable ? "" : "ProtectedMachines"
  );
  const [currentParams, setCurrentParams] = useState({} as any);
  const [rightSearchVal, setRightSearchVal] = useState("");
  const [isSelectAllLoading, setIsSelectAllLoading] = useState(false);
  const [allTypesData, setAllTypesData] = useState([] as any);
  const [machineTypeOptions, setMachineTypeOptions] = useState([] as any);
  const [rightProtectType, setRightProtectType] = useState("");
  const [licenseData, setLicenseData] = useState({} as any);

  const isCvmSelected = deviceArea === "CVM" || deviceArea === "ALL";

  const QueryState = {
    Offset: 0,
    Limit: 10,
    MachineType: isQrcodeSetting ? "LH" : isCVM ? "CVM" : "ALL",
    MachineRegion: "all-regions",
    Filters: { ...filter },
  };

  const reducer = (
    state: { Filters: any },
    action: { type: any; payload: any }
  ) => {
    switch (action.type) {
      case "SET_PAGE":
        return { ...state, ...action.payload };
      case "SET_FILTER":
        return {
          ...state,
          Filters: { ...state.Filters, ...action.payload },
          Offset: 0,
        };
      case "SET_AREA":
        delete (state?.Filters ?? {}).TagId;
        return {
          ...state,
          MachineType: action.payload,
          MachineRegion: "all-regions",
          Offset: 0,
        };
      case "SET_REGION":
        return { ...state, MachineRegion: action.payload, Offset: 0 };
      case "REFRESH":
        return { ...state, ...QueryState };
      default:
        return state;
    }
  };

  const [query, dispatch]: any = useReducer(reducer, QueryState);

  const resultStatus = (item: any, cmd = DescribeMachines) => {
    if (cmd === DescribeMachines) {
      return {
        isUninstalled: item?.AgentStatus === "UNINSTALLED",
        isOffline: item?.AgentStatus === "OFFLINE",
        isStopped: item?.InstanceStatus === "STOPPED",
      };
    }
    return {
      isUninstalled: item?.MachineStatus === "UNINSTALLED",
      isOffline: item?.MachineStatus === "OFFLINE",
      isStopped: item?.MachineStatus === "SHUTDOWN",
    };
  };

  const fetchProjectList = async () => {
    const resp = await requestApi({
      cmd: "DescribePermProject",
      serviceType: "account",
      data: { AllList: "1" },
      version: "2018-12-25",
    });
    const tmplist: any = getProjectList(resp || {});
    setProjectList(tmplist);
  };

  async function fireChange(keys: any) {
    const newRows: any = getNewRows(keys, selectRows, recordKey, allDevice);
    if (beforeOnChange) {
      const result = await beforeOnChange(keys, newRows);
      if (result === false) {
        return;
      }
    }
    setSelectKeys(keys);
    setSelectRows(newRows);
    onChange(keys, newRows, deviceArea, deviceRegion);
  }

  const getMachineTypeAndRegion = async () => {
    const res: any = await DescribeMachineRegionList();
    const maps = res?.RegionList?.reduce(
      (
        pre: { [x: string]: any },
        cur: { MachineType: string; CloudFrom: any; RegionList: any[] }
      ) => {
        pre[
          cur?.MachineType === "Other"
            ? `Other-${cur?.CloudFrom}`
            : cur?.MachineType
        ] = cur?.RegionList?.map?.((d: { [x: string]: any; Region: any }) => ({
          text: d?.RegionName,
          value: d?.Region,
        }));
        return pre;
      },
      {}
    );
    setAllTypesData(maps);
    const tClouds = res?.RegionList?.map?.(
      (item: { MachineType: any }) => item?.MachineType
    );
    const clouds = res?.RegionList?.map?.((item: { CloudFrom: any }) =>
      String(item?.CloudFrom)
    );
    const other = res?.RegionList?.filter?.(
      (item: { CloudFrom: any; MachineType: string }) =>
        String(item?.CloudFrom) === "0" && item?.MachineType === "Other"
    );
    const otherTypes: any = MIX_MACHINE_TYPES.filter(
      item => item?.optionName && clouds?.includes?.(item?.value)
    )?.map?.(item => ({
      text: item?.optionName,
      value: `Other-${item?.value}`,
    }));
    const arr = Object.values(machineTypes)?.filter?.(item =>
      tClouds?.includes?.(item?.value)
    );
    setMachineTypeOptions(
      arr
        ?.concat?.(otherTypes)
        ?.concat?.(
          other?.length ? [{ text: "其他服务器", value: "Other-0" }] : []
        )
    );
  };

  const selectAllMachines = async () => {
    setIsSelectAllLoading(true);
    const limit = 100;
    const res: any = await cmd({ ...currentParams, Offset: 0, Limit: limit });
    let list = res?.Machines || [];
    if (res?.TotalCount > limit) {
      const all = await requestAllPromises(
        res?.TotalCount,
        cmd,
        currentParams,
        "Machines"
      );
      list = list?.concat?.(all);
    }
    if (beforeChangeMachineList) {
      list = await beforeChangeMachineList(list);
    }
    if (!isAllMachineSelectable) {
      list = list?.filter?.(
        (item: { Uuid: any; InstanceState: string }) =>
          item?.Uuid &&
          item?.InstanceState !== "TERMINATED_PRO_VERSION" &&
          !resultStatus(item, cmd)?.isUninstalled &&
          isEnable(item)
      );
    } else {
      list = list?.filter?.((item: any) => isEnable(item));
    }
    const keys = list?.map?.((item: { [x: string]: any }) => item?.[recordKey]);
    setSelectRows(list);
    setSelectKeys(keys);
    onChange?.(keys, list, deviceArea, deviceRegion);
    setIsSelectAllLoading(false);
  };

  const fetchMachine = async (
    tags: any = undefined,
    tagFilters = filterParams,
    offset = 0,
    limit = 10,
    changePage: any = undefined
  ) => {
    setLoading(true);
    setAllDevice([]);

    // === Mock 模式：直接渲染传入的 mockMachines，跳过真实接口 ===
    if (isMock) {
      const list = (mockMachines || []).map((d: any) => ({
        ...d,
        OpenClawName:
          d?.OpenClawName ||
          aiAgentHostList?.find?.(
            (x: any) => x?.InstanceID === d?.MachineExtraInfo?.InstanceID
          )?.OpenClawName,
      }));
      setAllDevice(list as any);
      setTotal(list.length);
      setLoading(false);
      return;
    }

    const currentQuery = _.cloneDeep(query);
    if (deviceProjectId || projectIds?.length > 0) {
      currentQuery.ProjectIds =
        projectIds?.length > 0 ? projectIds : [deviceProjectId];
    }
    if (changePage) {
      currentQuery.Limit = limit;
      currentQuery.Offset = offset;
    }
    if (tagFilters && typeof tagFilters === "object") {
      Object.keys(tagFilters).forEach(item => {
        (currentQuery?.Filters ?? {})[item] = tagFilters[item];
      });
    }
    if (tags) {
      currentQuery.Offset = 0;
      // query.Offset = 0;
      const filterParams: any = { Tags: [] };
      const tagIds = tags
        ?.filter?.((item: { key: string }) => item?.key === "$1")
        ?.map?.((item: { val: any }) => item?.val);
      const tagKeys = tags
        ?.filter?.((item: { key: string }) => item?.key === "$2")
        ?.map?.((item: { val: any }) => item?.val);
      const tagValues = tags
        ?.filter?.(
          (item: { key: string }) => item?.key !== "$1" && item?.key !== "$2"
        )
        ?.map?.((item: { key: any; val: any }) => `${item?.key}$${item?.val}`);
      if (tagIds?.length) {
        filterParams.TagId = tagIds;
        (currentQuery?.Filters ?? {}).TagId = tagIds;
      } else {
        delete filterParams.TagId;
        delete (currentQuery?.Filters ?? {}).TagId;
      }
      if (tagKeys?.length) {
        filterParams.Tags = filterParams.Tags.concat(tagKeys);
      }
      if (tagValues?.length) {
        filterParams.Tags = filterParams.Tags.concat(tagValues);
      }
      if (filterParams.Tags?.length) {
        (currentQuery?.Filters ?? {}).Tags = filterParams.Tags;
      } else {
        delete filterParams.Tags;
      }
      setFilterParams(filterParams);
    }
    if (currentQuery?.MachineType?.indexOf?.("Other-") > -1) {
      currentQuery.Filters.CloudFrom = [
        currentQuery?.MachineType?.split?.("-")?.[1],
      ];
      currentQuery.MachineType = "Other";
    }
    const params = fixFilters(currentQuery);
    setCurrentParams(params);
    const Quuid = params?.Filters?.filter?.((d: any) => d?.Name === 'Quuid')?.[0]?.Values;
    if (Quuid && Quuid?.length > 100) {
      params.Filters = params?.Filters?.filter?.((d: any) => d?.Name !== 'Quuid')?.concat?.({
        Name: 'Quuid',
        Values: Quuid.slice(0, 100),
      });
    }
    const MachineResp: any = await cmd(params);
    console.log(6666677777, MachineResp, params, aiAgentHostList);
    let machineDeviceList = MachineResp?.Machines ?? [];
    if (Quuid?.length > 100) {
      const resp: any = await requestAllPromises(
        Quuid?.length,
        cmd,
        (x, i) => ({
          ...params,
          Filters: params?.Filters?.filter?.((d: any) => d?.Name !== 'Quuid')?.concat?.({
            Name: 'Quuid',
            Values: Quuid?.slice?.(x * 100 * 20 + (i + 1) * 100, x * 100 * 20 + (i + 1) * 100 + 100),
          }),
        }),
        'Machines',
        100,
        0 as any,
      );
      machineDeviceList = machineDeviceList?.concat?.(resp || []);
    }
    machineDeviceList =
      machineDeviceList?.map?.((d: any) => ({
        ...d,
        OpenClawName: aiAgentHostList?.find?.(
          (x: any) => x?.InstanceID === d?.MachineExtraInfo?.InstanceID
        )?.OpenClawName,
      })) ?? [];
    if (beforeChangeMachineList) {
      machineDeviceList = await beforeChangeMachineList(machineDeviceList);
    }
    const isEmpty = aiAgentHostList?.every?.(
      (d: any) => d?.ProtectType !== "Flagship"
    );
    setAllDevice(isEmpty ? [] : (machineDeviceList ?? []));
    setTotal(isEmpty ? 0 : machineDeviceList?.length || 0);
    setLoading(false);
  };

  const fetchSelectedRow = async () => {
    try {
      setFetchLoading?.(true);
      let result = { Rows: [], Keys: [] };
      if (typeof QuuidList?.[0] === "string") {
        const response = await requestApi({
          cmd: "DescribeMachines",
          data: {
            Offset: 0,
            Limit: 100,
            MachineRegion: "all-regions",
            MachineType: "ALL",
            Filters: [{ Name: "Quuid", Values: QuuidList }],
          },
        });
        console.log(3344, response);
        result = getSelectedRowsAndKeys(
          response?.Machines?.map?.((d: any) => ({
            ...d,
            OpenClawName: aiAgentHostList?.find?.(
              (x: any) => x?.InstanceID === d?.MachineExtraInfo?.InstanceID
            )?.OpenClawName,
          })) || [],
          getSelectedRowsExtends
        );
      } else if (typeof Uuids?.[0] === "string") {
        const response = await requestApi({
          cmd: "DescribeMachines",
          data: {
            Offset: 0,
            Limit: 100,
            MachineRegion: "all-regions",
            MachineType: "ALL",
            Uuids,
          },
        });
        result = getSelectedRowsAndKeys(response, getSelectedRowsExtends);
      } else {
        result = getSelectedRowsAndKeys(
          { HostInfoList: QuuidList },
          getSelectedRowsExtends
        );
      }
      setSelectRows(result.Rows);
      setSelectKeys(result.Keys);
      onChange?.(result.Keys, result.Rows, deviceArea, deviceRegion);
      setFetchLoading?.(false);
    } catch (error) {
      setSelectRows([]);
      setSelectKeys([]);
      onChange?.([], [], deviceArea, deviceRegion);
      setFetchLoading?.(false);
    }
  };

  const getLicenseData = async () => {
    const res: any = await DescribeLicenseGeneral();
    setLicenseData(res || {});
  };

  const onPageChange = (pageIndex: number) => {
    const pageSize = query.Limit;
    if (isCvmSelected && isCvmTagKey) {
      // query.Offset = (pageIndex - 1) * pageSize;
      // query.Limit = pageSize;
      fetchMachine(
        undefined,
        filterParams,
        (pageIndex - 1) * pageSize,
        pageSize,
        1
      );
    } else {
      dispatch({
        type: "SET_PAGE",
        payload: { Offset: (pageIndex - 1) * pageSize, Limit: pageSize },
      });
    }
  };

  function cancelAllSelected() {
    setSelectKeys([]);
    setSelectRows([]);
    onChange([], [], "ALL", "all-regions");
  }

  const leftColumns = getRenderColumns(
    showLeftTagColumns,
    [...renderColumns, ...renderLeftColumns],
    renderDisabledContent,
    resultStatus,
    cmd
  );

  const rightColumns = getRenderColumns(
    showRightTagColumns,
    [...renderColumns, ...renderRightColumns],
    renderDisabledContent,
    resultStatus,
    cmd
  );

  // 判断行是否禁用
  const isRowDisabled = useCallback(
    (record: any) =>
      (!isAllMachineSelectable && !record?.Uuid) ||
      (!isAllMachineSelectable &&
        record?.InstanceState === "TERMINATED_PRO_VERSION") ||
      (!isAllMachineSelectable && resultStatus(record, cmd)?.isUninstalled) ||
      !isEnable(record) ||
      !openSwitch ||
      isSelectAllLoading ||
      (checkMachineIsWindows(record) && isNewDnsBlock),
    [
      isAllMachineSelectable,
      isEnable,
      openSwitch,
      isSelectAllLoading,
      isNewDnsBlock,
      cmd,
    ]
  );

  // 渲染左侧 checkbox 带提示
  const renderLeftCheckbox = (
    record: any,
    checked: boolean,
    onToggle: () => void
  ) => {
    const disabled = isRowDisabled(record);

    // 特殊提示条件：漏洞防御 / 拦截策略 / 命令策略
    if (
      (isVulDefend || isNewDnsBlock || isBlockMode) &&
      record?.ProtectType !== "Flagship" &&
      (isNewDnsBlock || isBlockMode === "2" || isVulDefend)
    ) {
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Checkbox disabled checked={false} />
            </span>
          </TooltipTrigger>
          <TooltipContent className="max-w-[200px]">
            {isNewDnsBlock || isBlockMode === "2" ? (
              <div>
                <span>拦截策略仅对旗舰版机器生效，可</span>
                <a
                  onClick={() => window.open(AUTHORIZE_ROUTE)}
                  className="ml-1 cursor-pointer text-primary underline"
                >
                  点击升级版本
                </a>
              </div>
            ) : (
              <div>
                <span>漏洞防御功能仅支持旗舰版OpenClaw，点击 </span>
                <a
                  onClick={() => window.open(AUTHORIZE_ROUTE)}
                  className="ml-1 cursor-pointer text-primary underline"
                >
                  升级旗舰版
                </a>
                <span>，即可开启防御</span>
              </div>
            )}
          </TooltipContent>
        </Tooltip>
      );
    }

    // Windows 不支持拦截
    if (checkMachineIsWindows(record) && isNewDnsBlock) {
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Checkbox disabled checked={false} />
            </span>
          </TooltipTrigger>
          <TooltipContent className="max-w-[200px]">
            Windows OpenClaw暂不支持拦截
          </TooltipContent>
        </Tooltip>
      );
    }

    // 基础版提示升级
    if (
      !isVulDefend &&
      !isNewDnsBlock &&
      !isBlockMode &&
      !isAllMachineSelectable &&
      (!isEnable(record) ||
        filter?.Version === "ProtectedMachines" ||
        filter?.Version?.[0] === "ProtectedMachines") &&
      record?.ProtectType === "BASIC_VERSION"
    ) {
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Checkbox disabled checked={false} />
            </span>
          </TooltipTrigger>
          <TooltipContent className="max-w-[250px]">
            <div>
              <h3 className="text-sm font-semibold cwp-upgrade-icon">
                升级防护，可使用此功能
              </h3>
              <div className="my-2 text-foreground/70">
                {`该OpenClaw为基础版防护，暂不支持此功能，可点击${licenseData?.AvailableLicenseCnt > 0 ? "绑定授权" : "升级版本"}，体验功能。`}
              </div>
              <div className="text-right">
                <a
                  className="text-orange-500"
                  href={
                    licenseData?.AvailableLicenseCnt > 0
                      ? AUTHORIZE_ROUTE
                      : buyUrl
                  }
                  target="_blank"
                >
                  {licenseData?.AvailableLicenseCnt > 0
                    ? "绑定授权"
                    : "升级版本"}
                </a>
              </div>
            </div>
          </TooltipContent>
        </Tooltip>
      );
    }

    return (
      <Checkbox
        checked={checked}
        disabled={disabled}
        onCheckedChange={() => {
          if (!disabled) onToggle();
        }}
      />
    );
  };

  // 全选 checkbox
  const allSelectableKeys = (allDevice || [])
    .filter((r: any) => !isRowDisabled(r))
    .map((r: any) => r[recordKey]);
  const allChecked =
    allSelectableKeys.length > 0 &&
    allSelectableKeys.every((k: string) => selectKeys.includes(k));
  const someChecked =
    !allChecked &&
    allSelectableKeys.some((k: string) => selectKeys.includes(k));

  const handleSelectAll = () => {
    if (allChecked) {
      // 取消当前页的全选
      const remaining = selectKeys.filter(
        (k: string) => !allSelectableKeys.includes(k)
      );
      fireChange(remaining);
    } else {
      // 选中当前页所有可选项
      const merged = _.uniq([...selectKeys, ...allSelectableKeys]);
      fireChange(merged);
    }
  };

  // 右侧过滤后的数据
  const filteredRightRows =
    !rightSearchVal && !rightProtectType
      ? selectRows || []
      : rightSearchVal && rightProtectType
        ? selectRows
            ?.filter?.((item: any) => item?.ProtectType === rightProtectType)
            ?.filter?.(
              (item: {
                MachineName: string | string[];
                InstanceId: string | string[];
                InstanceID: string | string[];
                MachineIp: string | string[];
                MachineWanIp: string | string[];
              }) =>
                item?.MachineName?.indexOf?.(rightSearchVal) > -1 ||
                item?.InstanceId?.indexOf?.(rightSearchVal) > -1 ||
                item?.InstanceID?.indexOf?.(rightSearchVal) > -1 ||
                item?.MachineIp?.indexOf?.(rightSearchVal) > -1 ||
                item?.MachineWanIp?.indexOf?.(rightSearchVal) > -1
            )
        : rightSearchVal && !rightProtectType
          ? selectRows?.filter?.(
              (item: {
                MachineName: string | string[];
                InstanceId: string | string[];
                InstanceID: string | string[];
                MachineIp: string | string[];
                MachineWanIp: string | string[];
              }) =>
                item?.MachineName?.indexOf?.(rightSearchVal) > -1 ||
                item?.InstanceId?.indexOf?.(rightSearchVal) > -1 ||
                item?.InstanceID?.indexOf?.(rightSearchVal) > -1 ||
                item?.MachineIp?.indexOf?.(rightSearchVal) > -1 ||
                item?.MachineWanIp?.indexOf?.(rightSearchVal) > -1
            )
          : selectRows?.filter?.(
              (item: { ProtectType: string }) =>
                item?.ProtectType === rightProtectType
            );

  // 获取左侧筛选选项
  const getLeftFilterOptions = () => {
    if (isNewDnsBlock || isBlockMode === "2" || isVulDefend) {
      return VUL_DEFEND_HOST_TYPES_FLAGSHIP;
    }
    if (isBlockMode === "1" || !isBlockMode) {
      return VUL_DEFEND_HOST_TYPES_ANTIEXTORT_PAYED;
    }
    return VUL_DEFEND_HOST_TYPES_ANTIEXTORT;
  };

  useEffect(() => {
    if (isMock) {
      // mock 模式下：跳过 region / license / project 等真实接口
      setIsSelectAllLoading(false);
      return;
    }
    getMachineTypeAndRegion();
    setIsSelectAllLoading(false);
    if (showProjectFilter) {
      // fetchProjectList();
    }
    getLicenseData();
  }, []);

  useEffect(() => {
    if (selectedKeys) {
      setSelectKeys(selectedKeys);
    }
  }, [selectedKeys]);

  useEffect(() => {
    if (selectedRows) {
      setSelectRows(selectedRows);
    }
  }, [selectedRows]);

  useEffect(() => {
    fetchMachine();
  }, [query, deviceProjectId]);

  useEffect(() => {
    if (QuuidList?.length || Uuids?.length) {
      try {
        fetchSelectedRow();
      } catch (error) {
        setSelectRows([]);
        setSelectKeys([]);
      }
    }
    if (QuuidList?.length === 0) {
      setSelectKeys([]);
      setSelectRows([]);
    }
  }, [QuuidList, Uuids]);

  return (
    <span>
      {/* Transfer 穿梭框 */}
      <div className="grid grid-cols-2 gap-4 pt-5">
        {/* 左侧面板 - 选择OpenClaw */}
        <div className="border rounded-[4px] flex flex-col">
          {/* 标题栏 */}
          <div className="flex items-center justify-between px-3 py-2 border-b bg-muted/30">
            <strong className="text-sm">{leftTitle || "选择OpenClaw"}</strong>
          </div>
          {/* 表格 */}
          <ScrollArea
            className="flex-1"
            style={{ maxHeight: 330, minHeight: 330 }}
          >
            {loading ? (
              <div className="flex items-center justify-center h-full min-h-[200px]">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : (allDevice || []).length === 0 ? (
              <div className="flex items-center justify-center h-full min-h-[200px] text-[var(--text-weak)] text-sm">
                暂无数据
              </div>
            ) : (
              <Table className="cvm-select-table">
                <TableHeader>
                  <TableRow>
                    <TableHead
                      className="w-10"
                      style={{ cursor: !openSwitch ? "not-allowed" : "" }}
                    >
                      {openSwitch && (
                        <Checkbox
                          checked={
                            allChecked
                              ? true
                              : someChecked
                                ? "indeterminate"
                                : false
                          }
                          onCheckedChange={handleSelectAll}
                        />
                      )}
                    </TableHead>
                    {leftColumns.map(col => (
                      <TableHead
                        key={col.key}
                        style={{
                          cursor: !openSwitch ? "not-allowed" : "",
                        }}
                      >
                        {/* {col.key === "ProtectType" &&
                          (isVulDefend || isNewDnsBlock || isBlockMode) &&
                          openSwitch ? (
                          <ProtectTypeFilter
                            value={protectType}
                            options={getLeftFilterOptions()}
                            onChange={value => {
                              setProtectType(value);
                              dispatch({
                                type: "SET_FILTER",
                                payload: { Version: value === "" ? "" : [value] },
                              });
                            }}
                          />
                        ) : ( */}
                        {col.header}
                        {/* )} */}
                      </TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(allDevice?.slice?.(query.Offset, query.Offset + query.Limit) || []).map((record: any, idx: number) => {
                    const key = record[recordKey];
                    const checked = selectKeys.includes(key);
                    const disabled = isRowDisabled(record);
                    return (
                      <TableRow
                        key={key || idx}
                        className={`${disabled ? "opacity-50" : "cursor-pointer"}`}
                        onClick={() => {
                          if (!disabled) {
                            if (checked) {
                              fireChange(
                                selectKeys.filter((k: string) => k !== key)
                              );
                            } else {
                              fireChange([...selectKeys, key]);
                            }
                          }
                        }}
                      >
                        <TableCell className="w-10">
                          {renderLeftCheckbox(record, checked, () => {
                            if (checked) {
                              fireChange(
                                selectKeys.filter((k: string) => k !== key)
                              );
                            } else {
                              fireChange([...selectKeys, key]);
                            }
                          })}
                        </TableCell>
                        {leftColumns.map(col => (
                          <TableCell key={col.key}>
                            {col.render ? col.render(record) : record[col.key]}
                          </TableCell>
                        ))}
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            )}
          </ScrollArea>
          {/* 底部分页 & 提示 */}
          <div className="px-3 py-2 border-t space-y-1" style={{ paddingTop: 0, paddingBottom: 0 }}>
            {openSwitch && (
              <Pagination
                total={total}
                current={query.Offset / query.Limit + 1}
                pageSize={10}
                size="small"
                showTotal={(t) => `共 ${t} 条记录`}
                className="w-full justify-between"
                onChange={(p) => onPageChange(p)}
              />
            )}
          </div>
        </div>

        {/* 右侧面板 - 已选择OpenClaw */}
        <div className="border rounded-[4px] flex flex-col">
          {/* 标题栏 */}
          <div className="flex items-center justify-between px-3 py-2 border-b bg-muted/30">
            <strong className="text-sm">
              {rightTitle || `已选择OpenClaw（${selectKeys?.length || 0}）`}
            </strong>
            <Button
              variant="link"
              size="sm"
              className="h-auto p-0"
              disabled={!selectKeys?.length || isSelectAllLoading}
              onClick={() => cancelAllSelected()}
            >
              清空选择
            </Button>
          </div>
          {/* 搜索栏 */}
          {/* <div className="px-3 py-2 border-b">
            <SearchInput
              placeholder="请输入OpenClaw名称/实例ID/IP地址进行搜索"
              onChange={val => setRightSearchVal(val?.trim?.() || "")}
              onSearch={val => setRightSearchVal(val?.trim?.() || "")}
              onClear={() => setRightSearchVal("")}
            />
          </div> */}
          {/* 表格 */}
          <ScrollArea
            className="flex-1"
            style={{ maxHeight: 380, minHeight: 330 }}
          >
            {(filteredRightRows || []).length === 0 ? (
              <div className="flex items-center justify-center h-full min-h-[200px] text-[var(--text-weak)] text-sm">
                暂无数据
              </div>
            ) : (
              <Table className="cvm-select-table">
                <TableHeader>
                  <TableRow>
                    {rightColumns.map(col => (
                      <TableHead key={col.key} style={{ width: col.width }}>
                        {col.key === "ProtectType" && isVulDefend ? (
                          <ProtectTypeFilter
                            value={rightProtectType}
                            options={VUL_DEFEND_HOST_TYPES_ANTIEXTORT}
                            onChange={value => setRightProtectType(value)}
                          />
                        ) : (
                          col.header
                        )}
                      </TableHead>
                    ))}
                    <TableHead className="w-10" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(filteredRightRows || []).map((record: any, idx: number) => {
                    const key = record[recordKey];
                    return (
                      <TableRow key={key || idx}>
                        {rightColumns.map(col => (
                          <TableCell key={col.key} style={{ width: col.width }}>
                            {col.render ? col.render(record) : record[col.key]}
                          </TableCell>
                        ))}
                        <TableCell className="w-12">
                          <button
                            type="button"
                            disabled={!openSwitch}
                            className="p-0.5 rounded hover:bg-muted disabled:opacity-50"
                            onClick={() =>
                              fireChange(
                                selectKeys.filter((i: string) => i !== key)
                              )
                            }
                          >
                            <X className="h-4 w-4 text-muted-foreground" />
                          </button>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            )}
          </ScrollArea>
        </div>
      </div>
    </span>
  );
};

export default CvmSelectComponent;
