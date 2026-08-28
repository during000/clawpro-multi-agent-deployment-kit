-- 龙虾医生诊断会话增加独立激活时间，避免 STS 刷新污染超时起算点
-- Target release: Release/2026_07_30

ALTER TABLE `doctor_sessions`
  ADD COLUMN `activated_at` datetime(3) DEFAULT NULL COMMENT '诊断会话进入 active 状态的时间' AFTER `status`;
