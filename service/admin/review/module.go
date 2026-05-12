package review

import (
	reservationdb "reservation-sys/pkg/reservationdb"

	pb "reservation-sys/service/gateway/api/gen/notification"
)

var reviewService *ReviewService

// InitModule 初始化审核模块。
//
// 参数:
//   - notifyCli: Gateway 通知服务的 gRPC 客户端
//
// 注意: 调用前需确保 reservationdb.InitModule 已执行
func InitModule(notifyCli pb.NotificationServiceClient) {
	repo := reservationdb.GetRepository()
	reviewService = NewReviewService(repo)
}

// GetReviewService 获取审核服务实例。
// 未初始化时触发 panic，确保调用方在 InitModule 之后使用。
//
// 返回值:
//   - *ReviewService: 审核服务实例
func GetReviewService() *ReviewService {
	if reviewService == nil {
		panic("review module not initialized")
	}
	return reviewService
}
