import { cn } from "@/lib/utils";

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      className={cn("bg-[#f3f3f4] animate-pulse rounded-[4px]", className)}
      {...props}
    />
  );
}

export { Skeleton };
