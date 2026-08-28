-- ============================================================
-- 0612-add-memory-query-perf-index.sql
-- 优化记忆管理 overview / instances 列表接口慢查询（DB 扫描行数 200w+）
--
-- 背景：
--   /admin/memory/overview 与 /admin/memory/instances 的 count / 列表查询
--   均为 instances LEFT JOIN memory_tda_iplugins，单条平均扫描 ~206 万行、
--   平均耗时 12~13s，导致 DB 高负载。
--
-- 根因：
--   1) JOIN 列 instance_id 在 memory_tda_iplugins 上无可用索引：
--      唯一索引 idx_memory_tdai_instance_identifier(identifier, instance_id)
--      前导列是 identifier，JOIN 的 ON 子句只带 instance_id，用不上该索引，
--      也没有 instance_id 前导索引 → nested loop 每行全表扫 plugin 表。
--   2) instances 侧 where 过滤组合 (identifier, agent_type, deleted_at)
--      只有单列索引，过滤后仍需大量回表。
--
-- ============================================================

-- 1. memory_tda_iplugins：补 instance_id 前导索引，让 LEFT JOIN 走索引（核心修复）
--    覆盖 ON memory_tda_iplugins.instance_id = ? AND deleted_at IS NULL
--    【已在测试环境 hatchery-mem 验证】加索引后 plugin 表执行计划：
--      key=idx_memory_tda_iplugins_instance_id_deleted_at、ref=instances.instance_id、rows 4→1（N×M → N×1）
ALTER TABLE `memory_tda_iplugins`
  ADD INDEX `idx_memory_tda_iplugins_instance_id_deleted_at` (`instance_id`, `deleted_at`);

-- 2. instances：where 主过滤复合索引 (identifier, agent_type, deleted_at)
--    覆盖 overview / list / count 公共过滤 identifier= + agent_type IN + deleted_at IS NULL。
--    【环境差异】
--      - 测试库 hatchery-mem 为单租户（identifier card=1），优化器拒选此索引（仍走 deleted_at 单列）。
--      - 生产 hatchery 为【多租户】（identifier 多值，最大租户 clp-khzx8pe8 ~2.8w 行 + 多个小租户），
--        identifier 前导列有区分度，可做租户隔离，对中小租户收益显著，应加。
--    上线前建议在生产用 EXPLAIN 对大租户/小租户各验证一次是否命中；
--    若大租户因数据倾斜未命中，可对比顺序 (identifier, deleted_at, agent_type)。
ALTER TABLE `instances`
  ADD INDEX `idx_instances_identifier_agent_type_deleted_at` (`identifier`, `agent_type`, `deleted_at`);
