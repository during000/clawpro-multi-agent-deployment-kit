/**
 * Alert 组件演示
 * ─────────────────────────────────────────────────────────────────────────────
 * 完整展示 Alert 组件的所有 6 种变体和使用方式
 */

import React from "react";
import {
  Alert,
  AlertTitle,
  AlertDescription,
  AlertInfoIcon,
  AlertOperationInfoIcon,
  AlertSuccessIcon,
  AlertErrorIcon,
  AlertProductNewsIcon,
} from "./alert";
import { CircleAlert } from "lucide-react";

/**
 * Alert 演示组件 - 展示所有变体
 */
export function AlertDemo() {
  return (
    <div className="space-y-6 p-8 bg-gradient-to-b from-gray-50 to-white min-h-screen">
      <div className="max-w-2xl mx-auto">
        <h1 className="text-3xl font-bold mb-2">Alert 通知提示</h1>
        <p className="text-gray-600 mb-8">
          页面级别的统一提示条组件，支持 6 种变体
        </p>

        {/* Info 变体 */}
        <section className="mb-8">
          <h2 className="text-xl font-semibold mb-4">Info（信息提示）</h2>
          <div className="space-y-4">
            <div>
              <p className="text-sm text-gray-600 mb-2">简单提示</p>
              <Alert variant="info">
                <AlertInfoIcon />
                <AlertDescription>
                  数据每 5 分钟更新一次，请等待最新数据显示
                </AlertDescription>
              </Alert>
            </div>

            <div>
              <p className="text-sm text-gray-600 mb-2">带标题和描述</p>
              <Alert variant="info">
                <AlertInfoIcon />
                <AlertTitle>数据提示</AlertTitle>
                <AlertDescription>
                  系统正在同步数据，预计需要 2-3 分钟完成。在此期间，某些功能可能不可用。
                </AlertDescription>
              </Alert>
            </div>
          </div>
        </section>

        {/* Operation-Info 变体 */}
        <section className="mb-8">
          <h2 className="text-xl font-semibold mb-4">
            Operation-Info（操作说明）
          </h2>
          <div className="space-y-4">
            <div>
              <p className="text-sm text-gray-600 mb-2">操作上下文说明</p>
              <Alert variant="operation-info">
                <AlertOperationInfoIcon />
                <AlertTitle>修改提示</AlertTitle>
                <AlertDescription>
                  下方修改将影响所有 Agent，请确认后再保存
                </AlertDescription>
              </Alert>
            </div>

            <div>
              <p className="text-sm text-gray-600 mb-2">简单说明</p>
              <Alert variant="operation-info">
                <AlertOperationInfoIcon />
                <AlertDescription>
                  当前页面仅显示最近 30 天的数据，如需查看更多请使用数据导出功能
                </AlertDescription>
              </Alert>
            </div>
          </div>
        </section>

        {/* Warning 变体 */}
        <section className="mb-8">
          <h2 className="text-xl font-semibold mb-4">Warning（警告提示）</h2>
          <div className="space-y-4">
            <div>
              <p className="text-sm text-gray-600 mb-2">配置缺失警告</p>
              <Alert variant="warning">
                <CircleAlert className="h-4 w-4 text-[var(--alert-warning-icon)]" />
                <AlertTitle>配置不完整</AlertTitle>
                <AlertDescription>
                  有 3 项基础配置未完成（导入企业用户、配置至少一个通道、配置安全组），未完成配置将影响用户端的正常使用。
                </AlertDescription>
              </Alert>
            </div>

            <div>
              <p className="text-sm text-gray-600 mb-2">风险提示</p>
              <Alert variant="warning">
                <CircleAlert className="h-4 w-4 text-[var(--alert-warning-icon)]" />
                <AlertDescription>
                  私有网络（VPC）配额已耗尽，将影响用户端云设备的正常创建与使用
                </AlertDescription>
              </Alert>
            </div>
          </div>
        </section>

        {/* Success 变体 */}
        <section className="mb-8">
          <h2 className="text-xl font-semibold mb-4">
            Success（成功提示）
          </h2>
          <div className="space-y-4">
            <div>
              <p className="text-sm text-gray-600 mb-2">简单成功</p>
              <Alert variant="success">
                <AlertSuccessIcon />
                <AlertDescription>配置保存成功</AlertDescription>
              </Alert>
            </div>

            <div>
              <p className="text-sm text-gray-600 mb-2">详细成功</p>
              <Alert variant="success">
                <AlertSuccessIcon />
                <AlertTitle>操作成功</AlertTitle>
                <AlertDescription>
                  已成功创建 3 个新的 Agent，它们现在可以在工作区中使用
                </AlertDescription>
              </Alert>
            </div>
          </div>
        </section>

        {/* Error 变体 */}
        <section className="mb-8">
          <h2 className="text-xl font-semibold mb-4">Error（错误提示）</h2>
          <div className="space-y-4">
            <div>
              <p className="text-sm text-gray-600 mb-2">简单错误</p>
              <Alert variant="error">
                <AlertErrorIcon />
                <AlertDescription>保存失败，请稍后重试</AlertDescription>
              </Alert>
            </div>

            <div>
              <p className="text-sm text-gray-600 mb-2">详细错误</p>
              <Alert variant="error">
                <AlertErrorIcon />
                <AlertTitle>请求失败</AlertTitle>
                <AlertDescription>
                  网络异常，请检查网络连接后重试。如果问题持续存在，请联系技术支持。
                </AlertDescription>
              </Alert>
            </div>
          </div>
        </section>

        {/* Product-News 变体 */}
        <section className="mb-8">
          <h2 className="text-xl font-semibold mb-4">
            Product-News（产品动态）
          </h2>
          <div className="space-y-4">
            <Alert variant="product-news">
              <AlertProductNewsIcon />
              <AlertDescription>
                【产品动态】OpenClaw v2.4.0 已发布：记忆管理功能上线，支持多轮对话历史管理
              </AlertDescription>
            </Alert>

            <Alert variant="product-news">
              <AlertProductNewsIcon />
              <AlertDescription>
                【产品动态】新增企业级权限管理系统，可更细粒度地控制团队成员的操作权限
              </AlertDescription>
            </Alert>
          </div>
        </section>

        {/* 设计说明 */}
        <section className="mt-12 pt-8 border-t border-gray-200">
          <h2 className="text-xl font-semibold mb-4">设计规范</h2>
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <p className="font-medium text-gray-900">视觉参数</p>
              <ul className="text-gray-600 space-y-1 mt-2">
                <li>• 圆角：4px</li>
                <li>• 内边距：12px 16px</li>
                <li>• 图标：16px</li>
                <li>• 字号：12px</li>
                <li>• 行高：1.5</li>
              </ul>
            </div>
            <div>
              <p className="font-medium text-gray-900">使用原则</p>
              <ul className="text-gray-600 space-y-1 mt-2">
                <li>✓ 常驻页面显示（非自动消失）</li>
                <li>✓ 统一样式和变体选择</li>
                <li>✓ 不要自定义圆角或阴影</li>
                <li>✓ 使用规范图标</li>
                <li>✓ 短文案优先</li>
              </ul>
            </div>
          </div>
        </section>

        {/* 代码示例 */}
        <section className="mt-8 p-6 bg-gray-50 rounded-lg">
          <h3 className="font-semibold mb-3">快速使用</h3>
          <pre className="text-xs text-gray-700 overflow-x-auto">
{`import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";

// Info
<Alert variant="info">
  <AlertInfoIcon />
  <AlertDescription>信息提示</AlertDescription>
</Alert>

// Warning
<Alert variant="warning">
  <CircleAlert />
  <AlertDescription>警告提示</AlertDescription>
</Alert>

// Success
<Alert variant="success">
  <AlertSuccessIcon />
  <AlertDescription>成功提示</AlertDescription>
</Alert>`}
          </pre>
        </section>
      </div>
    </div>
  );
}

export default AlertDemo;
