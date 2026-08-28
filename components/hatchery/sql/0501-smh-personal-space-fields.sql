-- 本迁移为 smh_personal_spaces 表添加两个互相独立的追踪字段：
--   1. env_provision_rev            —— 环境脚本版本追踪，用于存量实例自动重跑 init_smh_env.sh
--   2. last_pushed_token_expires_at —— 当前已下发 token 的过期时间，用于 refreshTokens 跳过仍然有效的 token
-- 两者在业务上彼此独立，仅因为同属一次发版窗口而合并到同一个迁移文件里。

-- ========== 1. env_provision_rev ==========
-- 为 smh_personal_spaces 增加 env_provision_rev 字段，记录实例上 init_smh_env.sh 已经装到的 rev。
-- 配合 controller.CurrentSMHProvisionRev 常量使用：
--   每次 init_smh_env.sh / tencent-agent-storage skill 升级时把常量 +1，
--   定时任务 syncEnvs 会自动为 env_provision_rev < CurrentSMHProvisionRev 的存量实例重跑 init。
-- 存量数据：
--   已有 env_initialized=1 的实例默认保留 env_provision_rev=0，
--   表示它们处于老版本，首次发版后定时任务会自动升级到 rev=1。
ALTER TABLE `smh_personal_spaces`
  ADD COLUMN `env_provision_rev` int NOT NULL DEFAULT 0 AFTER `env_initialized`;

-- 为拆分后的 fresh / upgrade 两类扫描查询提供索引支撑：
--   fresh install 查询：env_initialized=false AND to_be_deleted_at IS NULL
--   upgrade 查询     ：env_initialized=true  AND env_provision_rev < ? AND to_be_deleted_at IS NULL
-- 联合索引以 env_initialized 打头，fresh 查询可命中前缀，upgrade 查询可继续利用 env_provision_rev。
ALTER TABLE `smh_personal_spaces`
  ADD INDEX `idx_smh_personal_spaces_env_sync` (`env_initialized`, `env_provision_rev`);

-- ========== 2. last_pushed_token_expires_at ==========
-- 为 smh_personal_spaces 增加 last_pushed_token_expires_at 字段，记录实例上当前 token 的过期时间。
-- 背景：原本 task.refreshTokens 每 3 小时无条件对所有 RUNNING 实例下发一次 token，
--   token TTL 24h 且 ensurePersonalSpaceToken 有 18h 缓存，绝大多数 TAT 下发的是同一个 token。
-- 改造后：
--   扫描频率提升至 5 分钟（保证关机重开实例快速补发），
--   但下发前先比对本字段与当前时间，剩余时长足够则跳过，真正做到"高频扫描 + 低频下发"。
-- 存量数据：已有实例字段默认 NULL，首次扫描命中 NULL 分支时会触发一次补发，随后进入稳态。
ALTER TABLE `smh_personal_spaces`
  ADD COLUMN `last_pushed_token_expires_at` datetime(3) NULL DEFAULT NULL AFTER `env_provision_rev`;
