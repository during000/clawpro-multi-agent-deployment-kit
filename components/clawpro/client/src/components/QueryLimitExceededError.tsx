/**
 * QueryLimitExceededError - 查询超限错误态
 *
 * 当用户选的组织关联实例数超出单次查询上限时，替代主内容区显示。
 * 场景：运维观测 / 会话管理 等按组织 + Agent 维度查询的页面。
 *
 * 本组件不提供交互按钮，用文字引导用户调整顶部筛选区。
 */
import { AlertTriangle } from "lucide-react";

export interface QueryLimitExceededErrorProps {
  /** 导致超限的组织名（用于文案高亮） */
  groupName?: string;
}

export function QueryLimitExceededError({ groupName }: QueryLimitExceededErrorProps) {
  return (
    <div className="flex items-center justify-center py-16 px-6">
      <div className="max-w-md w-full bg-white rounded-2xl border border-amber-200 p-6 flex gap-4">
        <div className="flex-shrink-0 w-10 h-10 rounded-full bg-amber-50 border border-amber-100 flex items-center justify-center">
          <AlertTriangle className="w-5 h-5 text-amber-600" />
        </div>
        <div className="flex-1 space-y-2">
          <h3 className="text-sm font-semibold text-gray-900">当前查询范围过大，无法返回结果</h3>
          <p className="text-xs text-gray-600 leading-relaxed">
            {groupName ? (
              <>
                组织「<span className="text-amber-700 font-medium">{groupName}</span>
                」下关联实例数超出单次查询上限。
              </>
            ) : (
              <>当前组织下关联实例数超出单次查询上限。</>
            )}
          </p>
          <div className="pt-1 space-y-1">
            <p className="text-xs text-gray-500">请选择以下任一方式缩小范围：</p>
            <ul className="text-xs text-gray-700 space-y-1 pl-4 list-disc">
              <li>
                在顶部 <span className="text-gray-900 font-medium">Agent</span> 筛选框选择具体 Agent 查询
              </li>
              <li>选择实例数更少的组织</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}
