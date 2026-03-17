package model

import "time"

const (
	VoteTargetPost    int8 = 1
	VoteTargetComment int8 = 2

	VoteValueDown int8 = -1
	VoteValueUp   int8 = 1
)

// Vote 统一表示对帖子或评论的投票，支持赞成、反对和取消。
type Vote struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID     uint64    `gorm:"column:user_id;not null;uniqueIndex:uk_votes_user_target,priority:1" json:"user_id"`
	TargetID   uint64    `gorm:"column:target_id;not null;uniqueIndex:uk_votes_user_target,priority:2" json:"target_id"`
	TargetType int8      `gorm:"column:target_type;type:tinyint;not null;uniqueIndex:uk_votes_user_target,priority:3" json:"target_type"`
	Value      int8      `gorm:"column:value;type:tinyint;not null;default:1" json:"value"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Vote) TableName() string { return "votes" }
