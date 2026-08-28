import * as React from "react";
import { Heart } from "lucide-react";
import { cn } from "@/lib/utils";

export interface FavoriteButtonProps {
  /** 是否已收藏 */
  isFavorited: boolean;
  /** 点击切换收藏状态 */
  onToggle: () => void;
  /** 自定义 className */
  className?: string;
}

/**
 * 收藏按钮组件（icon 形态）
 *
 * 仅心形图标，用于卡片及详情页（28×28 圆角方块）
 *
 * 色彩规范：
 * - 未收藏：灰色图标（gray-300），hover 变红 + 红色浅底
 * - 已收藏：红色图标（red-500/red-600）+ 红色浅底（red-50），hover 加深（red-100）
 */
function FavoriteButton({
  isFavorited,
  onToggle,
  className,
}: FavoriteButtonProps) {
  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    onToggle();
  };

  return (
    <button
      type="button"
      onClick={handleClick}
      className={cn(
        "w-7 h-7 rounded-lg flex items-center justify-center transition-colors",
        isFavorited
          ? "text-red-500 bg-red-50 hover:bg-red-100"
          : "text-gray-300 hover:text-red-500 hover:bg-red-50",
        className,
      )}
      title={isFavorited ? "取消收藏" : "添加到我的收藏"}
    >
      <Heart className={cn("w-3.5 h-3.5", isFavorited && "fill-current")} />
    </button>
  );
}

export { FavoriteButton };
