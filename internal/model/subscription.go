package model

import "time"

// Subscription 表示用户对 subreddit 的订阅关系。
type Subscription struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID      uint64    `gorm:"column:user_id;not null;uniqueIndex:uk_subscriptions_user_subreddit,priority:1" json:"user_id"`
	SubredditID uint64    `gorm:"column:subreddit_id;not null;uniqueIndex:uk_subscriptions_user_subreddit,priority:2" json:"subreddit_id"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Subscription) TableName() string { return "subscriptions" }
