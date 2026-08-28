import type { ReactNode } from "react";

import { BodyText, TenantPageTitle } from "@/components/ui/Typography";
import { cn } from "@/lib/utils";

type AdminPageHeaderProps = {
  title: ReactNode;
  description?: ReactNode;
  titleAccessory?: ReactNode;
  actions?: ReactNode;
  className?: string;
  contentClassName?: string;
  titleClassName?: string;
  descriptionClassName?: string;
  actionsClassName?: string;
};

export function AdminPageHeader({
  title,
  description,
  titleAccessory,
  actions,
  className,
  contentClassName,
  titleClassName,
  descriptionClassName,
  actionsClassName,
}: AdminPageHeaderProps) {
  return (
    <div
      data-slot="admin-page-header"
      className={cn(
        "flex items-start justify-between gap-4 flex-wrap mb-6",
        className
      )}
    >
      <div className={cn("min-w-0", actions && "flex-1", contentClassName)}>
        <div className="flex items-center gap-3 flex-wrap">
          <TenantPageTitle className={titleClassName}>{title}</TenantPageTitle>
          {titleAccessory}
        </div>
        {description ? (
          <BodyText tone="secondary" className={cn("mt-1", descriptionClassName)}>
            {description}
          </BodyText>
        ) : null}
      </div>
      {actions ? (
        <div
          className={cn(
            "flex items-center gap-2 flex-wrap shrink-0",
            actionsClassName
          )}
        >
          {actions}
        </div>
      ) : null}
    </div>
  );
}
