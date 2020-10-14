ALTER TABLE `goploy`.`project`
ADD COLUMN `before_deploy_script_mode` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '脚本类型(默认bash)' AFTER `review_url`,
ADD COLUMN `before_deploy_script` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT '获取代码后脚本路径' AFTER `before_deploy_script_mode`;