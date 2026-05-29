-- 滑动窗口限流 Lua 脚本
-- 在 Redis 服务端原子执行：清理过期记录、统计请求数、判断是否超限、写入记录、设置过期时间
--
-- KEYS[1] = 限流 Key
-- ARGV[1] = 窗口大小（秒）
-- ARGV[2] = 最大请求数
-- ARGV[3] = 当前时间戳（秒）
-- ARGV[4] = 唯一成员标识（时间戳:随机数）
--
-- 返回值: 1 = 允许, 0 = 拒绝

local key = KEYS[1]
local window = tonumber(ARGV[1])
local max_requests = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local member = ARGV[4]
local window_start = now - window

-- 1. 删除窗口外的过期记录
-- SQL: DELETE FROM zset WHERE score < window_start
redis.call('ZREMRANGEBYSCORE', key, 0, window_start)

-- 2. 统计当前窗口内的请求数量
-- SQL: SELECT COUNT(*) FROM zset WHERE score >= window_start
local count = redis.call('ZCARD', key)

-- 3. 判断是否超限，若超限则不写入直接拒绝
if count >= max_requests then
    return 0
end

-- 4. 写入本次请求记录
redis.call('ZADD', key, now, member)

-- 5. 设置 Key 过期时间
redis.call('EXPIRE', key, window)

return 1
