package commentusecase

import (
	"context"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/vote/votedomain"
)

type CommentRepository interface {
	Create(ctx context.Context, comment commentdomain.Comment) error
	FindVisibleByID(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error)
	UpdateContent(ctx context.Context, comment commentdomain.Comment) error
	MarkDeleted(ctx context.Context, comment commentdomain.Comment) error
	ListVisibleByPost(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error)
	ListVisibleTreeByPost(ctx context.Context, postID postdomain.PostID) ([]commentdomain.Comment, error)
	ListVisibleByAuthorInPublicCommunities(ctx context.Context, authorID userdomain.UserID, limit int, offset int) ([]commentdomain.Comment, error)
}

type PostReader interface {
	FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error)
}

type PublicUserFinder interface {
	FindByUsername(ctx context.Context, username userdomain.Username) (*userdomain.User, error)
}

type CommentMetadataRepository interface {
	LoadMetadataByCommentIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID]CommentMetadata, error)
}

type CommentVoteRepository interface {
	UpsertCommentVote(ctx context.Context, vote votedomain.CommentVote) error
	DeleteCommentVote(ctx context.Context, commentID commentdomain.CommentID, userID userdomain.UserID) error
	FindCommentVotesByIDsAndUser(ctx context.Context, commentIDs []commentdomain.CommentID, userID userdomain.UserID) (map[commentdomain.CommentID]votedomain.VoteValue, error)
	SummarizeCommentVotesByIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID]votedomain.CommentVoteSummary, error)
}

type CommentMetadata struct {
	Author postusecase.UserSummary
}

type AttachmentRepository interface {
	BindReadyImagesToComment(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error)
	ReplaceReadyImagesForComment(ctx context.Context, commentID commentdomain.CommentID, uploaderID userdomain.UserID, attachmentIDs []mediadomain.AttachmentID, maxCount int, now time.Time) ([]mediadomain.Attachment, error)
	ListReadyImagesByCommentIDs(ctx context.Context, commentIDs []commentdomain.CommentID) (map[commentdomain.CommentID][]mediadomain.Attachment, error)
}
