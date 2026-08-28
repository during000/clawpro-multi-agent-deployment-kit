-- 记忆分组策略表
-- 每一行表示"某个分组被某条策略选中，对应某个 plan"
CREATE TABLE IF NOT EXISTS `memory_plan_group_policies` (
    `id`         INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `group_id`   INT UNSIGNED NOT NULL COMMENT '分组 ID（对应 user_groups.id）',
    `plan`       VARCHAR(16) NOT NULL COMMENT '记忆版本：off / free / pro',
    `priority`   TINYINT NOT NULL DEFAULT 1 COMMENT '策略优先级：1=第一条策略, 2=第二条策略',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY `idx_mpgp_group` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
