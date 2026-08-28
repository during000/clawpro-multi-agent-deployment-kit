-- 多租户阶段一：site_configs 补齐租户级字段
-- 对应 change: openspec/changes/multi-tenant-universe-mode
--
-- 这五个字段此前作为进程级全局变量或启动参数使用，现统一落到 site_configs 表
-- 供 FixedSnapshot / TenantSnapshot 回读。所有字段均带向前兼容 default。
-- 单租户 Pod 升级后，InitDB 会按保守策略从启动参数回填空字段。

ALTER TABLE `site_configs`
    ADD COLUMN `uin`               VARCHAR(64)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''     COMMENT '租户腾讯云 UIN',
    ADD COLUMN `domain`            VARCHAR(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''     COMMENT '租户对外访问域名',
    ADD COLUMN `internal_secret`   VARCHAR(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''     COMMENT 'Gateway 内部鉴权密钥',
    ADD COLUMN `one_id_account_id` VARCHAR(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''     COMMENT 'OneID 租户账号 ID';
