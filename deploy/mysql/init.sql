-- 预约系统数据库初始化脚本
-- 架构: 双数据库设计
--   home_xy: 账号数据库（users, admins）- Gateway 管理
--   home_res: 预约+审核数据库（reservation_orders, reservation_slots, review_records）- Reservation + Admin 共享

SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;

-- 创建用户角色并设置密码
CREATE USER IF NOT EXISTS 'res_user'@'%' IDENTIFIED BY 'xSIn34sU7qQl31kQ3TVfcQ==';

-- =============================================
-- 数据库1: 账号数据库（Gateway 管理）
-- =============================================

CREATE DATABASE IF NOT EXISTS `home_xy` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 授权: res_user 对 home_xy 数据库的全部权限
GRANT ALL PRIVILEGES ON `home_xy`.* TO 'res_user'@'%';
USE `home_xy`;

-- 用户表
CREATE TABLE IF NOT EXISTS `users` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `openid` VARCHAR(100) NOT NULL COMMENT '微信唯一标识',
    `nickname` VARCHAR(255) DEFAULT NULL COMMENT '昵称',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1-正常, 0-已取消关注',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `last_login` DATETIME DEFAULT NULL COMMENT '最后登录时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_users_openid` (`openid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 管理员表
CREATE TABLE IF NOT EXISTS `admins` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `username` VARCHAR(50) NOT NULL COMMENT '登录账号',
    `password` VARCHAR(100) NOT NULL COMMENT '密码（bcrypt哈希）',
    `real_name` VARCHAR(50) NOT NULL COMMENT '真实姓名',
    `role` TINYINT NOT NULL DEFAULT 1 COMMENT '角色: 1-一级管理员, 2-二级管理员',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1-正常, 0-禁用',
    `last_login_at` DATETIME DEFAULT NULL COMMENT '最后登录时间',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_admins_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='管理员表';

-- 插入默认管理员账号
INSERT INTO `admins` (`username`, `password`, `real_name`, `role`) VALUES
('admin1', '$2a$10$ophtXKaQ85PoqlWd84MF7eKR/kg4EZFH7xfDG2PBKjlKp6teh14Xi', '一级管理员', 1),
('admin2', '$2a$10$ophtXKaQ85PoqlWd84MF7eKR/kg4EZFH7xfDG2PBKjlKp6teh14Xi', '二级管理员', 2)
ON DUPLICATE KEY UPDATE `username`=VALUES(`username`);

-- =============================================
-- 数据库2: 预约+审核数据库（Reservation + Admin 共享）
-- =============================================

CREATE DATABASE IF NOT EXISTS `home_res` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `home_res`;

-- 预约订单表
CREATE TABLE IF NOT EXISTS `reservation_orders` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `order_no` VARCHAR(50) NOT NULL COMMENT '订单号',
    `open_id` VARCHAR(100) NOT NULL COMMENT '微信用户标识',
    `applicant_name` VARCHAR(50) NOT NULL COMMENT '申请人姓名',
    `alumni_association` VARCHAR(100) NOT NULL COMMENT '所属学院校友会',
    `year` INT NOT NULL COMMENT '入学年份',
    `major` VARCHAR(30) NOT NULL COMMENT '专业',
    `reason` VARCHAR(500) NOT NULL COMMENT '会议内容/预约理由',
    `phone` VARCHAR(20) NOT NULL COMMENT '联系电话',
    `total_slots` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '预约时段数量(1~4)',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1-等待一级审核, 2-等待二级审核, 3-一级审核拒绝, 4-二级审核拒绝, 5-审核通过, 6-已取消, 7-已完成',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_orders_order_no` (`order_no`),
    KEY `idx_orders_open_id` (`open_id`),
    KEY `idx_orders_status` (`status`),
    KEY `idx_orders_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='预约订单表';

-- 预约时段明细表
CREATE TABLE IF NOT EXISTS `reservation_slots` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `order_id` BIGINT UNSIGNED NOT NULL COMMENT '关联订单ID',
    `start_time` DATETIME NOT NULL COMMENT '开始时间',
    `end_time` DATETIME NOT NULL COMMENT '结束时间',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '时段状态: 与订单状态同步',
    `password` VARCHAR(20) DEFAULT NULL COMMENT '门锁动态密码',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_slots_order_id` (`order_id`),
    KEY `idx_slots_time_range` (`start_time`, `end_time`),
    KEY `idx_slots_status` (`status`),
    CONSTRAINT `fk_slots_order_id` FOREIGN KEY (`order_id`) REFERENCES `reservation_orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='预约时段明细表';

-- 审核记录表
CREATE TABLE IF NOT EXISTS `review_records` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `order_id` BIGINT UNSIGNED NOT NULL COMMENT '关联订单ID',
    `reviewer_id` BIGINT UNSIGNED NOT NULL COMMENT '审核人ID',
    `reviewer_role` TINYINT NOT NULL COMMENT '审核人角色: 1-一级, 2-二级',
    `action` TINYINT NOT NULL COMMENT '操作: 1-通过, 2-拒绝',
    `comment` VARCHAR(500) DEFAULT NULL COMMENT '审核意见',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    KEY `idx_review_records_order_id` (`order_id`),
    KEY `idx_review_records_reviewer_id` (`reviewer_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='审核记录表';

-- 授权: res_user 对 home_res 数据库的全部权限
GRANT ALL PRIVILEGES ON `home_res`.* TO 'res_user'@'%';
FLUSH PRIVILEGES;

-- 完成
