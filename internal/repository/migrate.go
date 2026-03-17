package repository

import "cumt-nexus-api/internal/model"

func AutoMigrate() error {
	return DB.AutoMigrate(
		&model.User{},
		&model.Community{},
		&model.Post{},
		&model.Comment{},
		&model.PostVote{},
		&model.CommentVote{},
	)
}
