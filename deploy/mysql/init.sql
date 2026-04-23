-- 预约系统数据库初始化脚本
-- 创建时间: 2026-04-05

-- 设置字符集
SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;

-- 使用数据库
USE home_xy;

-- =============================================
-- 用户表
-- =============================================
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

-- =============================================
-- 预约表
-- =============================================
CREATE TABLE IF NOT EXISTS `reservations` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `order_no` VARCHAR(50) NOT NULL COMMENT '订单号',
    `open_id` VARCHAR(100) NOT NULL COMMENT '预约人标识',
    `application_name` VARCHAR(50) NOT NULL COMMENT '预约人名称',
    `reason` VARCHAR(500) NOT NULL COMMENT '预约理由',
    `phone` VARCHAR(20) NOT NULL COMMENT '电话号码',
    `num` INT NOT NULL COMMENT '预约人数',
    `start_time` DATETIME NOT NULL COMMENT '预约开始时间',
    `end_time` DATETIME NOT NULL COMMENT '预约结束时间',
    `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态: 0-待审核, 1-通过, 2-拒绝, 3-完成, 4-取消',
    `password` VARCHAR(20) DEFAULT NULL COMMENT '门锁动态密码',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_reservations_order_no` (`order_no`),
    KEY `idx_reservations_openid` (`openid`),
    KEY `idx_reservations_status` (`status`),
    KEY `idx_reservations_start_time` (`start_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='预约表';

-- =============================================
-- 初始数据（可选）
-- =============================================

-- 完成
