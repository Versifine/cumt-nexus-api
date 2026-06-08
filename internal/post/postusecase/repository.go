package postusecase

import (
	"context"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
)

type PostRepository interface {
	Create(ctx context.Context, post postdomain.Post) error
	FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error)
	UpdateContent(ctx context.Context, post postdomain.Post) error
	MarkDeleted(ctx context.Context, post postdomain.Post) error
	ListVisibleByCommunity(ctx context.Context, communityID communitydomain.CommunityID, sort PostListSort, limit int, offset int) ([]postdomain.Post, error)
	ListVisibleInPublicCommunities(ctx context.Context, sort PostListSort, limit int, offset int) ([]postdomain.Post, error)
	ListVisibleByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID, sort PostListSort, limit int, offset int) ([]postdomain.Post, error)
}

type PostSaveRepository interface {
	SavePost(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID, now time.Time) error
	DeletePostSave(ctx context.Context, postID postdomain.PostID, userID userdomain.UserID) error
	ListSavedVisiblePosts(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]postdomain.Post, error)
	FindSavedPostIDsByUser(ctx context.Context, postIDs []postdomain.PostID, userID userdomain.UserID) (map[postdomain.PostID]bool, error)
	SummarizeSavesByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID]int, error)
}

type PublicUserFinder interface {
	FindByUsername(ctx context.Context, username userdomain.Username) (*userdomain.User, error)
}

type PostMetadataRepository interface {
	LoadMetadataByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID]PostMetadata, error)
}

type VoteRepository interface {
	FindByPostIDsAndUser(ctx context.Context, postIDs []postdomain.PostID, userID userdomain.UserID) (map[postdomain.PostID]votedomain.VoteValue, error)
	SummarizeByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID]votedomain.PostVoteSummary, error)
}

type AttachmentRepository interface {
	BindReadyImagesToPost(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error)
	ReplaceReadyImagesForPost(ctx context.Context, postID postdomain.PostID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error)
	ListReadyImagesByPostIDs(ctx context.Context, postIDs []postdomain.PostID) (map[postdomain.PostID][]mediadomain.Attachment, error)
}
