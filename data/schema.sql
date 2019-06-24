
DROP TABLE IF EXISTS `dandelion_app_configs`;
CREATE TABLE `dandelion_app_configs` (
  `id` BIGINT(12) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `app_id` VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'app id',
  `status` TINYINT(1) NOT NULL DEFAULT '0' COMMENT '0: diabled, 1: enabled',
  `version` VARCHAR(16) NOT NULL DEFAULT '',
  `host` VARCHAR(32) NOT NULL DEFAULT '',
  `instance_id` VARCHAR(32) NOT NULL DEFAULT '',
  `commit_id` CHAR(40) NOT NULL DEFAULT '',
  `md5sum` CHAR(32) NOT NULL DEFAULT '',
  `author` VARCHAR(32) NOT NULL DEFAULT '',
  `created_time` BIGINT(12) UNSIGNED NOT NULL,
  `updated_time` BIGINT(12) UNSIGNED NOT NULL,
  KEY IDX_APP_ID_STATUS (`app_id`, `status`)
) ENGINE=InnoDB CHARACTER SET=utf8 COLLATE=utf8_general_ci;

DROP TABLE IF EXISTS `dandelion_app_instances`;
CREATE TABLE `dandelion_app_instances` (
  `id` BIGINT(12) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `app_id` VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'app id',
  `status` TINYINT(1) NOT NULL DEFAULT '0' COMMENT '0: diabled, 1: enabled',
  `host` VARCHAR(50) NOT NULL DEFAULT '',
  `instance_id` VARCHAR(50) NOT NULL DEFAULT '',
  `config_id` BIGINT(12) UNSIGNED NOT NULL DEFAULT '0',
  `commit_id` CHAR(40) NOT NULL DEFAULT '',
  `created_time` BIGINT(12) UNSIGNED NOT NULL,
  `updated_time` BIGINT(12) UNSIGNED NOT NULL,
  KEY IDX_APP_ID (`app_id`)
) ENGINE=InnoDB CHARACTER SET=utf8 COLLATE=utf8_general_ci;
