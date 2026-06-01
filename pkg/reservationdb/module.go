package reservationdb

import (
	"gorm.io/gorm"
)

var repo Repository

// InitModule 初始化预约数据库模块（创建仓库实例）。
//
// 数据库表结构由 deploy/mysql/init.sql 管理，不使用 GORM AutoMigrate。
//
// 参数:
//   - db: 已初始化的 GORM 数据库连接（指向 home_res 库）
//
// 注意: 必须在 InitModule 之前调用 platform.InitDB 完成数据库连接
func InitModule(db *gorm.DB) error {
	repo = NewRepository(db)
	return nil
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
