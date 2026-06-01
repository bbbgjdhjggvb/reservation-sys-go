package platform

import (
	"fmt"
	"log"
	"reservation-sys/pkg/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// InitDB 初始化数据库连接
func InitDB(cfg *config.MySQLConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("[error][pkg/platform/db]: MySQL 连接失败 (请检查密码或数据库是否存在): %w", err)
	}

	// 获取底层的 *sql.DB 对象
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("[error][pkg/platform/db]: 获取底层 SQL 对象失败: %w", err)
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)

	// 检测连通性
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("[error][pkg/platform/db]: MySQL Ping 失败 (网络不通或被防火墙拦截): %w", err)
	}

	log.Println("[info][pkg/platform/db]: MySQL 连接成功！")
	return db, nil
}
