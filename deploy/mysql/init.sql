-- 预约系统数据库初始化脚本
-- 创建时间: 2026-04-05
-- 更新时间: 2026-04-24
-- 架构: 订单+时段双表设计（支持多时段批量预约）

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
-- 预约订单表：一次提交生成一个订单
-- 存放申请人信息和共享字段
-- =============================================
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
    `status` TINYINT NOT NULL DEFAULT 0 COMMENT '整体状态: 0-待审核, 1-通过, 2-拒绝, 3-完成, 4-取消',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_orders_order_no` (`order_no`),
    KEY `idx_orders_open_id` (`open_id`),
    KEY `idx_orders_status` (`status`),
    KEY `idx_orders_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='预约订单表';

-- =============================================
-- 预约时段明细表：每个时间段一行
-- 独立状态、独立门锁密码
-- =============================================
CREATE TABLE IF NOT EXISTS `reservation_slots` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `order_id` BIGINT UNSIGNED NOT NULL COMMENT '关联订单ID',
    `start_time` DATETIME NOT NULL COMMENT '开始时间',
    `end_time` DATETIME NOT NULL COMMENT '结束时间',
    `status` TINYINT NOT NULL DEFAULT 0 COMMENT '时段状态: 0-待审核, 1-通过, 2-拒绝, 3-完成, 4-取消',
    `password` VARCHAR(20) DEFAULT NULL COMMENT '门锁动态密码',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_slots_order_id` (`order_id`),
    KEY `idx_slots_time_range` (`start_time`, `end_time`),
    KEY `idx_slots_status` (`status`),
    CONSTRAINT `fk_slots_order_id` FOREIGN KEY (`order_id`) REFERENCES `reservation_orders`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='预约时段明细表';

-- 完成
