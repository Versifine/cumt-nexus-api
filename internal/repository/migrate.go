package repository

import "cumt-nexus-api/internal/model"

func AutoMigrate() error {
	return DB.AutoMigrate(
		&model.User{},
		&model.Subreddit{},
		&model.Subscription{},
		&model.Post{},
		&model.Comment{},
		&model.Vote{},
	)
}
