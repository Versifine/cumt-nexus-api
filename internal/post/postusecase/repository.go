package postusecase

import (
	"context"

	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
)

type PostRepository interface {
	Create(ctx context.Context, post postdomain.Post) error
	FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error)
	ListVisibleByCommunity(ctx context.Context, communityID communitydomain.CommunityID, sort PostListSort, limit int, offset int) ([]postdomain.Post, error)
	ListVisibleInPublicCommunities(ctx context.Context, sort PostListSort, limit int, offset int) ([]postdomain.Post, error)
}

type VoteRepository interface {
	FindByPostIDsAndUser(ctx context.Context, postIDs []postdomain.PostID, userID userdomain.UserID) (map[postdomain.PostID]votedomain.VoteValue, error)
	SummarizeByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID]votedomain.PostVoteSummary, error)
}
