-- 新增技能下发成功计数字段
-- 用于技能广场按下载量排序，每次下发成功时原子递增
ALTER TABLE `skills` ADD COLUMN `distribute_count` bigint NOT NULL DEFAULT '0';
