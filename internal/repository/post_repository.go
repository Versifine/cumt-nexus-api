package repository

import (
	"context"
	"cumt-nexus-api/internal/model"
	"time"
)

type PostRepository struct{}

func NewPostRepository() *PostRepository {
	return &PostRepository{}
}

// PostListQuery 是帖子列表查询条件，目前只包含分页参数。
type PostListQuery struct {
	Offset int
	Limit  int
}

// PostListRow 是仓储层联表查询后的结果结构。
type PostListRow struct {
	ID              uint64    `gorm:"column:id"`
	Title           string    `gorm:"column:title"`
	ViewCount       int64     `gorm:"column:view_count"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	AuthorID        uint64    `gorm:"column:author_id"`
	AuthorNickname  string    `gorm:"column:author_nickname"`
	AuthorAvatarURL string    `gorm:"column:author_avatar_url"`
}

// PostDetailRow 是帖子详情查询后的结果结构。
type PostDetailRow struct {
	ID              uint64    `gorm:"column:id"`
	Title           string    `gorm:"column:title"`
	Content         string    `gorm:"column:content"`
	ViewCount       int64     `gorm:"column:view_count"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	AuthorID        uint64    `gorm:"column:author_id"`
	AuthorNickname  string    `gorm:"column:author_nickname"`
	AuthorAvatarURL string    `gorm:"column:author_avatar_url"`
}

func (r *PostRepository) Create(ctx context.Context, post *model.Post) error {
	return DB.WithContext(ctx).Create(post).Error
}

// List 通过联表一次性查出帖子列表和作者展示信息，避免 N+1 查询。
func (r *PostRepository) List(ctx context.Context, query PostListQuery) ([]*PostListRow, int64, error) {
	var (
		rows  []*PostListRow
		total int64
	)

	if err := DB.WithContext(ctx).Model(&model.Post{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := DB.WithContext(ctx).
		Table("posts AS p").
		Select(`
			p.id,
			p.title,
			p.view_count,
			p.created_at,
			u.id AS author_id,
			u.nickname AS author_nickname,
			u.avatar_url AS author_avatar_url`).
		Joins("LEFT JOIN users u ON u.id = p.user_id").
		Order("p.created_at DESC").
		Offset(query.Offset).
		Limit(query.Limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

// FindDetailByID 查询单篇帖子详情，并联表补齐作者展示信息。
func (r *PostRepository) FindDetailByID(ctx context.Context, id uint64) (*PostDetailRow, error) {
	var row PostDetailRow

	err := DB.WithContext(ctx).
		Table("posts AS p").
		Select(`
			p.id,
			p.title,
			p.content,
			p.view_count,
			p.created_at,
			u.id AS author_id,
			u.nickname AS author_nickname,
			u.avatar_url AS author_avatar_url`).
		Joins("LEFT JOIN users u ON u.id = p.user_id").
		Where("p.id = ?", id).
		Take(&row).Error
	if err != nil {
		return nil, err
	}

	return &row, nil
}
