package service

type CommentRepo interface {
}

type CommentService struct {
	commentRepo CommentRepo
}

func NewCommentService(commentRepo CommentRepo) *CommentService {
	return &CommentService{commentRepo: commentRepo}
}

type CreateCommentInput struct {
	PostID   uint64  `json:"post_id"`
	UserID   uint64  `json:"user_id"`
	ParentID *uint64 `json:"parent_id,omitempty"`
	Content  string  `json:"content"`
}

func (cs *CommentService) CreateComment(input CreateCommentInput) error {
	// Implement the logic to create a comment
	



	return nil
}
