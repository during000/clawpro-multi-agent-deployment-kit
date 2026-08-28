-- Add tenant-scoped indexes for GET /admin/audit user_id, username, and resource_id filters.
-- The existing standalone user_id index remains available for non-tenant lookups.

SET @audit_user_id_index_exists = (
  SELECT COUNT(1)
  FROM `information_schema`.`statistics`
  WHERE `table_schema` = DATABASE()
    AND `table_name` = 'audit_logs'
    AND `index_name` = 'idx_audit_logs_identifier_user_id'
);
SET @audit_user_id_index_sql = IF(
  @audit_user_id_index_exists = 0,
  'ALTER TABLE `audit_logs` ADD INDEX `idx_audit_logs_identifier_user_id` (`identifier`, `user_id`)',
  'SELECT 1'
);
PREPARE audit_user_id_index_stmt FROM @audit_user_id_index_sql;
EXECUTE audit_user_id_index_stmt;
DEALLOCATE PREPARE audit_user_id_index_stmt;

SET @audit_username_index_exists = (
  SELECT COUNT(1)
  FROM `information_schema`.`statistics`
  WHERE `table_schema` = DATABASE()
    AND `table_name` = 'audit_logs'
    AND `index_name` = 'idx_audit_logs_identifier_username'
);
SET @audit_username_index_sql = IF(
  @audit_username_index_exists = 0,
  'ALTER TABLE `audit_logs` ADD INDEX `idx_audit_logs_identifier_username` (`identifier`, `username`)',
  'SELECT 1'
);
PREPARE audit_username_index_stmt FROM @audit_username_index_sql;
EXECUTE audit_username_index_stmt;
DEALLOCATE PREPARE audit_username_index_stmt;

SET @audit_resource_id_index_exists = (
  SELECT COUNT(1)
  FROM `information_schema`.`statistics`
  WHERE `table_schema` = DATABASE()
    AND `table_name` = 'audit_logs'
    AND `index_name` = 'idx_audit_logs_identifier_resource_id'
);
SET @audit_resource_id_index_sql = IF(
  @audit_resource_id_index_exists = 0,
  'ALTER TABLE `audit_logs` ADD INDEX `idx_audit_logs_identifier_resource_id` (`identifier`, `resource_id`)',
  'SELECT 1'
);
PREPARE audit_resource_id_index_stmt FROM @audit_resource_id_index_sql;
EXECUTE audit_resource_id_index_stmt;
DEALLOCATE PREPARE audit_resource_id_index_stmt;
