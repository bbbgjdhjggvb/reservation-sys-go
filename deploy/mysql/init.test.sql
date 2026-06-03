-- 预约系统测试数据初始化脚本
-- 所有日期使用 CURDATE() + 偏移量动态计算，确保每次重建数据卷时日期始终有效。
--
-- 状态码对应 pkg/reservationdb/model.go:
--   1=等待一级审核, 2=等待二级审核, 3=一级审核拒绝, 4=二级审核拒绝,
--   5=审核通过, 6=订单已经取消, 7=订单已经完成

SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;

-- =============================================
-- 基准日期变量
-- d0=今天, d1=明天, d2=后天（日历 is_mine 测试用）
-- pN=N天前（管理后台审核测试用）
-- slot_time(t, h) = TIMESTAMP(日期变量, 'HH:MM:00')
-- =============================================
SET @d0  = CURDATE();
SET @d1  = DATE_ADD(CURDATE(), INTERVAL  1 DAY);
SET @d2  = DATE_ADD(CURDATE(), INTERVAL  2 DAY);

SET @p5  = DATE_SUB(CURDATE(), INTERVAL  5 DAY);
SET @p10 = DATE_SUB(CURDATE(), INTERVAL 10 DAY);
SET @p15 = DATE_SUB(CURDATE(), INTERVAL 15 DAY);
SET @p20 = DATE_SUB(CURDATE(), INTERVAL 20 DAY);
SET @p25 = DATE_SUB(CURDATE(), INTERVAL 25 DAY);
SET @p30 = DATE_SUB(CURDATE(), INTERVAL 30 DAY);
SET @p35 = DATE_SUB(CURDATE(), INTERVAL 35 DAY);
SET @p40 = DATE_SUB(CURDATE(), INTERVAL 40 DAY);
SET @p50 = DATE_SUB(CURDATE(), INTERVAL 50 DAY);
SET @p60 = DATE_SUB(CURDATE(), INTERVAL 60 DAY);

-- =============================================
-- 测试用户
-- =============================================
USE home_xy;

INSERT INTO `users` (`openid`, `nickname`, `status`) VALUES
('test_openid_001', '张三', 1),
('test_openid_002', '李四', 1),
('test_openid_003', '王五', 1),
('test_openid_004', '赵六', 1),
('test_openid_005', '钱七', 0)
ON DUPLICATE KEY UPDATE `nickname`=VALUES(`nickname`);

-- =============================================
-- 预约订单 — 覆盖所有状态（日期相对今天动态偏移）
-- =============================================
USE home_res;

-- 1. 等待一级审核（单时段）— p5 天前
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(1, 'ORD0000000001', 'test_openid_001', '张三', '计算机学院校友会', 2018, '计算机科学与技术', '校友返校座谈会', '13800000001', 1, 1, TIMESTAMP(@p5, '09:00:00'));

-- 2. 等待一级审核（多时段）— p10 天前
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(2, 'ORD0000000002', 'test_openid_002', '李四', '经济学院校友会', 2016, '金融学', '校友交流沙龙（上午+下午）', '13800000002', 2, 1, TIMESTAMP(@p10, '10:30:00'));

-- 3. 审核通过（单时段，已通过两级审核）— p5 天前
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(3, 'ORD0000000003', 'test_openid_001', '张三', '计算机学院校友会', 2018, '计算机科学与技术', '技术分享会', '13800000001', 1, 5, TIMESTAMP(@p5, '08:00:00'));

-- 4. 审核通过（4个时段，全天）— p5 天前
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(4, 'ORD0000000004', 'test_openid_003', '王五', '法学院校友会', 2015, '法学', '校友年会（全天）', '13800000003', 4, 5, TIMESTAMP(@p15, '14:00:00'));

-- 5. 一级审核拒绝 — p20 天前
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(5, 'ORD0000000005', 'test_openid_002', '李四', '经济学院校友会', 2016, '金融学', '私人聚会', '13800000002', 1, 3, TIMESTAMP(@p20, '16:00:00'));

-- 6. 订单已经完成（单时段）— p30 天前
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(6, 'ORD0000000006', 'test_openid_001', '张三', '计算机学院校友会', 2018, '计算机科学与技术', '读书分享会', '13800000001', 1, 7, TIMESTAMP(@p30, '09:00:00'));

-- 7. 订单已经完成（多时段）— p35 天前
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(7, 'ORD0000000007', 'test_openid_003', '王五', '法学院校友会', 2015, '法学', '法律论坛', '13800000003', 2, 7, TIMESTAMP(@p35, '10:00:00'));

-- 8. 订单已经取消（单时段）— p25 天前
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(8, 'ORD0000000008', 'test_openid_004', '赵六', '文学院校友会', 2019, '汉语言文学', '诗词朗诵会（后因行程冲突取消）', '13800000004', 1, 6, TIMESTAMP(@p25, '11:00:00'));

-- 9. 订单已经取消（多时段）— p30 天前
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(9, 'ORD0000000009', 'test_openid_004', '赵六', '文学院校友会', 2019, '汉语言文学', '校友聚餐（后取消）', '13800000004', 2, 6, TIMESTAMP(@p30, '15:00:00'));

-- 10. 等待一级审核（用户已取消关注）— p5 天前
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(10, 'ORD0000000010', 'test_openid_005', '钱七', '外语学院校友会', 2020, '英语', '外籍校友交流活动', '13800000005', 1, 1, TIMESTAMP(@p5, '14:00:00'));

-- 11. 等待二级审核（已通过一级审核）— p5 天前
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(11, 'ORD0000000011', 'test_openid_001', '张三', '计算机学院校友会', 2018, '计算机科学与技术', '校友技术沙龙', '13800000001', 1, 2, TIMESTAMP(@p5, '10:00:00'));

-- =============================================
-- 预约时段明细 — 历史订单（pN 天前）
-- =============================================

-- 订单1: 等待一级审核（单时段）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(1, TIMESTAMP(@p5, '09:00:00'), TIMESTAMP(@p5, '12:00:00'), 1, NULL);

-- 订单2: 等待一级审核（多时段 2个）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(2, TIMESTAMP(@p5, '09:00:00'), TIMESTAMP(@p5, '12:00:00'), 1, NULL),
(2, TIMESTAMP(@p5, '14:00:00'), TIMESTAMP(@p5, '17:00:00'), 1, NULL);

-- 订单3: 审核通过（单时段，有密码）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(3, TIMESTAMP(@p5, '09:00:00'), TIMESTAMP(@p5, '12:00:00'), 5, '883721');

-- 订单4: 审核通过（4时段，有密码）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(4, TIMESTAMP(@p10, '09:00:00'), TIMESTAMP(@p10, '12:00:00'), 5, '551023'),
(4, TIMESTAMP(@p10, '14:00:00'), TIMESTAMP(@p10, '17:00:00'), 5, '551024'),
(4, TIMESTAMP(@p5,  '09:00:00'), TIMESTAMP(@p5,  '12:00:00'), 5, '551025'),
(4, TIMESTAMP(@p5,  '14:00:00'), TIMESTAMP(@p5,  '17:00:00'), 5, '551026');

-- 订单5: 一级审核拒绝（单时段）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(5, TIMESTAMP(@p15, '09:00:00'), TIMESTAMP(@p15, '12:00:00'), 3, NULL);

-- 订单6: 已完成（单时段，有密码）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(6, TIMESTAMP(@p25, '09:00:00'), TIMESTAMP(@p25, '12:00:00'), 7, '221845');

-- 订单7: 已完成（多时段，有密码）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(7, TIMESTAMP(@p30, '09:00:00'), TIMESTAMP(@p30, '12:00:00'), 7, '211901'),
(7, TIMESTAMP(@p30, '14:00:00'), TIMESTAMP(@p30, '17:00:00'), 7, '211902');

-- 订单8: 已取消（单时段）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(8, TIMESTAMP(@p20, '14:00:00'), TIMESTAMP(@p20, '17:00:00'), 6, NULL);

-- 订单9: 已取消（多时段）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(9, TIMESTAMP(@p25, '09:00:00'), TIMESTAMP(@p25, '12:00:00'), 6, NULL),
(9, TIMESTAMP(@p25, '14:00:00'), TIMESTAMP(@p25, '17:00:00'), 6, NULL);

-- 订单10: 等待一级审核（用户已取消关注）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(10, TIMESTAMP(@p5, '14:00:00'), TIMESTAMP(@p5, '17:00:00'), 1, NULL);

-- 订单11: 等待二级审核（已通过一级审核）
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(11, TIMESTAMP(@p5, '14:00:00'), TIMESTAMP(@p5, '17:00:00'), 2, NULL);

-- =============================================
-- 历史审核记录
-- =============================================

-- 订单3 (审核通过): 一级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(3, 1, 1, 1, '同意，请按时到场', TIMESTAMP(@p5, '10:00:00'));

-- 订单4 (审核通过): 一级通过 + 二级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(4, 1, 1, 1, '同意',           TIMESTAMP(@p15, '16:00:00')),
(4, 2, 2, 1, '审核通过',       TIMESTAMP(@p15, '17:00:00'));

-- 订单5 (一级审核拒绝): 一级拒绝
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(5, 1, 1, 2, '该用途不符合场地使用规定，不予批准', TIMESTAMP(@p20, '17:00:00'));

-- 订单6 (已完成): 一级通过 + 二级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(6, 1, 1, 1, '同意',           TIMESTAMP(@p30, '11:00:00')),
(6, 2, 2, 1, '同意，安排妥当', TIMESTAMP(@p30, '14:00:00'));

-- 订单7 (已完成): 一级通过 + 二级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(7, 1, 1, 1, '同意',     TIMESTAMP(@p35, '12:00:00')),
(7, 2, 2, 1, '审核通过', TIMESTAMP(@p35, '14:00:00'));

-- 订单11 (等待二级审核): 一级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(11, 1, 1, 1, '申请合理，同意进入二级审核', TIMESTAMP(@p5, '14:00:00'));


-- =============================================
-- 日历 is_mine 测试数据 — 使用 d1=明天, d2=后天
-- 确保每次重建数据卷时，日期始终在日历可预约范围内（今天~今天+13天）
--
-- 场景说明：
--   test_openid_001 在明天有 2 个 pending + 1 个 approved
--   test_openid_002 在明天有 1 个 pending（与001同一天）
--   test_openid_001 在后天有 2 个 approved
--   test_openid_003 在明天有 1 个 pending-level2
-- =============================================

-- 20. test_openid_001 在明天上午的 pending 订单（2个时段）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(20, 'ORD0000000020', 'test_openid_001', '张三', '计算机学院校友会', 2018, '计算机科学与技术', '校友技术交流会', '13800000001', 2, 1, TIMESTAMP(@d0, '09:00:00'));

-- 21. test_openid_002 在明天下午的 pending 订单（与001同一天，验证他人 is_mine=false）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(21, 'ORD0000000021', 'test_openid_002', '李四', '经济学院校友会', 2016, '金融学', '校友联谊会', '13800000002', 1, 1, TIMESTAMP(@d0, '10:00:00'));

-- 22. test_openid_001 在明天下午的 approved 订单（同一用户同一天不同状态）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(22, 'ORD0000000022', 'test_openid_001', '张三', '计算机学院校友会', 2018, '计算机科学与技术', '项目评审会', '13800000001', 1, 5, TIMESTAMP(@p5, '08:00:00'));

-- 23. test_openid_001 在后天的 approved 订单（跨天验证 is_mine）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(23, 'ORD0000000023', 'test_openid_001', '张三', '计算机学院校友会', 2018, '计算机科学与技术', '年度总结会议', '13800000001', 2, 5, TIMESTAMP(@p5, '14:00:00'));

-- 24. test_openid_003 在明天晚上的 pending-level2 订单（第三个用户同一天）
INSERT INTO `reservation_orders` (`id`, `order_no`, `open_id`, `applicant_name`, `alumni_association`, `year`, `major`, `reason`, `phone`, `total_slots`, `status`, `created_at`) VALUES
(24, 'ORD0000000024', 'test_openid_003', '王五', '法学院校友会', 2015, '法学', '法律咨询日', '13800000003', 1, 2, TIMESTAMP(@d0, '15:00:00'));

-- =============================================
-- 日历测试数据对应的时段（使用 @d1=明天, @d2=后天）
-- =============================================

-- 订单20: test_openid_001, 明天, pending (2个时段：08-10, 10-12)
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(20, TIMESTAMP(@d1, '08:00:00'), TIMESTAMP(@d1, '10:00:00'), 1, NULL),
(20, TIMESTAMP(@d1, '10:00:00'), TIMESTAMP(@d1, '12:00:00'), 1, NULL);

-- 订单21: test_openid_002, 明天, pending (1个时段：13-15)
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(21, TIMESTAMP(@d1, '13:00:00'), TIMESTAMP(@d1, '15:00:00'), 1, NULL);

-- 订单22: test_openid_001, 明天, approved (1个时段：15-17，与自己的 pending 同一天)
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(22, TIMESTAMP(@d1, '15:00:00'), TIMESTAMP(@d1, '17:00:00'), 5, '990011');

-- 订单23: test_openid_001, 后天, approved (2个时段：08-10, 10-12)
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(23, TIMESTAMP(@d2, '08:00:00'), TIMESTAMP(@d2, '10:00:00'), 5, '990012'),
(23, TIMESTAMP(@d2, '10:00:00'), TIMESTAMP(@d2, '12:00:00'), 5, '990013');

-- 订单24: test_openid_003, 明天, pending-level2 (1个时段：18-20)
INSERT INTO `reservation_slots` (`order_id`, `start_time`, `end_time`, `status`, `password`) VALUES
(24, TIMESTAMP(@d1, '18:00:00'), TIMESTAMP(@d1, '20:00:00'), 2, NULL);

-- =============================================
-- 日历测试数据的审核记录
-- =============================================

-- 订单22 (test_openid_001, approved): 一级通过 + 二级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(22, 1, 1, 1, '同意',                TIMESTAMP(@p5, '10:00:00')),
(22, 2, 2, 1, '审核通过，按时使用',  TIMESTAMP(@p5, '09:00:00'));

-- 订单23 (test_openid_001, approved): 一级通过 + 二级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(23, 1, 1, 1, '同意',     TIMESTAMP(@p5, '16:00:00')),
(23, 2, 2, 1, '审核通过', TIMESTAMP(@p5, '09:00:00'));

-- 订单24 (test_openid_003, 等待二级审核): 一级通过
INSERT INTO `review_records` (`order_id`, `reviewer_id`, `reviewer_role`, `action`, `comment`, `created_at`) VALUES
(24, 1, 1, 1, '同意进入二级审核', TIMESTAMP(@d0, '17:00:00'));


-- =============================================
-- 日历 is_mine 场景验证（每次重建数据卷后自动适配当前日期）
-- =============================================
-- 以 test_openid_001 身份查看明天日历时：
--   08:00-10:00 → 待审核   (订单20, 自己的 pending)
--   10:00-12:00 → 待审核   (订单20, 自己的 pending)
--   13:00-15:00 → 已占用   (订单21, 李四的 pending → 合并到已占用)
--   15:00-17:00 → 我的预约 (订单22, 自己的 approved)
--   18:00-20:00 → 已占用   (订单24, 王五的 pending → 合并到已占用)
--
-- 以 test_openid_002 身份查看明天日历时：
--   08:00-10:00 → 已占用   (订单20, 张三的)
--   10:00-12:00 → 已占用   (订单20, 张三的)
--   13:00-15:00 → 待审核   (订单21, 自己的 pending)
--   15:00-17:00 → 已占用   (订单22, 张三的)
--   18:00-20:00 → 已占用   (订单24, 王五的)
-- =============================================

-- 完成
