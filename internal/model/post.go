package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	PostTypeText  int8 = 1
	PostTypeLink  int8 = 2
	PostTypeImage int8 = 3

	PostStatusActive  int8 = 1
	PostStatusLocked  int8 = 2
	PostStatusRemoved int8 = 3
)

// Post 是 subreddit 下的帖子实体，兼容 text/link/image 三种基础类型。
type Post struct {
	ID           uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SubredditID  uint64         `gorm:"column:subreddit_id;not null;default:0;index:idx_posts_subreddit_created,priority:1" json:"subreddit_id"`
	UserID       uint64         `gorm:"column:user_id;not null;index:idx_posts_user_id" json:"user_id"`
	PostType     int8           `gorm:"column:post_type;type:tinyint;not null;default:1" json:"post_type"`
	Title        string         `gorm:"column:title;type:varchar(300);not null" json:"title"`
	Slug         string         `gorm:"column:slug;type:varchar(320);not null;default:'';index:idx_posts_slug" json:"slug"`
	Content      string         `gorm:"column:content;type:longtext;not null" json:"content"`
	URL          string         `gorm:"column:url;type:varchar(1024);not null;default:''" json:"url"`
	ViewCount    int64          `gorm:"column:view_count;not null;default:0" json:"view_count"`
	Score        int64          `gorm:"column:score;not null;default:0;index:idx_posts_score" json:"score"`
	CommentCount int64          `gorm:"column:comment_count;not null;default:0" json:"comment_count"`
	Status       int8           `gorm:"column:status;type:tinyint;not null;default:1;index:idx_posts_status" json:"status"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime;index:idx_posts_subreddit_created,priority:2" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (Post) TableName() string { return "posts" }
