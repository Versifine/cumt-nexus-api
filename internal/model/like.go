package model

import "time"

type Like struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID     uint64    `gorm:"column:user_id;not null;uniqueIndex:uk_user_target,priority:1" json:"user_id"`
	TargetID   uint64    `gorm:"column:target_id;not null;uniqueIndex:uk_user_target,priority:2" json:"target_id"`
	TargetType int8      `gorm:"column:target_type;type:tinyint;not null;uniqueIndex:uk_user_target,priority:3" json:"target_type"` // 1帖子 2评论
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Like) TableName() string { return "likes" }
