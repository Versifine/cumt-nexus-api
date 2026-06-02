package searchusecase

import (
	"context"
	"time"
)

type Repository interface {
	SearchCommunities(ctx context.Context, query string, limit int, offset int) ([]CommunityResult, error)
	SearchPosts(ctx context.Context, query string, limit int, offset int) ([]PostResult, error)
}

type CommunityResult struct {
	ID          string
	Slug        string
	Name        string
	Description string
	Kind        string
	Status      string
	Visibility  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PostResult struct {
	ID            string
	CommunityID   string
	CommunitySlug string
	AuthorID      string
	Title         string
	BodyExcerpt   string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
