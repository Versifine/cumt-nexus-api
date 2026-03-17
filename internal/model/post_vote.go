package model

import "time"

const (
	VoteValueDown int8 = -1
	VoteValueUp   int8 = 1
)

type PostVote struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PostID    uint64    `gorm:"column:post_id;not null;uniqueIndex:uk_post_votes_post_user,priority:1" json:"post_id"`
	UserID    uint64    `gorm:"column:user_id;not null;uniqueIndex:uk_post_votes_post_user,priority:2" json:"user_id"`
	Value     int8      `gorm:"column:value;type:tinyint;not null" json:"value"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PostVote) TableName() string { return "post_votes" }
