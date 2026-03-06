package model

import "time"

type User struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username     string    `gorm:"column:username;type:varchar(64);not null;uniqueIndex:uk_users_username" json:"username"`
	PasswordHash string    `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	Nickname     string    `gorm:"column:nickname;type:varchar(64);not null;default:''" json:"nickname"`
	AvatarURL    string    `gorm:"column:avatar_url;type:varchar(255);not null;default:''" json:"avatar_url"`
	Role         int8      `gorm:"column:role;type:TINYINT;not null;default:0" json:"role"`
	Status       int8      `gorm:"column:status;type:TINYINT;not null;default:1" json:"status"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
