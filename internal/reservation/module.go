// internal/reservation/module.go
package reservation

import (
	"reservation-sys/internal/platform"

	"gorm.io/gorm"
)

// 模块级别的服务实例，方便其他模块依赖
var reservationService *ReservationService

// InitModule 初始化预约模块
func InitModule(db *gorm.DB) {
	// 自动迁移表结构（订单表 + 时段明细表）
	platform.AutoMigrate(db, &ReservationOrder{})
	platform.AutoMigrate(db, &ReservationSlot{})

	// 初始化 Repository
	repo := NewReservationRepository(db)

	// 初始化 Service
	reservationService = NewReservationService(repo)
}

// GetReservationService 获取预约服务实例
func GetReservationService() *ReservationService {
	if reservationService == nil {
		panic("reservation module not initialized")
	}
	return reservationService
}
