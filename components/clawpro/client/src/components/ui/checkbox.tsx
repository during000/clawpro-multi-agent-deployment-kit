import * as React from "react";
import * as CheckboxPrimitive from "@radix-ui/react-checkbox";
import { CheckIcon, MinusIcon } from "lucide-react";

import { cn } from "@/lib/utils";

function Checkbox({
  className,
  checked,
  ...props
}: React.ComponentProps<typeof CheckboxPrimitive.Root>) {
  return (
    <CheckboxPrimitive.Root
      data-slot="checkbox"
      checked={checked}
      className={cn(
        "peer size-4 shrink-0 rounded-[4px] border border-[var(--border-control)] bg-white transition-colors outline-none",
        "hover:border-[#1447E6]",
        "data-[state=checked]:bg-[#355EF1] data-[state=checked]:border-[#1447E6] data-[state=checked]:text-white",
        "data-[state=indeterminate]:bg-[#355EF1] data-[state=indeterminate]:border-[#1447E6] data-[state=indeterminate]:text-white",
        "focus-visible:border-[#1447E6] focus-visible:ring-2 focus-visible:ring-[#355EF1]/20",
        "disabled:cursor-not-allowed disabled:bg-[#f3f3f4] disabled:border-[var(--border-control)] disabled:data-[state=checked]:bg-[#d3d6db] disabled:data-[state=checked]:border-[var(--border-control)]",
        className
      )}
      {...props}
    >
      <CheckboxPrimitive.Indicator
        data-slot="checkbox-indicator"
        className="flex items-center justify-center text-current transition-none"
      >
        {checked === "indeterminate" ? (
          <MinusIcon className="size-3.5" strokeWidth={3} />
        ) : (
          <CheckIcon className="size-3.5" />
        )}
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  );
}

export { Checkbox };
