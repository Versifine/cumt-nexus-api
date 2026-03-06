package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Conf 声明一个全局配置变量，方便其他包在初始化时随时读取配置
// 这比将其作为参数一层层往下传要方便得多，在 Gin 后端开发中非常常见
var Conf *Config

func InitConfig() error {
	// 1. 尝试加载 .env 文件，将变量注入到系统环境
	// 忽略错误，因为在容器化生产环境中，通常不需要 .env 文件，而是直接靠系统的真实环境变量
	_ = godotenv.Load()

	// 2. 告诉 Viper 要读取的本地静态配置文件名称和路径
	viper.SetConfigName("config") // 文件名是 config
	viper.SetConfigType("yaml")   // 文件后缀是 .yaml
	viper.AddConfigPath(".")      // 代表当前项目根目录

	// 3. 开启读取环境变量
	viper.AutomaticEnv()
	// 如果配置层级是 mysql.password，Viper 默认找 MYSQL_PASSWORD
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 4. 手动显式绑定：将 .env 中特殊的键名映射到我们的配置层级里
	viper.BindEnv("app.port", "APP_PORT")
	viper.BindEnv("app.mode", "APP_MODE")
	viper.BindEnv("mysql.host", "DB_HOST")
	viper.BindEnv("mysql.port", "DB_PORT")
	viper.BindEnv("mysql.user", "DB_USER")
	viper.BindEnv("mysql.password", "DB_PASSWORD") // 让 DB_PASSWORD 直接覆盖 mysql.password
	viper.BindEnv("mysql.dbname", "DB_NAME")
	viper.BindEnv("redis.host", "REDIS_HOST")
	viper.BindEnv("redis.port", "REDIS_PORT")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("jwt.secret", "JWT_SECRET")

	// 5. 执行读取本地文件 (config.yaml)
	if err := viper.ReadInConfig(); err != nil {
		// 如果只是找不到 yaml 文件（比如生产环境全靠纯环境变量），我们其实可以不抛出 panic
		// 这里严谨一点：如果不是“找不到文件”这种类型的错误，就报错返回
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("viper read config failed: %w", err)
		}
	}

	// 6. 将所有读取到的（YAML + Env）配置信息，反序列化填充到全局变量 Conf 指针中
	Conf = new(Config)
	if err := viper.Unmarshal(Conf); err != nil {
		return fmt.Errorf("viper unmarshal failed: %w", err)
	}

	return nil
}
