/**
 * Segment 组件完整展示台 — All States & Variants
 * ───────────────────────────────────────────────────────────────────────────
 * 用于本地开发调试，展示 Admin（方角）+ Tenant（胶囊）的全状态组合
 */

import React, { useState } from 'react';
import {
  PortableAdminSegment,
  PortableAdminSegmentItem,
  PortableAdminSegmentGroup,
  PortableAdminSegmentOption,
  PortableTenantSegment,
  PortableTenantSegmentItem,
  PortableTenantSegmentGroup,
  PortableTenantSegmentOption,
} from './segment';

export function SegmentShowcase() {
  // 受控状态
  const [adminTab, setAdminTab] = useState('overview');
  const [tenantFilter, setTenantFilter] = useState('all');

  // 非受控状态
  const [adminMode, setAdminMode] = useState('list');
  const [tenantRange, setTenantRange] = useState('day');

  return (
    <div className="space-y-12 p-8 bg-gray-50 min-h-screen font-sans">
      {/* ════════════════════════════════════════════════════════════════════ */}
      {/* 1. Admin Segment - 受控组件 */}
      {/* ════════════════════════════════════════════════════════════════════ */}

      <section className="space-y-4 bg-white rounded-lg p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900">
          1. Admin Segment（方角 / 受控）
        </h2>
        <p className="text-sm text-gray-600">
          用于卡片内部分区切换（概览 / 详情 / 配置）
        </p>

        {/* 默认状态 */}
        <div className="space-y-2">
          <p className="text-xs font-medium text-gray-500 uppercase">Default / Hover / Active</p>
          <PortableAdminSegment value={adminTab} onValueChange={setAdminTab}>
            <PortableAdminSegmentItem value="overview">
              概览
            </PortableAdminSegmentItem>
            <PortableAdminSegmentItem value="detail">
              详情
            </PortableAdminSegmentItem>
            <PortableAdminSegmentItem value="setting">
              配置
            </PortableAdminSegmentItem>
          </PortableAdminSegment>
          <p className="text-xs text-gray-500 mt-2">
            当前选中: <span className="font-mono font-semibold">{adminTab}</span>
          </p>
        </div>

        {/* 禁用状态 */}
        <div className="space-y-2 pt-4 border-t border-gray-200">
          <p className="text-xs font-medium text-gray-500 uppercase">With Disabled</p>
          <PortableAdminSegment value="a" onValueChange={() => {}}>
            <PortableAdminSegmentItem value="a">
              可用
            </PortableAdminSegmentItem>
            <PortableAdminSegmentItem value="b" disabled>
              禁用项
            </PortableAdminSegmentItem>
            <PortableAdminSegmentItem value="c">
              可用
            </PortableAdminSegmentItem>
          </PortableAdminSegment>
        </div>

        {/* 多项展示 */}
        <div className="space-y-2 pt-4 border-t border-gray-200">
          <p className="text-xs font-medium text-gray-500 uppercase">5 Items (Max Recommended)</p>
          <PortableAdminSegment value={adminTab} onValueChange={setAdminTab}>
            <PortableAdminSegmentItem value="overview">Overview</PortableAdminSegmentItem>
            <PortableAdminSegmentItem value="detail">Details</PortableAdminSegmentItem>
            <PortableAdminSegmentItem value="setting">Settings</PortableAdminSegmentItem>
            <PortableAdminSegmentItem value="access">Access</PortableAdminSegmentItem>
            <PortableAdminSegmentItem value="logs">Logs</PortableAdminSegmentItem>
          </PortableAdminSegment>
        </div>
      </section>

      {/* ════════════════════════════════════════════════════════════════════ */}
      {/* 2. Admin SegmentGroup - 非受控组件 */}
      {/* ════════════════════════════════════════════════════════════════════ */}

      <section className="space-y-4 bg-white rounded-lg p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900">
          2. Admin SegmentGroup（方角 / 非受控）
        </h2>
        <p className="text-sm text-gray-600">
          用于筛选条、模式切换等自管状态场景
        </p>

        {/* 基础用法 */}
        <div className="space-y-2">
          <p className="text-xs font-medium text-gray-500 uppercase">Filter / Mode Toggle</p>
          <PortableAdminSegmentGroup>
            <PortableAdminSegmentOption
              active={adminMode === 'list'}
              onClick={() => setAdminMode('list')}
            >
              列表
            </PortableAdminSegmentOption>
            <PortableAdminSegmentOption
              active={adminMode === 'grid'}
              onClick={() => setAdminMode('grid')}
            >
              网格
            </PortableAdminSegmentOption>
            <PortableAdminSegmentOption
              active={adminMode === 'compact'}
              onClick={() => setAdminMode('compact')}
            >
              紧凑
            </PortableAdminSegmentOption>
          </PortableAdminSegmentGroup>
          <p className="text-xs text-gray-500 mt-2">
            当前模式: <span className="font-mono font-semibold">{adminMode}</span>
          </p>
        </div>

        {/* 筛选场景 */}
        <div className="space-y-2 pt-4 border-t border-gray-200">
          <p className="text-xs font-medium text-gray-500 uppercase">Filter Options</p>
          <PortableAdminSegmentGroup>
            <PortableAdminSegmentOption active={adminMode === 'all'} onClick={() => setAdminMode('all')}>
              全部
            </PortableAdminSegmentOption>
            <PortableAdminSegmentOption active={adminMode === 'active'} onClick={() => setAdminMode('active')}>
              已激活
            </PortableAdminSegmentOption>
            <PortableAdminSegmentOption active={adminMode === 'inactive'} onClick={() => setAdminMode('inactive')}>
              已禁用
            </PortableAdminSegmentOption>
          </PortableAdminSegmentGroup>
        </div>
      </section>

      {/* ════════════════════════════════════════════════════════════════════ */}
      {/* 3. Tenant Segment - 受控组件 */}
      {/* ════════════════════════════════════════════════════════════════════ */}

      <section className="space-y-4 bg-white rounded-lg p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900">
          3. Tenant Segment（胶囊 / 受控）
        </h2>
        <p className="text-sm text-gray-600">
          用于租户端的局部分段切换（outline variant）
        </p>

        {/* 默认状态 */}
        <div className="space-y-2">
          <p className="text-xs font-medium text-gray-500 uppercase">Default / Hover / Active + Outline</p>
          <PortableTenantSegment value={tenantFilter} onValueChange={setTenantFilter}>
            <PortableTenantSegmentItem value="all">
              全部
            </PortableTenantSegmentItem>
            <PortableTenantSegmentItem value="active">
              已激活
            </PortableTenantSegmentItem>
            <PortableTenantSegmentItem value="inactive">
              已禁用
            </PortableTenantSegmentItem>
          </PortableTenantSegment>
          <p className="text-xs text-gray-500 mt-2">
            当前筛选: <span className="font-mono font-semibold">{tenantFilter}</span>
          </p>
        </div>

        {/* 禁用状态 */}
        <div className="space-y-2 pt-4 border-t border-gray-200">
          <p className="text-xs font-medium text-gray-500 uppercase">With Disabled</p>
          <PortableTenantSegment value="x" onValueChange={() => {}}>
            <PortableTenantSegmentItem value="x">
              可用
            </PortableTenantSegmentItem>
            <PortableTenantSegmentItem value="y" disabled>
              禁用项
            </PortableTenantSegmentItem>
            <PortableTenantSegmentItem value="z">
              可用
            </PortableTenantSegmentItem>
          </PortableTenantSegment>
        </div>

        {/* 多项展示 */}
        <div className="space-y-2 pt-4 border-t border-gray-200">
          <p className="text-xs font-medium text-gray-500 uppercase">4 Items</p>
          <PortableTenantSegment value={tenantFilter} onValueChange={setTenantFilter}>
            <PortableTenantSegmentItem value="overview">Overview</PortableTenantSegmentItem>
            <PortableTenantSegmentItem value="settings">Settings</PortableTenantSegmentItem>
            <PortableTenantSegmentItem value="members">Members</PortableTenantSegmentItem>
            <PortableTenantSegmentItem value="audit">Audit Log</PortableTenantSegmentItem>
          </PortableTenantSegment>
        </div>
      </section>

      {/* ════════════════════════════════════════════════════════════════════ */}
      {/* 4. Tenant SegmentGroup - 非受控组件 */}
      {/* ════════════════════════════════════════════════════════════════════ */}

      <section className="space-y-4 bg-white rounded-lg p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900">
          4. Tenant SegmentGroup（胶囊 / 非受控）
        </h2>
        <p className="text-sm text-gray-600">
          用于日期范围、时间周期等选择场景
        </p>

        {/* 日期范围 */}
        <div className="space-y-2">
          <p className="text-xs font-medium text-gray-500 uppercase">Date Range Picker</p>
          <PortableTenantSegmentGroup>
            <PortableTenantSegmentOption
              active={tenantRange === 'day'}
              onClick={() => setTenantRange('day')}
            >
              日
            </PortableTenantSegmentOption>
            <PortableTenantSegmentOption
              active={tenantRange === 'week'}
              onClick={() => setTenantRange('week')}
            >
              周
            </PortableTenantSegmentOption>
            <PortableTenantSegmentOption
              active={tenantRange === 'month'}
              onClick={() => setTenantRange('month')}
            >
              月
            </PortableTenantSegmentOption>
            <PortableTenantSegmentOption
              active={tenantRange === 'year'}
              onClick={() => setTenantRange('year')}
            >
              年
            </PortableTenantSegmentOption>
          </PortableTenantSegmentGroup>
          <p className="text-xs text-gray-500 mt-2">
            当前周期: <span className="font-mono font-semibold">{tenantRange}</span>
          </p>
        </div>

        {/* 视图模式 */}
        <div className="space-y-2 pt-4 border-t border-gray-200">
          <p className="text-xs font-medium text-gray-500 uppercase">View Mode</p>
          <PortableTenantSegmentGroup>
            <PortableTenantSegmentOption
              active={tenantRange === 'card'}
              onClick={() => setTenantRange('card')}
            >
              卡片
            </PortableTenantSegmentOption>
            <PortableTenantSegmentOption
              active={tenantRange === 'list'}
              onClick={() => setTenantRange('list')}
            >
              列表
            </PortableTenantSegmentOption>
            <PortableTenantSegmentOption
              active={tenantRange === 'table'}
              onClick={() => setTenantRange('table')}
            >
              表格
            </PortableTenantSegmentOption>
          </PortableTenantSegmentGroup>
        </div>
      </section>

      {/* ════════════════════════════════════════════════════════════════════ */}
      {/* 5. 对比展示 */}
      {/* ════════════════════════════════════════════════════════════════════ */}

      <section className="space-y-4 bg-white rounded-lg p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900">
          5. Admin vs Tenant 对比
        </h2>
        <p className="text-sm text-gray-600">
          两套端别的视觉差异明确，绝不混用
        </p>

        <div className="grid grid-cols-2 gap-6">
          {/* Admin */}
          <div className="space-y-2">
            <h3 className="text-sm font-semibold text-gray-700">Admin（方角）</h3>
            <div className="p-4 bg-gray-50 rounded border border-gray-200">
              <div className="flex flex-col gap-2">
                <p className="text-xs text-gray-600 mb-2">6px 圆角容器 + 4px 项</p>
                <PortableAdminSegmentGroup>
                  <PortableAdminSegmentOption active onClick={() => {}}>
                    方角
                  </PortableAdminSegmentOption>
                  <PortableAdminSegmentOption active={false} onClick={() => {}}>
                    设计
                  </PortableAdminSegmentOption>
                </PortableAdminSegmentGroup>
              </div>
            </div>
          </div>

          {/* Tenant */}
          <div className="space-y-2">
            <h3 className="text-sm font-semibold text-gray-700">Tenant（胶囊）</h3>
            <div className="p-4 bg-gray-50 rounded border border-gray-200">
              <div className="flex flex-col gap-2">
                <p className="text-xs text-gray-600 mb-2">80px 圆角容器 + 40px 项</p>
                <PortableTenantSegmentGroup>
                  <PortableTenantSegmentOption active onClick={() => {}}>
                    胶囊
                  </PortableTenantSegmentOption>
                  <PortableTenantSegmentOption active={false} onClick={() => {}}>
                    设计
                  </PortableTenantSegmentOption>
                </PortableTenantSegmentGroup>
              </div>
            </div>
          </div>
        </div>

        {/* 特征表 */}
        <div className="overflow-x-auto pt-4">
          <table className="w-full text-sm border-collapse">
            <thead className="bg-gray-100">
              <tr>
                <th className="border border-gray-300 px-3 py-2 text-left font-semibold">特性</th>
                <th className="border border-gray-300 px-3 py-2 text-left font-semibold">Admin</th>
                <th className="border border-gray-300 px-3 py-2 text-left font-semibold">Tenant</th>
              </tr>
            </thead>
            <tbody>
              <tr className="hover:bg-gray-50">
                <td className="border border-gray-300 px-3 py-2 font-medium">容器圆角</td>
                <td className="border border-gray-300 px-3 py-2 font-mono text-xs">6px</td>
                <td className="border border-gray-300 px-3 py-2 font-mono text-xs">80px 胶囊</td>
              </tr>
              <tr className="hover:bg-gray-50">
                <td className="border border-gray-300 px-3 py-2 font-medium">项圆角</td>
                <td className="border border-gray-300 px-3 py-2 font-mono text-xs">4px</td>
                <td className="border border-gray-300 px-3 py-2 font-mono text-xs">40px</td>
              </tr>
              <tr className="hover:bg-gray-50">
                <td className="border border-gray-300 px-3 py-2 font-medium">项 Padding</td>
                <td className="border border-gray-300 px-3 py-2 font-mono text-xs">4px 16px</td>
                <td className="border border-gray-300 px-3 py-2 font-mono text-xs">4px 12px</td>
              </tr>
              <tr className="hover:bg-gray-50">
                <td className="border border-gray-300 px-3 py-2 font-medium">字重（Active）</td>
                <td className="border border-gray-300 px-3 py-2 font-mono text-xs">Semibold（600）</td>
                <td className="border border-gray-300 px-3 py-2 font-mono text-xs">Medium（500）</td>
              </tr>
              <tr className="hover:bg-gray-50">
                <td className="border border-gray-300 px-3 py-2 font-medium">Active Outline</td>
                <td className="border border-gray-300 px-3 py-2 text-gray-600">无</td>
                <td className="border border-gray-300 px-3 py-2 font-mono text-xs">1px #CDD4DC</td>
              </tr>
              <tr className="hover:bg-gray-50">
                <td className="border border-gray-300 px-3 py-2 font-medium">默认文字</td>
                <td className="border border-gray-300 px-3 py-2 font-mono text-xs">#7B818F</td>
                <td className="border border-gray-300 px-3 py-2 font-mono text-xs">#334155</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* ════════════════════════════════════════════════════════════════════ */}
      {/* 6. 使用建议 */}
      {/* ════════════════════════════════════════════════════════════════════ */}

      <section className="space-y-4 bg-blue-50 rounded-lg p-6 border border-blue-200">
        <h2 className="text-lg font-semibold text-blue-900">📋 使用建议</h2>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <h3 className="font-semibold text-green-700">✅ 推荐做法</h3>
            <ul className="text-sm space-y-1 text-gray-700">
              <li>✓ 项数 ≤ 5（超出改用侧导航）</li>
              <li>✓ 提前在路由层确定端别</li>
              <li>✓ 活跃项同时有背景 + 阴影</li>
              <li>✓ 卡片内分区切换用 Segment</li>
              <li>✓ Page header 用 LineTabs（不是 Segment）</li>
            </ul>
          </div>

          <div className="space-y-2">
            <h3 className="font-semibold text-red-700">❌ 禁止做法</h3>
            <ul className="text-sm space-y-1 text-gray-700">
              <li>✗ 混用 Admin 和 Tenant 皮肤</li>
              <li>✗ 在组件内部自己判断端别</li>
              <li>✗ 仅靠颜色微差表示活跃</li>
              <li>✗ 硬塞 > 5 项到 Segment</li>
              <li>✗ 用 className 临时改圆角</li>
            </ul>
          </div>
        </div>
      </section>
    </div>
  );
}
