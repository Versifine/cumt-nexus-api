package model

import "time"

type CommentVote struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CommentID uint64    `gorm:"column:comment_id;not null;uniqueIndex:uk_comment_votes_comment_user,priority:1" json:"comment_id"`
	UserID    uint64    `gorm:"column:user_id;not null;uniqueIndex:uk_comment_votes_comment_user,priority:2" json:"user_id"`
	Value     int8      `gorm:"column:value;type:tinyint;not null" json:"value"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (CommentVote) TableName() string { return "comment_votes" }
