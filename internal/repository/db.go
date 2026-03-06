package repository

import (
	"cumt-nexus-api/internal/config"
	"cumt-nexus-api/internal/logger"

	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	c := config.Conf.MySQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", c.User, c.Password, c.Host, c.Port, c.DBName)
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	sqlDB, err := DB.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetMaxIdleConns(10)
	}
	logger.Log.Info("数据库连接成功")
	return nil
}
