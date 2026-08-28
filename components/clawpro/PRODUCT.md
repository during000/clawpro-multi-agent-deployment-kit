# Product

## Register

product

## Users

ClawPro serves product operators, tenant users, administrators, and developers who manage AI agents, skills, instances, quotas, security policies, logs, files, and operational settings. Users work in task-heavy product surfaces, usually in authenticated Admin or Tenant flows, where clarity, consistency, density, and predictable controls matter more than decoration.

## Product Purpose

ClawPro provides a product UI for managing AI agent workflows and platform resources across Admin, Tenant, and Landing surfaces. The design system should help teams ship a consistent ClawPro skin across the demo branch and host product repositories, with clear migration rules and portable fallback guidance when the host repository cannot directly reuse demo components.

Success means product frontend teams can select a page, identify its surface type, map host components to ClawPro specs, implement the skin without visual guesswork, and pass QA checks for layout, components, states, accessibility, and design-token consistency.

## Brand Personality

Precise, efficient, trustworthy.

The UI should feel product-grade and workflow-oriented: calm enough for dense operations, sharp enough for technical credibility, and consistent enough that users can focus on the task instead of relearning controls page by page.

## Anti-references

ClawPro should not look like a generic SaaS template, a decorative landing-page concept, a glassmorphism dashboard, or an over-animated AI-generated interface. Avoid old blue-purple brand colors, heavy shadows, arbitrary rounded cards, inconsistent button styles, emoji icons, and page-local component inventions that bypass the portable design pack.

## Design Principles

1. Spec first, aesthetics second: ClawPro portable design pack is the source of truth for tokens, surface layers, component mapping, and fallback rules.
2. Product density with clarity: Admin and Tenant pages may be information-dense, but hierarchy, labels, states, and actions must stay readable.
3. One component vocabulary: repeated tasks should use repeated visual patterns, especially tables, filters, buttons, cards, dialogs, empty states, and batch actions.
4. Preserve surface boundaries: Admin, Tenant, and Landing rules should not be mechanically mixed.
5. Portable by default: every reviewed or migrated component should have a host-repo mapping path or a fallback path.

## Accessibility & Inclusion

Target WCAG AA for core product UI. Body text should meet contrast requirements, focus states must be visible, form errors must be explicit, keyboard navigation should work for controls and overlays, motion must respect reduced-motion preferences, and color should not be the only way to communicate state. Empty, loading, error, disabled, no-permission, and destructive-action states are part of the required user experience.
