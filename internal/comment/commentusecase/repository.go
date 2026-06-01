package commentusecase

import (
	"context"

	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
)

type CommentRepository interface {
	Create(ctx context.Context, comment commentdomain.Comment) error
	FindVisibleByID(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error)
	ListVisibleByPost(ctx context.Context, postID postdomain.PostID, limit int, offset int) ([]commentdomain.Comment, error)
}
