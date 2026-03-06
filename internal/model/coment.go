package model

import (
	"time"

	"gorm.io/gorm"
)

type Comment struct {
	ID        uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PostID    uint64         `gorm:"column:post_id;not null;index:idx_comments_post_id" json:"post_id"`
	UserID    uint64         `gorm:"column:user_id;not null;index:idx_comments_user_id" json:"user_id"`
	ParentID  uint64         `gorm:"column:parent_id;not null;default:0;index:idx_comments_parent_id" json:"parent_id"` // 顶级评论=0
	Content   string         `gorm:"column:content;type:text;not null" json:"content"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (Comment) TableName() string { return "comments" }
