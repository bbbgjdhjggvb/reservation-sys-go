package platform

import (
	"fmt"
	"log"
	"reservation-sys/pkg/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// InitDB 初始化数据库连接
func InitDB(cfg *config.MySQLConfig) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("[error][pkg/platform/db]: MySQL 连接失败 (请检查密码或数据库是否存在): %v\n password: %s", err, cfg.Password)
	}

	// 获取底层的 *sql.DB 对象
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("[error][pkg/platform/db]: 获取底层 SQL 对象失败: %v", err)
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)

	// 检测连通性
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("[error][pkg/platform/db]: MySQL Ping 失败 (网络不通或被防火墙拦截): %v", err)
	}

	log.Println("[info][pkg/platform/db]: MySQL 连接成功！")
	return db
}

// AutoMigrate 自动迁移表结构
// 由各模块在 InitModule 时调用，保持 platform 层不依赖业务模型
func AutoMigrate(db *gorm.DB, models ...any) {
	if err := db.AutoMigrate(models...); err != nil {
		log.Fatalf("[error][pkg/platform/db]: 自动迁移表格失败: %v", err)
	}
	log.Println("[info][pkg/platform/db]: 自动迁移表格成功")
}
