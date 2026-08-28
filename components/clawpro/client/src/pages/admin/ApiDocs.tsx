// API 文档页面 — 全屏独立页面，不包含管控端左侧菜单栏
// 通过 createPortal 将内容直接渲染到 body，覆盖 AdminLayout

import { useEffect, useState, useRef, useCallback, useMemo } from "react";
import { createPortal } from "react-dom";
import {
  ArrowLeft,
  ChevronRight,
  ChevronDown,
  Book,
  Search,
  Copy,
  Check,
} from "lucide-react";
import {
  navTree,
  overviewSections,
  baseInfo,
  authInfo,
  responseFormat,
  changelogEntries,
  parameterTypes,
  endpointNavId,
  parseEndpointNavId,
  type NavItem,
  type OverviewSection,
  type EndpointDetail as EndpointDetailType,
} from "./apiDocsData";
import { endpointsBySection } from "./apiDocsEndpoints";
import { adminEndpointsBySection } from "./apiDocsEndpointsAdmin";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";

// ─── 合并所有端点数据 ────────────────────────
const allEndpoints: Record<string, EndpointDetailType[]> = {
  ...endpointsBySection,
  ...adminEndpointsBySection,
};

// ─── 全局统一表格列宽 ─────────────────────
// 所有页面表格遵循统一列宽：第1列 180px、第2列 120px、第3列 60px，最后一列自动撑满
const COL_W1 = "w-[180px]";
const COL_W2 = "w-[120px]";
const COL_W3 = "w-[60px]";
const COL_W_API = "w-[45%]"; // 接口名称列（概览表格）

// ─── 接口名称组件（method + path 统一字体渲染）─────────
function MethodPath({
  method,
  path,
  size = "sm",
  className = "",
}: {
  method: string;
  path?: string;
  size?: "sm" | "xs";
  className?: string;
}) {
  return (
    <span
      className={`font-mono truncate ${
        size === "xs" ? "text-[12px]" : "text-[13px]"
      } ${className}`}
    >
      {method.toUpperCase()}
      {path ? ` ${path}` : ""}
    </span>
  );
}

// ─── 代码块组件 ─────────────────────────────
function CodeBlock({ code, language }: { code: string; language?: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <div className="relative group rounded-xl overflow-hidden border border-gray-200/80">
      {language && (
        <div className="px-3 py-1.5 bg-gray-50 border-b border-gray-200/80 text-[11px] text-[#A3A3A3] font-medium uppercase tracking-wider">
          {language}
        </div>
      )}
      <pre className="p-4 bg-[#FAFBFC] text-sm leading-relaxed overflow-x-auto">
        <code className="text-[#0A0A0A] font-mono text-[13px]">{code}</code>
      </pre>
      <button
        onClick={handleCopy}
        className="absolute top-2 right-2 p-1.5 rounded-xl bg-white/80 border border-gray-200/80 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer hover:bg-gray-50"
        title="复制代码"
      >
        {copied ? (
          <Check className="w-3.5 h-3.5 text-green-500" />
        ) : (
          <Copy className="w-3.5 h-3.5 text-[#A3A3A3]" />
        )}
      </button>
    </div>
  );
}

// ─── 左侧导航树节点 ────────────────────────
function NavTreeNode({
  item,
  activeId,
  expandedIds,
  onToggle,
  onSelect,
  depth = 0,
}: {
  item: NavItem;
  activeId: string;
  expandedIds: Set<string>;
  onToggle: (id: string) => void;
  onSelect: (id: string) => void;
  depth?: number;
}) {
  const hasChildren = !!item.children?.length;
  const isExpanded = expandedIds.has(item.id);
  const isActive = activeId === item.id;

  // 判断是否为具体接口节点（endpoint:xxx:METHOD /path）
  const parsed = parseEndpointNavId(item.id);
  const isEndpointItem = !!parsed;

  // 解析 method 和 path 用于展示
  let epMethod = "";
  let epPath = "";
  if (isEndpointItem && parsed) {
    const parts = parsed.endpointName.split(" ");
    epMethod = parts[0];
    epPath = parts.slice(1).join(" ");
  }

  return (
    <div>
      <button
        onClick={() => {
          if (hasChildren) {
            onToggle(item.id);
            // 如果是接口分类节点（有children），点击也切换到该分类页面
            if (!isEndpointItem) onSelect(item.id);
          } else {
            onSelect(item.id);
          }
        }}
        className="w-full flex items-center gap-1 px-3 py-[7px] rounded-xl text-[13px] transition-all duration-150 cursor-pointer"
        style={{
          paddingLeft: depth * 12 + 12,
          background: isActive ? "rgba(20,71,230,0.08)" : "transparent",
          color: isActive ? "#355EF1" : "#374151",
          fontWeight: isActive ? 600 : hasChildren && !isEndpointItem ? 500 : 400,
        }}
        onMouseEnter={(e) => {
          if (!isActive)
            (e.currentTarget as HTMLElement).style.background = "#F3F4F6";
        }}
        onMouseLeave={(e) => {
          if (!isActive)
            (e.currentTarget as HTMLElement).style.background = isActive
              ? "rgba(20,71,230,0.08)"
              : "transparent";
        }}
      >
        {hasChildren && !isEndpointItem ? (
          isExpanded ? (
            <ChevronDown className="w-3.5 h-3.5 shrink-0 text-[#A3A3A3]" />
          ) : (
            <ChevronRight className="w-3.5 h-3.5 shrink-0 text-[#A3A3A3]" />
          )
        ) : (
          <span className="w-3.5 shrink-0" />
        )}
        {isEndpointItem ? (
          <span className="truncate">
            <MethodPath method={epMethod} path={epPath} size="xs" />
          </span>
        ) : (
          <span className="truncate">{item.label}</span>
        )}
      </button>
      {hasChildren && isExpanded && (
        <div>
          {item.children!.map((child) => (
            <NavTreeNode
              key={child.id}
              item={child}
              activeId={activeId}
              expandedIds={expandedIds}
              onToggle={onToggle}
              onSelect={onSelect}
              depth={depth + 1}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// ─── 概览表格组件（2列：接口名称 + 接口功能）────
function OverviewTable({
  section,
  onNavigate,
}: {
  section: OverviewSection;
  onNavigate: (sectionId: string, endpointName: string) => void;
}) {
  return (
    <div className="mb-8">
      <h3 className="text-[15px] font-semibold text-[#0A0A0A] mb-3">
        {section.title}
      </h3>
      <div className="border border-gray-200/80 rounded-xl overflow-hidden">
        <Table className="w-full text-sm">
          <TableHeader>
            <TableRow className="bg-gray-50/80">
              <TableHead className={`text-left uppercase tracking-wider ${COL_W_API}`}>
                接口名称
              </TableHead>
              <TableHead className="text-left uppercase tracking-wider">
                接口功能
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {section.entries.map((entry, idx) => {
              const parts = entry.name.split(" ");
              const method = parts[0];
              const path = parts.slice(1).join(" ");
              return (
                <TableRow
                  key={idx}
                  className="border-t border-gray-200 hover:bg-gray-50/50 transition-colors"
                >
                  <TableCell>
                    <button
                      onClick={() => onNavigate(section.sectionId, entry.name)}
                      className="text-[#355EF1] hover:underline cursor-pointer font-medium"
                    >
                      <MethodPath method={method} path={path} />
                    </button>
                  </TableCell>
                  <TableCell className="text-[#737373] text-[13px] break-words">
                    {entry.description}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

// ─── 接口详情渲染（5段式：描述 / 输入参数 / 输出参数 / 示例 / 错误码）───
function EndpointDetailView({
  endpoint,
}: {
  endpoint: EndpointDetailType;
}) {
  return (
    <div className="space-y-8">
      {/* 标题行 */}
      <div className="flex items-center gap-3">
        <h3 className="text-lg font-semibold text-[#0A0A0A] font-mono">
          {endpoint.method} {endpoint.path}
        </h3>
      </div>

      {/* 1. 接口描述 */}
      <div>
        <h4 className="text-base font-semibold text-[#0A0A0A] mb-3">1. 接口描述</h4>
        <p className="text-sm text-[#737373] leading-relaxed mb-4">
          {endpoint.description}
        </p>
        <div className="border border-gray-200/80 rounded-xl overflow-hidden">
          <Table className="w-full text-sm table-fixed">
            <TableHeader>
              <TableRow className="bg-gray-50/80">
                <TableHead className={`text-left ${COL_W1}`}>属性</TableHead>
                <TableHead className="text-left">值</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow className="border-t border-gray-200">
                <TableCell className="text-[#0A0A0A] font-medium">认证</TableCell>
                <TableCell className="text-[#737373]">{endpoint.auth}</TableCell>
              </TableRow>
              {endpoint.contentType && (
                <TableRow className="border-t border-gray-200">
                  <TableCell className="text-[#0A0A0A] font-medium">Content-Type</TableCell>
                  <TableCell>
                    <code className="text-[13px] bg-gray-50 px-1.5 py-0.5 rounded font-mono text-[#334155]">
                      {endpoint.contentType}
                    </code>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      {/* 2. 输入参数 */}
      <div>
        <h4 className="text-base font-semibold text-[#0A0A0A] mb-3">2. 输入参数</h4>
        {endpoint.inputParams.length > 0 ? (
          <div className="border border-gray-200/80 rounded-xl overflow-hidden">
            <Table className="w-full text-sm table-fixed">
              <TableHeader>
                <TableRow className="bg-gray-50/80">
                  <TableHead className={`text-left ${COL_W1}`}>参数</TableHead>
                  <TableHead className={`text-left ${COL_W2}`}>类型</TableHead>
                  <TableHead className={`text-left ${COL_W3}`}>必填</TableHead>
                  <TableHead className="text-left">说明</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {endpoint.inputParams.map((p, i) => (
                  <TableRow key={i} className="border-t border-gray-200">
                    <TableCell className="font-mono text-[13px] text-[#0A0A0A] break-all">{p.name}</TableCell>
                    <TableCell className="font-mono text-[13px] text-[#737373]">{p.type}</TableCell>
                    <TableCell className="text-[13px] text-[#737373]">{p.required}</TableCell>
                    <TableCell className="text-[13px] text-[#737373] break-words">{p.description}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <p className="text-sm text-[#A3A3A3]">无</p>
        )}
      </div>

      {/* 3. 输出参数 */}
      <div>
        <h4 className="text-base font-semibold text-[#0A0A0A] mb-3">3. 输出参数</h4>
        {endpoint.outputParams.length > 0 ? (
          <div className="border border-gray-200/80 rounded-xl overflow-hidden">
            <Table className="w-full text-sm table-fixed">
              <TableHeader>
                <TableRow className="bg-gray-50/80">
                  <TableHead className={`text-left ${COL_W1}`}>字段</TableHead>
                  <TableHead className={`text-left ${COL_W2}`}>类型</TableHead>
                  <TableHead className="text-left">说明</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {endpoint.outputParams.map((p, i) => (
                  <TableRow key={i} className="border-t border-gray-200">
                    <TableCell className="font-mono text-[13px] text-[#0A0A0A] break-all">{p.name}</TableCell>
                    <TableCell className="font-mono text-[13px] text-[#737373]">{p.type}</TableCell>
                    <TableCell className="text-[13px] text-[#737373] break-words">{p.description}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <p className="text-sm text-[#A3A3A3]">无</p>
        )}
      </div>

      {/* 4. 示例 */}
      <div className="space-y-4">
        <h4 className="text-base font-semibold text-[#0A0A0A]">4. 示例</h4>
        <div>
          <p className="text-xs text-[#737373] mb-1.5 font-medium">请求示例：</p>
          <CodeBlock code={endpoint.requestExample} language="bash" />
        </div>
        <div>
          <p className="text-xs text-[#737373] mb-1.5 font-medium">响应示例：</p>
          <CodeBlock code={endpoint.responseExample} language="json" />
        </div>
      </div>

      {/* 5. 错误码 */}
      {endpoint.errorCodes.length > 0 && (
        <div>
          <h4 className="text-base font-semibold text-[#0A0A0A] mb-3">5. 错误码</h4>
          <div className="border border-gray-200/80 rounded-xl overflow-hidden">
            <Table className="w-full text-sm table-fixed">
              <TableHeader>
                <TableRow className="bg-gray-50/80">
                  <TableHead className={`text-left ${COL_W1}`}>状态码</TableHead>
                  <TableHead className="text-left">错误信息</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {endpoint.errorCodes.map((ec, i) => (
                  <TableRow key={i} className="border-t border-gray-200">
                    <TableCell className="font-mono text-[13px] text-[#0A0A0A]">{ec.code}</TableCell>
                    <TableCell className="text-[13px] text-[#737373] break-words">{ec.error}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      )}
    </div>
  );
}

// ─── 接口分类详情页 ──────────────────────────
function SectionDetailPage({
  section,
  endpoints,
  onNavigateToEndpoint,
}: {
  section: OverviewSection;
  endpoints: EndpointDetailType[];
  onNavigateToEndpoint: (sectionId: string, endpointName: string) => void;
}) {
  return (
    <div className="space-y-8 page-enter">
      <div>
        <h1 className="text-2xl font-bold text-[#0A0A0A] mb-2">
          {section.title}
        </h1>
        <p className="text-sm text-[#737373]">
          共 {section.entries.length} 个接口
        </p>
      </div>

      {/* 概览表格 */}
      <div className="border border-gray-200/80 rounded-xl overflow-hidden">
        <Table className="w-full text-sm">
          <TableHeader>
            <TableRow className="bg-gray-50/80">
              <TableHead className={`text-left uppercase tracking-wider ${COL_W_API}`}>
                接口名称
              </TableHead>
              <TableHead className="text-left uppercase tracking-wider">
                接口功能
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {section.entries.map((entry, idx) => {
              const parts = entry.name.split(" ");
              const method = parts[0];
              const path = parts.slice(1).join(" ");
              return (
                <TableRow
                  key={idx}
                  className="border-t border-gray-200 hover:bg-gray-50/50 transition-colors"
                >
                  <TableCell>
                    <button
                      onClick={() => onNavigateToEndpoint(section.sectionId, entry.name)}
                      className="text-[#355EF1] hover:underline cursor-pointer font-medium"
                    >
                      <MethodPath method={method} path={path} />
                    </button>
                  </TableCell>
                  <TableCell className="text-[#737373] text-[13px] break-words">
                    {entry.description}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

// ─── 主组件 ─────────────────────────────────
export default function ApiDocs() {
  const [activeId, setActiveId] = useState("intro");
  const [expandedIds, setExpandedIds] = useState<Set<string>>(
    () => new Set(["calling", "api"])
  );
  const [navSearchQuery, setNavSearchQuery] = useState("");
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = prevOverflow;
    };
  }, []);

  // ─── 过滤导航树 ──────────────────────────
  const filterNavTree = useCallback(
    (items: NavItem[], query: string): NavItem[] => {
      if (!query.trim()) return items;
      const q = query.toLowerCase();
      return items
        .map((item) => {
          // 检查当前节点 label 是否匹配
          const labelMatch = item.label.toLowerCase().includes(q);
          // 检查 keywords 是否匹配（接口描述、页面关键词等）
          const keywordsMatch = item.keywords?.some((kw) =>
            kw.toLowerCase().includes(q)
          );
          // 递归过滤子节点
          const filteredChildren = item.children
            ? filterNavTree(item.children, query)
            : [];
          // 如果当前节点匹配或有匹配的子节点，则保留
          if (labelMatch || keywordsMatch || filteredChildren.length > 0) {
            return {
              ...item,
              children:
                filteredChildren.length > 0
                  ? filteredChildren
                  : item.children, // 如果自身匹配但子节点无匹配，保留所有子节点
            };
          }
          return null;
        })
        .filter(Boolean) as NavItem[];
    },
    []
  );

  const filteredNavTree = useMemo(
    () => filterNavTree(navTree, navSearchQuery),
    [navSearchQuery, filterNavTree]
  );

  // 搜索时自动展开所有匹配的节点
  useEffect(() => {
    if (navSearchQuery.trim()) {
      const getAllIds = (items: NavItem[]): string[] =>
        items.flatMap((item) => [
          item.id,
          ...(item.children ? getAllIds(item.children) : []),
        ]);
      setExpandedIds(new Set(getAllIds(filteredNavTree)));
    }
  }, [navSearchQuery, filteredNavTree]);

  const handleToggle = useCallback((id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const handleSelect = useCallback((id: string) => {
    setActiveId(id);
    // 如果选中的是单个接口节点，确保其父级分类和 api 组织展开
    const parsed = parseEndpointNavId(id);
    if (parsed) {
      setExpandedIds((prev) => {
        const next = new Set(prev);
        next.add("api");
        next.add(parsed.sectionId);
        return next;
      });
    }
    contentRef.current?.scrollTo({ top: 0, behavior: "smooth" });
  }, []);

  const handleNavigateToEndpoint = useCallback(
    (sectionId: string, endpointName: string) => {
      // 直接导航到具体接口
      const navId = endpointNavId(sectionId, endpointName);
      setActiveId(navId);
      setExpandedIds((prev) => {
        const next = new Set(prev);
        next.add("api");
        next.add(sectionId);
        return next;
      });
      contentRef.current?.scrollTo({ top: 0, behavior: "smooth" });
    },
    []
  );


  // ─── 渲染内容区域 ──────────────────────────
  const renderContent = () => {
    switch (activeId) {
      case "intro":
        return (
          <div className="space-y-6 page-enter">
            <div>
              <h1 className="text-2xl font-bold text-[#0A0A0A] mb-2">简介</h1>
              <p className="text-sm text-[#737373] leading-relaxed">
                腾讯云 ClawPro（Agent 企业版）提供完整的 REST API，支持通过 API Token 进行自动化管理。
              </p>
            </div>
            <div>
              <h2 className="text-base font-semibold text-[#0A0A0A] mb-3">基础信息</h2>
              <div className="border border-gray-200/80 rounded-xl overflow-hidden">
                <Table className="w-full text-sm table-fixed">
                  <TableHeader>
                    <TableRow className="bg-gray-50/80">
                      <TableHead className={`text-left ${COL_W1}`}>项目</TableHead>
                      <TableHead className="text-left">说明</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow className="border-t border-gray-200">
                      <TableCell className="text-[#0A0A0A] font-medium">Base URL</TableCell>
                      <TableCell className="break-words">
                        <code className="text-[13px] bg-gray-50 px-2 py-0.5 rounded text-[#334155] font-mono">{baseInfo.baseUrl}</code>
                      </TableCell>
                    </TableRow>
                    <TableRow className="border-t border-gray-200">
                      <TableCell className="text-[#0A0A0A] font-medium">协议</TableCell>
                      <TableCell className="text-[#737373]">{baseInfo.protocol}</TableCell>
                    </TableRow>
                    <TableRow className="border-t border-gray-200">
                      <TableCell className="text-[#0A0A0A] font-medium">数据格式</TableCell>
                      <TableCell className="text-[#737373]">{baseInfo.dataFormat}</TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        );

      case "changelog":
        return (
          <div className="space-y-8 page-enter">
            <h1 className="text-2xl font-bold text-[#0A0A0A] mb-2">更新历史</h1>
            {changelogEntries.map((release, ri) => (
              <div key={ri} className="border border-gray-200/80 rounded-xl overflow-hidden">
                {/* 版本头 */}
                <div className="bg-gray-50/80 px-5 py-3 border-b border-gray-200/60">
                  <h2 className="text-base font-semibold text-[#0A0A0A]">{release.version}</h2>
                  <p className="text-xs text-[#737373] mt-0.5">发布时间：{release.date}</p>
                </div>
                <div className="px-5 py-4 space-y-4">
                  <p className="text-sm text-[#737373]">{release.summary}</p>
                  {release.sections.map((section, si) => (
                    <div key={si} className="space-y-3">
                      <h3 className="text-sm font-semibold text-[#0A0A0A]">{section.title}：</h3>
                      {/* 接口组织 */}
                      {"groups" in section && section.groups?.map((group, gi) => (
                        <div key={gi} className="ml-1">
                          <p className="text-sm font-medium text-[#334155] mb-1">• {group.name}</p>
                          <ul className="ml-5 space-y-0.5">
                            {group.items.map((item, ii) => (
                              <li key={ii} className="text-sm text-[#737373] font-mono text-[13px] leading-relaxed">
                                {item}
                              </li>
                            ))}
                          </ul>
                        </div>
                      ))}
                      {/* 改善说明 */}
                      {"notes" in section && section.notes?.map((note, ni) => (
                        <p key={ni} className="text-sm text-[#737373] ml-1">• {note}</p>
                      ))}
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        );

      case "overview":
        return (
          <div className="space-y-6 page-enter">
            <div className="mb-2">
              <h1 className="text-2xl font-bold text-[#0A0A0A]">API 概览</h1>
            </div>
            <p className="text-sm text-[#737373] leading-relaxed mb-6">
              以下为平台所有开放 API 的汇总，点击接口名称查看详细文档。
            </p>
            {overviewSections.map((section) => (
              <OverviewTable
                key={section.sectionId}
                section={section}
                onNavigate={handleNavigateToEndpoint}
              />
            ))}
          </div>
        );

      case "request-structure":
        return (
          <div className="space-y-6 page-enter">
            <h1 className="text-2xl font-bold text-[#0A0A0A] mb-2">请求结构</h1>
            <p className="text-sm text-[#737373] leading-relaxed">
              所有 API 请求均基于 HTTPS 协议，使用 RESTful 风格。请求的 Base URL 为：
            </p>
            <CodeBlock code={baseInfo.baseUrl} language="text" />
            <div className="space-y-3">
              <h2 className="text-base font-semibold text-[#0A0A0A]">请求方法</h2>
              <p className="text-sm text-[#737373] leading-relaxed">
                API 支持标准的 HTTP 方法：
              </p>
              <div className="flex gap-3">
                {["GET", "POST", "PUT", "DELETE"].map((m) => (
                  <MethodPath key={m} method={m} className="text-[#737373]" />
                ))}
              </div>
            </div>
            <div className="space-y-3">
              <h2 className="text-base font-semibold text-[#0A0A0A]">请求头</h2>
              <div className="border border-gray-200/80 rounded-xl overflow-hidden">
                <Table className="w-full text-sm table-fixed">
                  <TableHeader>
                    <TableRow>
                      <TableHead className={COL_W1}>Header</TableHead>
                      <TableHead>说明</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow>
                      <TableCell className="font-mono text-[13px] text-[#0A0A0A]">Authorization</TableCell>
                      <TableCell className="text-[#737373] break-words whitespace-normal">Bearer Token 认证（大部分接口必须）</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell className="font-mono text-[13px] text-[#0A0A0A]">Content-Type</TableCell>
                      <TableCell className="text-[#737373] break-words whitespace-normal">
                        <code className="text-[13px] bg-gray-50 px-1.5 py-0.5 rounded">application/json</code>{" "}
                        或{" "}
                        <code className="text-[13px] bg-gray-50 px-1.5 py-0.5 rounded">application/x-www-form-urlencoded</code>
                        （依据接口要求）
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        );

      case "auth":
        return (
          <div className="space-y-6 page-enter">
            <h1 className="text-2xl font-bold text-[#0A0A0A] mb-2">认证方式</h1>
            <p className="text-sm text-[#737373] leading-relaxed">
              所有接口（除公共接口外）均需要认证。API 请求通过 <code className="bg-gray-50 px-1.5 py-0.5 rounded text-[13px]">Authorization</code> 请求头传递 Bearer Token。
            </p>
            <CodeBlock code={authInfo.header} language="bash" />
            <div>
              <h2 className="text-base font-semibold text-[#0A0A0A] mb-3">Token 类型</h2>
              <div className="border border-gray-200/80 rounded-xl overflow-hidden">
                <Table className="w-full text-sm table-fixed">
                  <TableHeader>
                    <TableRow>
                      <TableHead className={COL_W1}>类型</TableHead>
                      <TableHead className={COL_W2}>前缀</TableHead>
                      <TableHead>权限范围</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {authInfo.tokenTypes.map((t, i) => (
                      <TableRow key={i}>
                        <TableCell className="text-[#0A0A0A] font-medium">{t.type}</TableCell>
                        <TableCell className="font-mono text-[#737373]">{t.prefix}</TableCell>
                        <TableCell className="text-[#737373] break-words whitespace-normal">{t.scope}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              <div className="mt-3 flex items-start gap-2 text-sm text-amber-700 bg-amber-50 border border-amber-200/80 rounded-xl px-4 py-3">
                <span className="shrink-0 mt-0.5">⚠️</span>
                <span>{authInfo.note}</span>
              </div>
            </div>
          </div>
        );

      case "response":
        return (
          <div className="space-y-6 page-enter">
            <h1 className="text-2xl font-bold text-[#0A0A0A] mb-2">返回结果</h1>
            <p className="text-sm text-[#737373] leading-relaxed">
              API 统一使用 JSON 格式返回数据。
            </p>
            <div>
              <h2 className="text-base font-semibold text-[#0A0A0A] mb-3">成功响应</h2>
              <CodeBlock code={responseFormat.success} language="json" />
              <p className="text-sm text-[#737373] mt-2">{responseFormat.successNote}</p>
            </div>
            <div>
              <h2 className="text-base font-semibold text-[#0A0A0A] mb-3">失败响应</h2>
              <CodeBlock code={responseFormat.error} language="json" />
            </div>
          </div>
        );

      case "param-types":
        return (
          <div className="space-y-6 page-enter">
            <h1 className="text-2xl font-bold text-[#0A0A0A] mb-2">参数类型</h1>
            <p className="text-sm text-[#737373] leading-relaxed mb-4">
              以下为接口文档中使用的常见参数类型说明。
            </p>
            <div className="border border-gray-200/80 rounded-xl overflow-hidden">
              <Table className="w-full text-sm table-fixed">
                <TableHeader>
                  <TableRow>
                    <TableHead className={COL_W1}>类型</TableHead>
                    <TableHead className={COL_W2}>说明</TableHead>
                    <TableHead>示例</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {parameterTypes.map((pt, i) => (
                    <TableRow key={i}>
                      <TableCell className="font-mono text-[13px] text-[#0A0A0A]">{pt.type}</TableCell>
                      <TableCell className="text-[#737373]">{pt.desc}</TableCell>
                      <TableCell className="font-mono text-[13px] text-[#737373] break-words whitespace-normal">{pt.example}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>
        );

      default: {
        // 先检查是否为单个接口节点
        const parsedEndpoint = parseEndpointNavId(activeId);
        if (parsedEndpoint) {
          const { sectionId, endpointName } = parsedEndpoint;
          const sectionEndpoints = allEndpoints[sectionId] || [];
          const parts = endpointName.split(" ");
          const method = parts[0];
          const path = parts.slice(1).join(" ");
          const ep = sectionEndpoints.find(
            (e) => e.method === method && e.path === path
          );
          const matchedSectionTitle = overviewSections.find(
            (s) => s.sectionId === sectionId
          )?.title;
          if (ep) {
            return (
              <div className="space-y-6 page-enter">
                {/* 面包屑导航 */}
                <div className="flex items-center gap-1.5 text-sm text-[#A3A3A3]">
                  <button
                    onClick={() => handleSelect(sectionId)}
                    className="hover:text-[#355EF1] transition-colors cursor-pointer"
                  >
                    {matchedSectionTitle || sectionId}
                  </button>
                  <ChevronRight className="w-3.5 h-3.5" />
                  <span className="text-[#737373] font-medium">
                    <MethodPath method={ep.method} path={ep.path} />
                  </span>
                </div>
                <EndpointDetailView endpoint={ep} />
              </div>
            );
          }
        }

        // 接口分类详情页面
        const matchedSection = overviewSections.find(
          (s) => s.sectionId === activeId
        );
        if (matchedSection) {
          const endpoints = allEndpoints[activeId] || [];
          return (
            <SectionDetailPage
              section={matchedSection}
              endpoints={endpoints}
              onNavigateToEndpoint={handleNavigateToEndpoint}
            />
          );
        }
        return (
          <div className="text-center py-20 text-[#A3A3A3] text-sm page-enter">
            请从左侧导航选择要查看的内容
          </div>
        );
      }
    }
  };

  return createPortal(
    <div
      className="fixed inset-0 z-[9999] flex flex-col"
      style={{ background: "#FFFFFF" }}
    >
      {/* ─── 顶部导航栏 ───────────────────────── */}
      <header
        className="shrink-0 bg-white border-b border-gray-200 px-6"
      >
        <div className="h-16 flex items-center gap-4 max-w-[1400px] mx-auto">
          <button
            onClick={() => window.close()}
            onAuxClick={() => window.history.back()}
            className="flex items-center gap-1.5 text-sm text-[#525252] hover:text-[#0A0A0A] transition-colors cursor-pointer"
            title="关闭页面"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>返回管控端</span>
          </button>
          <div className="h-4 w-px bg-gray-200" />
          <div className="flex items-center gap-2">
            <div
              className="w-6 h-6 rounded-xl flex items-center justify-center bg-[linear-gradient(90deg,#020617_70%,#355EF1_110%)]"
            >
              <Book className="w-3.5 h-3.5 text-white" />
            </div>
            <h1 className="text-sm font-semibold text-[#0A0A0A]">
              ClawPro API文档
            </h1>
          </div>
          <div className="flex-1" />
          <span className="text-xs text-[#A3A3A3]">v1.0 · 2026-04-07</span>
        </div>
      </header>

      {/* ─── 主体区域 ─────────────────────────── */}
      <div className="flex-1 flex overflow-hidden">
        {/* 左侧导航 */}
        <aside
          className="w-[300px] shrink-0 bg-white border-r border-gray-200 flex flex-col"
        >
          <div className="p-4 border-b border-gray-200">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[#A3A3A3]" />
              <input
                type="text"
                placeholder="搜索文档..."
                value={navSearchQuery}
                onChange={(e) => setNavSearchQuery(e.target.value)}
                className="w-full pl-8 pr-3 py-1.5 text-xs border border-gray-200 rounded-xl bg-gray-50/80 focus:outline-none focus:ring-2 focus:ring-[#355EF1]/20 focus:border-[#1447E6] transition-all placeholder:text-[#A3A3A3]"
              />
            </div>
          </div>
          <nav className="flex-1 overflow-y-auto py-3 px-2">
            {filteredNavTree.map((item) => (
              <NavTreeNode
                key={item.id}
                item={item}
                activeId={activeId}
                expandedIds={expandedIds}
                onToggle={handleToggle}
                onSelect={handleSelect}
              />
            ))}
            {filteredNavTree.length === 0 && navSearchQuery.trim() && (
              <div className="text-center py-8 text-[#A3A3A3] text-xs">
                未找到匹配的文档
              </div>
            )}
          </nav>
        </aside>

        {/* 右侧内容 */}
        <main
          ref={contentRef}
          className="flex-1 overflow-y-auto"
        >
          <div className="max-w-[900px] mx-auto px-10 py-8">
            {renderContent()}
          </div>
        </main>
      </div>
    </div>,
    document.body
  );
}
