package reservation

import (
	"encoding/json"
	"testing"
)

// FuzzSubmitReqValidation 测试 dto.go 中 SubmitReq 的 JSON 反序列化健壮性
//
// 函数功能：验证任意 JSON 输入解析 SubmitReq 不会 panic
//
// 测试场景：
// 1. 各种合法/非法 JSON 输入反序列化不应 panic
//  1. Fuzz 引擎自动变异种子数据（随机增删字节、翻转二进制位、插入特殊字符、调整数组长度、改变数字值）
//  2. JSON 解析错误是可接受的（返回 error），但不能 panic
func FuzzSubmitReqValidation(f *testing.F) {
	// 种子语料
	seeds := []string{
		`{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"测试","phone":"13800138000","slots":[{"start_time":"2026-03-25 08:00:00","end_time":"2026-03-25 10:00:00"}]}`,
		`{}`,
		`{"applicant_name":""}`,
		`{"slots":[]}`,
		`{"slots":null}`,
		`{"applicant_name":"测试","alumni_association":"校友会","year":2020,"major":"CS","reason":"测试","phone":"13800138000","slots":[{"start_time":"invalid","end_time":"2026-03-25 10:00:00"}]}`,
		`{"applicant_name":"A","alumni_association":"B","year":-1,"major":"C","reason":"D","phone":"12345678901","slots":[{"start_time":"2026-03-25 08:00:00","end_time":"2026-03-25 10:00:00"}]}`,
		`{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"测试","phone":"13800138000","slots":[{"start_time":"2026-03-25 08:00:00","end_time":"2026-03-25 10:00:00"},{"start_time":"2026-03-25 13:00:00","end_time":"2026-03-25 15:00:00"},{"start_time":"2026-03-26 08:00:00","end_time":"2026-03-26 10:00:00"},{"start_time":"2026-03-27 08:00:00","end_time":"2026-03-27 10:00:00"}]}`,
	}

	for _, s := range seeds {
		// f.Add 添加种子
		f.Add([]byte(s))
	}

	// Fuzz 测试的作用是持续地生成海量随机输入来反复调用
	// 会自动变异：随机增删字节、翻转二进制位、插入特殊字符、调整数组长度、改变数字值
	// 测试会一直持续运行
	f.Fuzz(func(t *testing.T, data []byte) {
		var req SubmitReq
		// JSON 解析不应 panic，无论输入是什么
		err := json.Unmarshal(data, &req)
		_ = err // 错误是可以接受的，但不能 panic
	})
}
