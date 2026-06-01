-- 预约系统测试数据初始化脚本
-- 用于测试阶段，插入不同状态的预约订单及对应时段
-- 状态码对应 pkg/reservationdb/model.go:
--   1=等待一级审核, 2=等待二级审核, 3=一级审核拒绝, 4=二级审核拒绝,
--   5=审核通过, 6=订单已经取消, 7=订单已经完成

SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;

USE home_xy;

-- =============================================
-- 测试用户
-- =============================================
INSERT INTO `users` (`openid`, `nickname`, `status`) VALUES
('test_openid_001', '张三', 1),
('test_openid_002', '李四', 1),
('test_openid_003', '王五', 1),
('test_openid_004', '赵六', 1),
('test_openid_005', '钱七', 0)
ON DUPLICATE KEY UPDATE `nickname`=VALUES(`nickname`);

USE home_res;

-- =============================================
-- 预约订单 — 覆盖所有状态
-- 状态码: 1=等待一级审核, 2=等待二级审核, 3=一级审核拒绝,
--         4=二级审核拒绝, 5=审核通过, 6=已取消, 7=已完成
-- =============================================

-- 1. 等待一级审核（单时段）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(1, 'ORD20260501001', 'test_openid_001', '张三', '计算机学院校友会', 2018, '计算机科学与技术', '校友返校座谈会', '13800000001', 1, 1, '2026-05-01 09:00:00');

-- 2. 等待一级审核（多时段）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(2, 'ORD20260501002', 'test_openid_002', '李四', '经济学院校友会', 2016, '金融学', '校友交流沙龙（上午+下午）', '13800000002', 2, 1, '2026-05-01 10:30:00');

-- 3. 审核通过（单时段，已通过两级审核）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(3, 'ORD20260430001', 'test_openid_001', '张三', '计算机学院校友会', 2018, '计算机科学与技术', '技术分享会', '13800000001', 1, 5, '2026-04-30 08:00:00');

-- 4. 审核通过（多时段，4个时段，已通过两级审核）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(4, 'ORD20260429001', 'test_openid_003', '王五', '法学院校友会', 2015, '法学', '校友年会（全天）', '13800000003', 4, 5, '2026-04-29 14:00:00');

-- 5. 一级审核拒绝
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(5, 'ORD20260428001', 'test_openid_002', '李四', '经济学院校友会', 2016, '金融学', '私人聚会', '13800000002', 1, 3, '2026-04-28 16:00:00');

-- 6. 订单已经完成（单时段）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(6, 'ORD20260420001', 'test_openid_001', '张三', '计算机学院校友会', 2018, '计算机科学与技术', '读书分享会', '13800000001', 1, 7, '2026-04-20 09:00:00');

-- 7. 订单已经完成（多时段）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(7, 'ORD20260418001', 'test_openid_003', '王五', '法学院校友会', 2015, '法学', '法律论坛', '13800000003', 2, 7, '2026-04-18 10:00:00');

-- 8. 订单已经取消（单时段）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(8, 'ORD20260425001', 'test_openid_004', '赵六', '文学院校友会', 2019, '汉语言文学', '诗词朗诵会（后因行程冲突取消）', '13800000004', 1, 6, '2026-04-25 11:00:00');

-- 9. 订单已经取消（多时段）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(9, 'ORD20260422001', 'test_openid_004', '赵六', '文学院校友会', 2019, '汉语言文学', '校友聚餐（后取消）', '13800000004', 2, 6, '2026-04-22 15:00:00');

-- 10. 等待一级审核（用户已取消关注）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(10, 'ORD20260501003', 'test_openid_005', '钱七', '外语学院校友会', 2020, '英语', '外籍校友交流活动', '13800000005', 1, 1, '2026-05-01 14:00:00');

-- 11. 等待二级审核（已通过一级审核）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(11, 'ORD20260502001', 'test_openid_001', '张三', '计算机学院校友会', 2018, '计算机科学与技术', '校友技术沙龙', '13800000001', 1, 2, '2026-05-02 10:00:00');

-- =============================================
-- 预约时段明细 — 状态与对应订单一致
-- =============================================

-- 订单1: 等待一级审核（单时段）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(1, '2026-05-05 09:00:00', '2026-05-05 12:00:00', 1, NULL);

-- 订单2: 等待一级审核（多时段 2个）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(2, '2026-05-06 09:00:00', '2026-05-06 12:00:00', 1, NULL),
(2, '2026-05-06 14:00:00', '2026-05-06 17:00:00', 1, NULL);

-- 订单3: 审核通过（单时段，有密码）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(3, '2026-05-03 09:00:00', '2026-05-03 12:00:00', 5, '883721');

-- 订单4: 审核通过（4时段，有密码）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(4, '2026-05-04 09:00:00', '2026-05-04 12:00:00', 5, '551023'),
(4, '2026-05-04 14:00:00', '2026-05-04 17:00:00', 5, '551024'),
(4, '2026-05-05 09:00:00', '2026-05-05 12:00:00', 5, '551025'),
(4, '2026-05-05 14:00:00', '2026-05-05 17:00:00', 5, '551026');

-- 订单5: 一级审核拒绝（单时段）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(5, '2026-05-02 09:00:00', '2026-05-02 12:00:00', 3, NULL);

-- 订单6: 已完成（单时段，有密码）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(6, '2026-04-22 09:00:00', '2026-04-22 12:00:00', 7, '221845');

-- 订单7: 已完成（多时段，有密码）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(7, '2026-04-21 09:00:00', '2026-04-21 12:00:00', 7, '211901'),
(7, '2026-04-21 14:00:00', '2026-04-21 17:00:00', 7, '211902');

-- 订单8: 已取消（单时段）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(8, '2026-04-28 14:00:00', '2026-04-28 17:00:00', 6, NULL);

-- 订单9: 已取消（多时段）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(9, '2026-04-25 09:00:00', '2026-04-25 12:00:00', 6, NULL),
(9, '2026-04-25 14:00:00', '2026-04-25 17:00:00', 6, NULL);

-- 订单10: 等待一级审核（用户已取消关注）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(10, '2026-05-07 14:00:00', '2026-05-07 17:00:00', 1, NULL);

-- 订单11: 等待二级审核（已通过一级审核）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(11, '2026-05-03 14:00:00', '2026-05-03 17:00:00', 2, NULL);

-- =============================================
-- 审核记录
-- =============================================

-- 订单3 (审核通过): 一级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(3, 1, 1, 1, '同意，请按时到场', '2026-04-30 10:00:00');

-- 订单4 (审核通过): 一级通过 + 二级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(4, 1, 1, 1, '同意', '2026-04-29 16:00:00'),
(4, 2, 2, 1, '审核通过', '2026-04-29 17:00:00');

-- 订单5 (一级审核拒绝): 一级拒绝
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(5, 1, 1, 2, '该用途不符合场地使用规定，不予批准', '2026-04-28 17:00:00');

-- 订单6 (已完成): 一级通过 + 二级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(6, 1, 1, 1, '同意', '2026-04-20 11:00:00'),
(6, 2, 2, 1, '同意，安排妥当', '2026-04-20 14:00:00');

-- 订单7 (已完成): 一级通过 + 二级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(7, 1, 1, 1, '同意', '2026-04-18 12:00:00'),
(7, 2, 2, 1, '审核通过', '2026-04-18 14:00:00');

-- 订单11 (等待二级审核): 一级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(11, 1, 1, 1, '申请合理，同意进入二级审核', '2026-05-02 14:00:00');

-- 完成
