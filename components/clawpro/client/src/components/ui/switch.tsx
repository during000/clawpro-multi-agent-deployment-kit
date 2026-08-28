import * as React from "react";
import * as SwitchPrimitive from "@radix-ui/react-switch";

import { cn } from "@/lib/utils";

function Switch({
  className,
  ...props
}: React.ComponentProps<typeof SwitchPrimitive.Root>) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        // 0608：去掉 shadcn 默认 `border-2 border-transparent`（语义上 Switch 没有边框，
        // 用透明 border 充当 padding 容易让业务侧误以为可改边框色）。
        // 改为：root 不要 border，Thumb 通过 `m-0.5` 内缩 2px（垂直/水平 padding 一致）。
        "peer inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors outline-none",
        "data-[state=checked]:bg-[#355EF1] data-[state=unchecked]:bg-[#d3d6db]",
        "hover:data-[state=unchecked]:bg-[#b0b6c3]",
        "focus-visible:ring-2 focus-visible:ring-[#355EF1]/20 focus-visible:ring-offset-2",
        // disabled：弱化为 #EAEEF4（unchecked）/ #C7D7FE（checked），关闭 hover 反馈，不再使用 opacity-50
        "disabled:cursor-not-allowed",
        "disabled:hover:data-[state=unchecked]:bg-[#EAEEF4] data-[disabled]:data-[state=unchecked]:bg-[#EAEEF4]",
        "data-[disabled]:data-[state=checked]:bg-[#C7D7FE]",
        className
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        data-slot="switch-thumb"
        className={cn(
          // m-0.5（4 边各 2px）让 16px Thumb 在 20×36 root 内的 padding 完全一致；
          // checked 时 translate-x-4（16px）= 36 - 16 - 2 - 2，刚好顶到右侧 2px。
          "pointer-events-none block size-4 m-0.5 rounded-full bg-white shadow-sm ring-0 transition-transform data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0"
        )}
      />
    </SwitchPrimitive.Root>
  );
}

export { Switch };
