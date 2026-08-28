-- 多租户默认语言隔离：site_configs 新增 default_lang 字段
-- 对应 change: feat/default-lang-support-multitenant
--
-- universe 多租户模式下，不同租户可配置独立默认语言（zh/en），
-- 影响 HandleSite 返回的 is_overseas、通道可见性过滤（如 slack）、
-- 以及 i18n 语言检测 matcher 的首选语言。
-- 非 universe 模式回退到进程级 --lang 参数。
-- 存量租户的 default_lang 保持为 ''（空值），
-- 在 Go 层 gorm:"default:zh" 写入新行时自动填充，运行时回退到 --lang。

ALTER TABLE `site_configs`
    ADD COLUMN `default_lang` VARCHAR(8) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '租户默认语言：zh 或 en';
