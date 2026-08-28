# TopNav — 用户端顶部导航组件包

> 设计来源：Figma 「ClawPro 项目设计」-> 公共组件/导航
> 节点：[358:2322](https://www.figma.com/design/1PraDigxMbE8KrBayR1bbb/?node-id=358-2322) / 297:3719

本目录基于 Figma 交互稿 1:1 还原了用户端顶部导航，并按"壳子 + 插槽"的思路拆成了若干可复用组件。如果你正在开发新页面、想接入用户端顶栏、或者需要单独拿其中一块（比如 segmented Tabs / 消息中心）出来用，可以直接使用这里的组件。

## 名词解释

| 名词 | 英文原文 | 大白话解释 |
| --- | --- | --- |
| 顶部导航 | Top Navigation | 网页最顶上那一条横向工具栏，放 Logo / 主菜单 / 用户头像之类的。 |
| Segmented Tabs | Segmented Control | 像"分段开关"那样的几个胶囊按钮，按一下切换内容，被选中的那个会高亮。 |
| 插槽 | Slot | 像积木上预留的孔位，组件留好位置，使用者把自己的东西插进去。 |
| 壳子组件 | Shell / Layout component | 只管"框架"不管"内容"的组件，内容由别人塞进来。 |
| Forward ref | React.forwardRef | 父组件能"穿透"拿到子组件里某个真正 DOM 元素的引用。 |
| Mock 通知 | Mocked data | 假数据，给原型展示用，不是真后台拉的。 |

## 组件清单

| 组件 / 工具 | 作用 | 对应 Figma 节点 |
| --- | --- | --- |
| `TopNav` | 顶栏壳子（高度 64、padding、底分隔线、固定定位） | 297:3719 |
| `NavDivider` | 右侧图标之间的竖向 1px 分隔线 | Vector 5 / 6 / 7 |
| `CenterTabs` | 中央的分段切换 Tab | 297:3468 |
| `NavIconButton` | 右侧的图标 / 图标+文字按钮，带红点能力 | 297:3285 / 363:5028 |
| `HelpIcon` / `BellIcon` / `SwitchAdminIcon` / `ChevronDownIcon` | 内联 SVG 图标，可通过 `currentColor` 跟随父级变色 | 297:3274、297:3275、363:5053、I297:3564;297:3447 |
| `NotificationPanel` | 消息通知（铃铛按钮 + 下拉面板） | 297:3275（图标）+ 自定义面板 |
| `UserMenu` | 用户菜单（圆形头像 + 用户名 + 下拉箭头 + DropdownMenu） | 297:3564 / 297:3460 |

## 快速开始

```tsx
import { useLocation } from "wouter";
import {
  TopNav,
  NavDivider,
  CenterTabs,
  NavIconButton,
  HelpIcon,
  SwitchAdminIcon,
  NotificationPanel,
  UserMenu,
} from "@/components/topnav";
import { DropdownMenuItem } from "@/components/ui/dropdown-menu";

function MyLayout({ children }: { children: React.ReactNode }) {
  const [location, navigate] = useLocation();

  const items = [
    { label: "我的 Agent", value: "/my-openclaw" },
    { label: "技能广场", value: "/skill-square" },
  ];

  return (
    <>
      <TopNav
        center={
          <CenterTabs
            items={items}
            activeValue={location}
            onChange={(v) => navigate(v)}
          />
        }
        right={
          <>
            <NavIconButton icon={<HelpIcon />} title="使用指南" />
            <NavDivider />
            <NotificationPanel notifications={[]} />
            <NavDivider />
            <NavIconButton icon={<SwitchAdminIcon />} label="切换管控端" />
            <NavDivider />
            <UserMenu username="jingsujiang" badge="管理员">
              <DropdownMenuItem>重置密码</DropdownMenuItem>
              <DropdownMenuItem className="text-red-600">退出登录</DropdownMenuItem>
            </UserMenu>
          </>
        }
      />
      <main className="pt-16">{children}</main>
    </>
  );
}
```

## 视觉规范

- **容器**：高 64px、`padding 12px 28px`、`bg #FFFFFF/95%` + backdrop-blur、底边 `1px solid #E2E8F0`
- **Logo**：viewBox 0..120，宽 120px、高 25px；蓝色 `#355EF1` 爪图标 + 黑色字标
- **中央 Tabs**：容器高 39 / `bg #F5F5F5` / radius 4，胶囊项 padding `4px 12px`、radius 3
  - Active：`bg #FFF / color #020617 / shadow 0 1px 2px rgba(0,0,0,.05)`
  - Normal：`color #334155`，hover `#020617`
- **右侧图标按钮**：padding `6px 8px`、radius 4，hover `bg #F1F5F9`
- **未读红点**：`4×4 / #E85C5C`，定位在按钮右上角偏移 6,6
- **分隔线**：宽 1px、高 14px、`bg #E2E8F0`
- **头像**：31×31 圆形、`bg #8CBCF7`，字母 PingFang 600 / 13.3 / `#000`
- **下拉箭头**：14×14、`stroke #020617`

## 设计上的一些权衡

1. **`<img src=svg>` vs 内联 SVG**
   - Logo 因为内含字标，复杂且无需变色，所以直接 `<img>` 引用 SVG 文件（保持像素无损）
   - 三个右侧功能图标和下拉箭头改成了内联 SVG（`NavIcons.tsx`），用 `currentColor` 着色，**好处是 hover 时颜色能自动跟着父按钮变**

2. **红点解耦**
   - Figma 中铃铛 SVG 自带了硬编码红点。这里把红点从 SVG 里抽掉，改用 React 控制 `showDot`，使「是否有未读」可以联动数据

3. **TenantLayout 瘦身**
   - 原 TenantLayout 把通知、用户菜单等所有视觉细节都写在了同一个文件里（~650 行）。
   - 现在 TenantLayout 只负责：路由/角色等业务态接入 + Mock 通知数据 + 菜单项业务文案，视觉细节完全交给 topnav 组件包

## 文件结构

```
components/topnav/
├── TopNav.tsx              # 壳子组件，导出 NavDivider
├── CenterTabs.tsx          # 中央 segmented Tabs
├── NavIconButton.tsx       # 右侧图标按钮（含红点）
├── NavIcons.tsx            # 内联 SVG 图标集
├── NotificationPanel.tsx   # 消息通知（含面板）
├── UserMenu.tsx            # 用户菜单
├── index.ts                # 统一出口
└── README.md               # 本文件
```

素材原始 SVG：`@/assets/topnav/`
