import { useEffect, useState } from "react";
import { userStore } from "@/pages/admin/MemberManagement/userStore";

const CURRENT_USER_ID = "alice@acompany.com";
const ALLOW_ALL_KEY = "admin_allow_local_client_access";
const GROUP_RULES_KEY = "admin_local_client_access_group_rules";

interface BooleanPolicyRule {
  id: string;
  groupIds: string[];
  value: boolean;
}

function parseGroupRules(raw: string | null): BooleanPolicyRule[] {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(
      (rule): rule is BooleanPolicyRule =>
        !!rule &&
        typeof rule === "object" &&
        Array.isArray((rule as BooleanPolicyRule).groupIds) &&
        typeof (rule as BooleanPolicyRule).value === "boolean"
    );
  } catch {
    return [];
  }
}

export function isProjectCollaborationAccessAllowed(): boolean {
  const stored = localStorage.getItem(ALLOW_ALL_KEY);
  const allowAll = stored === null ? true : stored === "true";
  if (allowAll) return true;

  const userGroupIds = new Set(
    userStore.getById(CURRENT_USER_ID)?.groupIds ?? []
  );
  return parseGroupRules(localStorage.getItem(GROUP_RULES_KEY)).some(
    (rule) =>
      rule.value && rule.groupIds.some((groupId) => userGroupIds.has(groupId))
  );
}

export function useProjectCollaborationAccessAllowed(): boolean {
  const [allowed, setAllowed] = useState(isProjectCollaborationAccessAllowed);

  useEffect(() => {
    const sync = () => setAllowed(isProjectCollaborationAccessAllowed());
    const handleStorage = (event: StorageEvent) => {
      if (event.key === ALLOW_ALL_KEY || event.key === GROUP_RULES_KEY) sync();
    };

    window.addEventListener("storage", handleStorage);
    const unsubscribeUserStore = userStore.subscribe(sync);
    return () => {
      window.removeEventListener("storage", handleStorage);
      unsubscribeUserStore();
    };
  }, []);

  return allowed;
}
