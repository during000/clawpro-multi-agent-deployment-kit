-- 0424-webui-apigateway.sql
-- 增加 site_configs.api_gateway_config 字段（WebUI 接入云 API 网关配置）
-- JSON 结构：{"enable": bool, "gateway_instance_id": "ins-xxx", "base_domain": "xxx.com"}
-- 默认 '{}'（即禁用），软功能，不影响主流程
-- 关联 OpenSpec change：webui-apigateway

ALTER TABLE `site_configs`
  ADD COLUMN `api_gateway_config` varchar(1024) COLLATE utf8mb4_unicode_ci DEFAULT '{}' COMMENT 'WebUI 接入云 API 网关配置 JSON，详见 webui-apigateway change';
