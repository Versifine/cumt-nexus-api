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
		return fmt.Errorf("连接 MySQL 失败: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取原生 MySQL 连接池失败: %w", err)
	}

	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)

	logger.Log.Infow("MySQL 连接初始化成功",
		"component", "mysql",
		"host", c.Host,
		"port", c.Port,
		"dbname", c.DBName,
		"max_open_conns", 100,
		"max_idle_conns", 10,
	)

	return nil
}
