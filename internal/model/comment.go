package model

import (
	"time"

	"gorm.io/gorm"
)

// Comment 使用 parent_id + root_id + depth + path 表达评论树。
type Comment struct {
	ID           uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PostID       uint64         `gorm:"column:post_id;not null;index:idx_comments_post_created,priority:1" json:"post_id"`
	UserID       uint64         `gorm:"column:user_id;not null;index:idx_comments_user_id" json:"user_id"`
	ParentID     uint64         `gorm:"column:parent_id;not null;default:0;index:idx_comments_parent_id" json:"parent_id"`
	RootID       uint64         `gorm:"column:root_id;not null;default:0;index:idx_comments_root_created,priority:1" json:"root_id"`
	Depth        int16          `gorm:"column:depth;type:smallint;not null;default:0;index:idx_comments_post_depth,priority:2" json:"depth"`
	Path         string         `gorm:"column:path;type:varchar(255);not null;default:'';index:idx_comments_path" json:"path"`
	Content      string         `gorm:"column:content;type:longtext;not null" json:"content"`
	Score        int64          `gorm:"column:score;not null;default:0" json:"score"`
	RepliesCount int64          `gorm:"column:replies_count;not null;default:0" json:"replies_count"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime;index:idx_comments_post_created,priority:2" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (Comment) TableName() string { return "comments" }
