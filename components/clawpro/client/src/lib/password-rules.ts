/**
 * password-rules.ts - 密码强度规则（共享）
 *
 * 与"落地页 → 立即创建 → 首次登录重置密码"（client/src/components/SsoLoginDialog.tsx
 * 的 ResetPasswordView）保持完全一致，便于全站口径统一。
 *
 * - 长度：8 - 16
 * - 必须同时包含：大写字母、小写字母、数字、特殊符号
 * - 错误文案：返回首条不满足的规则文案（用于行内红字）；通过则返回 null
 *
 * 注意：管控端「重置密码」的"最近 N 个密码黑名单"特意不引入此处，
 * 由调用方按需自行处理（管理员替改场景默认不启用）。
 */

/** 密码强度规范说明（用于 ⓘ tooltip） */
export const PASSWORD_RULES_HINT =
  "密码长度 8-16 位，必须同时包含大写字母、小写字母、数字和特殊符号";

/** 密码长度上限（与 input maxLength 对齐） */
export const PASSWORD_MAX_LENGTH = 16;

/**
 * 密码强度校验：返回首条不满足的规则文案；通过返回 null
 *
 * 与 SsoLoginDialog.tsx 中的 validatePasswordStrength 行为完全一致。
 */
export function validatePasswordStrength(pwd: string): string | null {
  if (pwd.length < 8 || pwd.length > 16) {
    return "密码长度需为 8-16 位";
  }
  if (!/[A-Z]/.test(pwd)) {
    return "密码必须包含大写字母";
  }
  if (!/[a-z]/.test(pwd)) {
    return "密码必须包含小写字母";
  }
  if (!/\d/.test(pwd)) {
    return "密码必须包含数字";
  }
  // 特殊符号：除字母数字外的可见 ASCII 字符
  if (!/[!-/:-@[-`{-~]/.test(pwd)) {
    return "密码必须包含特殊符号";
  }
  return null;
}
