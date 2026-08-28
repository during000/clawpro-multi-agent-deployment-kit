import { describe, expect, it, vi } from 'vitest';
import { isNewTagExpired } from '../components/onboarding';
import {
  TENANT_SKILL_SQUARE_PUBLIC_POINT_BUBBLE_KEY,
  isTenantSkillSquarePublicGuideActive,
  isTenantSkillSquarePublicNewTagActive,
  shouldCompleteTenantSkillSquarePublicGuide,
} from './tenantSkillSquarePublicOnboarding';

describe('tenantSkillSquarePublicOnboarding', () => {
  it('generates the Point Bubble persistence key', () => {
    expect(TENANT_SKILL_SQUARE_PUBLIC_POINT_BUBBLE_KEY).toBe(
      'onboarding.point-bubble.tenant-skill-square-public-tab-2026-07-30.dismissed',
    );
  });

  it('honors the shared Point Bubble +08:00 activity boundaries', () => {
    expect(isTenantSkillSquarePublicGuideActive(Date.parse('2026-07-15T23:59:59+08:00'))).toBe(false);
    expect(isTenantSkillSquarePublicGuideActive(Date.parse('2026-07-30T00:00:00+08:00'))).toBe(true);
    expect(isTenantSkillSquarePublicGuideActive(Date.parse('2026-08-14T23:59:59+08:00'))).toBe(true);
    expect(isTenantSkillSquarePublicGuideActive(Date.parse('2026-08-15T00:00:00+08:00'))).toBe(false);
  });

  it('keeps the New Tag inside the shared +08:00 activity boundaries', () => {
    expect(isTenantSkillSquarePublicNewTagActive(Date.parse('2026-07-15T23:59:59+08:00'))).toBe(false);
    expect(isTenantSkillSquarePublicNewTagActive(Date.parse('2026-07-30T00:00:00+08:00'))).toBe(true);
    expect(isTenantSkillSquarePublicNewTagActive(Date.parse('2026-08-14T23:59:59+08:00'))).toBe(true);
    expect(isTenantSkillSquarePublicNewTagActive(Date.parse('2026-08-15T00:00:00+08:00'))).toBe(false);
  });

  it('completes the Bubble only after visiting public skills and leaving Skill Square', () => {
    expect(shouldCompleteTenantSkillSquarePublicGuide(false, '/skill-square')).toBe(false);
    expect(shouldCompleteTenantSkillSquarePublicGuide(true, '/skill-square?tab=public')).toBe(false);
    expect(shouldCompleteTenantSkillSquarePublicGuide(true, '/project-collaboration')).toBe(true);
    expect(shouldCompleteTenantSkillSquarePublicGuide(false, '/model-quota')).toBe(false);
  });

  it('expires the New Tag after its 14-day window', () => {
    vi.useFakeTimers();
    try {
      vi.setSystemTime(new Date('2026-08-13T00:00:00+08:00'));
      expect(isNewTagExpired('2026-07-30T00:00:00+08:00', 14)).toBe(false);

      vi.setSystemTime(new Date('2026-08-13T00:00:00.001+08:00'));
      expect(isNewTagExpired('2026-07-30T00:00:00+08:00', 14)).toBe(true);
    } finally {
      vi.useRealTimers();
    }
  });
});
