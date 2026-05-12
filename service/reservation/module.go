package reservation

import (
	reservationdb "reservation-sys/pkg/reservationdb"
)

// 模块级别的服务实例
var reservationService *ReservationService

// InitModule 初始化预约模块。
//
// 注意: 调用前需确保 reservationdb.InitModule 已执行
func InitModule() {
	repo := reservationdb.GetRepository()
	reservationService = NewReservationService(repo)
}

// GetReservationService 获取预约服务实例。
// 未初始化时触发 panic，确保调用方在 InitModule 之后使用。
//
// 返回值:
//   - *ReservationService: 预约服务实例
func GetReservationService() *ReservationService {
	if reservationService == nil {
		panic("reservation module not initialized")
	}
	return reservationService
}
