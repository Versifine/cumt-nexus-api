package model

import "time"

type Post struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"column:user_id;not null;index:idx_posts_user_id" json:"user_id"`
	Title     string    `gorm:"column:title;type:varchar(128);not null" json:"title"`
	Content   string    `gorm:"column:content;type:text;not null" json:"content"`
	ViewCount int64     `gorm:"column:view_count;not null;default:0" json:"view_count"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;index:idx_posts_created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Post) TableName() string { return "posts" }
