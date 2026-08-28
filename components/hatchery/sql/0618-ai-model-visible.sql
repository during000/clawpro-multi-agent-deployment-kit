-- 解耦 ai_models.enabled 与"用户可见"语义。
--
-- 历史背景：
--   旧版本 enabled=true 同时承担两个职责：
--     1. 模型可用于 LLM 路由（agent → 上游 API）
--     2. 在管控端的"用户可见"开关
--   导致管理员关闭"用户可见"后，存量 agent 也无法使用该模型（bug）。
--
-- 修复方案：
--   - 新增 visible 字段，专门表达"用户端是否可见"
--   - enabled 仅保留"是否可用于 LLM 路由"语义
--   - 用户端列表/绑定接口过滤 (enabled=true AND visible=true)
--   - LLM 代理路由仅过滤 enabled=true（保证存量 agent 继续工作）
--
-- 数据迁移策略（"先对齐后变更"）：
--   1) visible 继承旧 enabled 的值，保留"原本就对用户可见"的配置
--   2) enabled 全部刷为 1，确保所有未删除的存量模型都"可用"
--      （旧版本一旦关掉"用户可见"也意味着"不可用"，对齐后默认全部可用，
--       由管理员后续按需通过新的"启用"开关单独控制）

-- 步骤 1：新增 visible 字段
ALTER TABLE `ai_models` ADD COLUMN `visible` tinyint(1) NOT NULL DEFAULT 0;

-- 步骤 2：visible 继承旧 enabled
UPDATE `ai_models` SET `visible` = `enabled` WHERE `deleted_at` IS NULL;

-- 步骤 3：enabled 统一对齐为 1，保证存量模型都可用
UPDATE `ai_models` SET `enabled` = 1 WHERE `deleted_at` IS NULL;
