package reservationdb

import (
	"reservation-sys/pkg/platform"

	"gorm.io/gorm"
)

var repo Repository

// InitModule 初始化预约数据库模块（自动迁移表结构 + 创建仓库实例）。
//
// 参数:
//   - db: 已初始化的 GORM 数据库连接（指向 home_res 库）
//
// 注意: 必须在 InitModule 之前调用 platform.InitDB 完成数据库连接
func InitModule(db *gorm.DB) {
	platform.AutoMigrate(db, &ReservationOrder{})
	platform.AutoMigrate(db, &ReservationSlot{})
	platform.AutoMigrate(db, &ReviewRecord{})

	repo = NewRepository(db)
}

// GetRepository 获取仓库实例。
//
// 返回值:
//   - Repository: 已初始化的数据仓库接口
//
// 注意: 未初始化时触发 panic，确保调用方在 InitModule 之后使用
func GetRepository() Repository {
	if repo == nil {
		panic("reservationdb module not initialized")
	}
	return repo
}
