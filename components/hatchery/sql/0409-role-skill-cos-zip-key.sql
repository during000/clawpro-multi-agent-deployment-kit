-- MR !215: 角色技能新增 cos_zip_key 字段（SMH common space 中的 zip 路径）
ALTER TABLE `open_claw_role_skills`
  ADD COLUMN `cos_zip_key` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER `source`;
