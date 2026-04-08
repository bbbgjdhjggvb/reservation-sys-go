package platform

import (
	"fmt"
	"log"
	"reservation-sys/internal/auth"
	"reservation-sys/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB(cfg *config.MySQLConfig) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("[error]: MySQL 连接失败 (请检查密码或数据库是否存在): %v", err)
	}

	// 获取底层的 *sql.DB 对象
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("[error]: 获取底层 SQL 对象失败: %v", err)
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)

	// 检测连通性
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("[error]: MySQL Ping 失败 (网络不通或被防火墙拦截): %v", err)
	}

	log.Println("[info]: MySQL 连接成功！")

	/* 数据库中表格迁移
	 * 1. 在初始开发阶段使用gorm的表格自动迁移
	 * 2. 在上线的时候使用sql创建表格
	 */
	autoMigrate(db)
	return db
}

func autoMigrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&auth.User{})
	if err != nil {
		log.Fatalf("[error]: 自动迁移表格失败: %v", err)
	} else {
		log.Println("[info]: 自动迁移表格成功")
	}
}
