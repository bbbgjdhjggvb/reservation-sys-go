// internal/reservation/service_test.go
package reservation

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestReservationService_Submit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	// 测试用例
	tests := []struct {
		name      string
		openid    string
		req       *SubmitReq
		mockSetup func()
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "正常提交预约",
			openid: "test_openid_001",
			req: &SubmitReq{
				ApplicantName: "张三",
				Reason:        "举办活动",
				Phone:         "13800138000",
				StartTime:     "2026-03-25 14:00:00",
				EndTime:       "2026-03-25 16:00:00",
			},
			mockSetup: func() {
				// Mock: 查询时间段，返回空（无冲突）
				mockRepo.EXPECT().
					FindByTimeRange(gomock.Any(), gomock.Any()).
					Return([]*Reservation{}, nil)
				// Mock: 创建成功
				mockRepo.EXPECT().
					Create(gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "时间段已被占用",
			openid: "test_openid_001",
			req: &SubmitReq{
				ApplicantName: "张三",
				Reason:        "举办活动",
				Phone:         "13800138000",
				StartTime:     "2026-03-25 14:00:00",
				EndTime:       "2026-03-25 16:00:00",
			},
			mockSetup: func() {
				// Mock: 查询时间段，返回已有预约
				mockRepo.EXPECT().
					FindByTimeRange(gomock.Any(), gomock.Any()).
					Return([]*Reservation{{ID: 1}}, nil)
			},
			wantErr: true,
			errMsg:  "该时间段已被预约",
		},
		{
			name:   "时间格式错误",
			openid: "test_openid_001",
			req: &SubmitReq{
				ApplicantName: "张三",
				Reason:        "举办活动",
				Phone:         "13800138000",
				StartTime:     "2026-03-25", // 格式错误
				EndTime:       "2026-03-25 16:00:00",
			},
			mockSetup: func() {},
			wantErr:   true,
			errMsg:    "时间格式错误",
		},
		{
			name:   "结束时间早于开始时间",
			openid: "test_openid_001",
			req: &SubmitReq{
				ApplicantName: "张三",
				Reason:        "举办活动",
				Phone:         "13800138000",
				StartTime:     "2026-03-25 18:00:00",
				EndTime:       "2026-03-25 16:00:00",
			},
			mockSetup: func() {},
			wantErr:   true,
			errMsg:    "结束时间必须晚于开始时间",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			res, err := svc.Submit(tt.openid, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, tt.openid, res.OpenID)
			}
		})
	}
}

func TestReservationService_GetMyReservations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	t.Run("获取用户预约列表成功", func(t *testing.T) {
		expected := []*Reservation{
			{ID: 1, OpenID: "test_openid_001", Status: StatusPending},
			{ID: 2, OpenID: "test_openid_001", Status: StatusApproved},
		}

		mockRepo.EXPECT().
			FindByOpenID("test_openid_001").
			Return(expected, nil)

		result, err := svc.GetMyReservations("test_openid_001")

		assert.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("数据库错误", func(t *testing.T) {
		mockRepo.EXPECT().
			FindByOpenID("test_openid_001").
			Return(nil, errors.New("db error"))

		result, err := svc.GetMyReservations("test_openid_001")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestReservationService_Cancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	t.Run("取消成功", func(t *testing.T) {
		mockRepo.EXPECT().
			FindByID(uint(1)).
			Return(&Reservation{ID: 1, OpenID: "test_001", Status: StatusPending}, nil)
		mockRepo.EXPECT().
			Cancel(uint(1), "test_001").
			Return(nil)

		err := svc.Cancel(1, "test_001")
		assert.NoError(t, err)
	})

	t.Run("预约不存在", func(t *testing.T) {
		mockRepo.EXPECT().
			FindByID(uint(999)).
			Return(nil, errors.New("预约不存在"))

		err := svc.Cancel(999, "test_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "预约不存在")
	})

	t.Run("无权操作他人预约", func(t *testing.T) {
		mockRepo.EXPECT().
			FindByID(uint(1)).
			Return(&Reservation{ID: 1, OpenID: "other_user", Status: StatusPending}, nil)

		err := svc.Cancel(1, "test_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无权操作")
	})

	t.Run("已完成的预约无法取消", func(t *testing.T) {
		mockRepo.EXPECT().
			FindByID(uint(1)).
			Return(&Reservation{ID: 1, OpenID: "test_001", Status: StatusCompleted}, nil)

		err := svc.Cancel(1, "test_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "当前状态无法取消")
	})
}
