import { buildPersistenceKey } from '@/components/onboarding';

export const TENANT_SKILL_SQUARE_PUBLIC_UPDATE = {
  updateId: 'tenant-skill-square-public-tab-2026-07-30',
  releaseDate: '2026-07-30T00:00:00+08:00',
  expiresAt: '2026-08-14T23:59:59+08:00',
  newTagVisibleDays: 14,
} as const;

export const TENANT_SKILL_SQUARE_PUBLIC_POINT_BUBBLE_KEY = buildPersistenceKey(
  'point-bubble',
  TENANT_SKILL_SQUARE_PUBLIC_UPDATE.updateId,
);

export const TENANT_SKILL_SQUARE_PUBLIC_VISITED_EVENT =
  'onboarding:tenant-skill-square-public-visited';

export const TENANT_SKILL_SQUARE_PUBLIC_ANALYTICS = {
  updateId: TENANT_SKILL_SQUARE_PUBLIC_UPDATE.updateId,
  layer: '结构层',
  scenario: '1.5.2',
  endpoint: 'tenant' as const,
};

export function isTenantSkillSquarePublicGuideActive(now = Date.now()): boolean {
  const startsAt = new Date(TENANT_SKILL_SQUARE_PUBLIC_UPDATE.releaseDate).getTime();
  const expiresAt = new Date(TENANT_SKILL_SQUARE_PUBLIC_UPDATE.expiresAt).getTime();

  return now >= startsAt && now <= expiresAt;
}

export function isTenantSkillSquarePublicNewTagActive(now = Date.now()): boolean {
  const startsAt = new Date(TENANT_SKILL_SQUARE_PUBLIC_UPDATE.releaseDate).getTime();
  const expiresAt = new Date(TENANT_SKILL_SQUARE_PUBLIC_UPDATE.expiresAt).getTime();

  return now >= startsAt && now <= expiresAt;
}

let hasVisitedTenantSkillSquarePublic = false;

export function markTenantSkillSquarePublicVisited(): void {
  hasVisitedTenantSkillSquarePublic = true;
}

export function hasTenantSkillSquarePublicBeenVisited(): boolean {
  return hasVisitedTenantSkillSquarePublic;
}

export function clearTenantSkillSquarePublicVisited(): void {
  hasVisitedTenantSkillSquarePublic = false;
}

export function shouldCompleteTenantSkillSquarePublicGuide(
  hasVisitedPublicSkills: boolean,
  pathname: string,
): boolean {
  const path = pathname.split(/[?#]/, 1)[0];
  const isSkillSquare = path === '/skill-square' || path.startsWith('/skill-square/');
  return hasVisitedPublicSkills && !isSkillSquare;
}
