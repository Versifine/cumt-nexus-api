package model

import "time"

const (
	SubredditVisibilityPublic     int8 = 1
	SubredditVisibilityRestricted int8 = 2
	SubredditVisibilityPrivate    int8 = 3
)

// Subreddit 是 Reddit clone 中的社区实体。
type Subreddit struct {
	ID               uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name             string    `gorm:"column:name;type:varchar(64);not null;uniqueIndex:uk_subreddits_name" json:"name"`
	DisplayName      string    `gorm:"column:display_name;type:varchar(128);not null" json:"display_name"`
	Description      string    `gorm:"column:description;type:text;not null" json:"description"`
	Sidebar          string    `gorm:"column:sidebar;type:text;not null" json:"sidebar"`
	IconURL          string    `gorm:"column:icon_url;type:varchar(255);not null;default:''" json:"icon_url"`
	BannerURL        string    `gorm:"column:banner_url;type:varchar(255);not null;default:''" json:"banner_url"`
	Visibility       int8      `gorm:"column:visibility;type:tinyint;not null;default:1" json:"visibility"`
	CreatedBy        uint64    `gorm:"column:created_by;not null;index:idx_subreddits_created_by" json:"created_by"`
	SubscribersCount int64     `gorm:"column:subscribers_count;not null;default:0" json:"subscribers_count"`
	PostCount        int64     `gorm:"column:post_count;not null;default:0" json:"post_count"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Subreddit) TableName() string { return "subreddits" }
