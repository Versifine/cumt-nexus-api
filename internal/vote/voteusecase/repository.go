package voteusecase

import (
	"context"

	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
)

type PostVoteRepository interface {
	Upsert(ctx context.Context, vote votedomain.PostVote) error
	DeleteByPostAndUser(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID) error
	FindByPostAndUser(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID) (*votedomain.PostVote, error)
	FindByPostIDsAndUser(ctx context.Context, postIDs []postdomain.PostID, userID userdomain.UserID) (map[postdomain.PostID]votedomain.VoteValue, error)
	SummarizeByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID]PostVoteSummary, error)
}

type PostVoteSummary = votedomain.PostVoteSummary
