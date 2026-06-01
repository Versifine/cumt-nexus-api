package postusecase

import (
	"context"

	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
)

type PostRepository interface {
	Create(ctx context.Context, post postdomain.Post) error
	FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error)
	ListVisibleByCommunity(ctx context.Context, communityID communitydomain.CommunityID, limit int, offset int) ([]postdomain.Post, error)
}
