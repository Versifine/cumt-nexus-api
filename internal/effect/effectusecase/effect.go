package effectusecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

const InitialPointBalance = 100

var effectIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type UseCase struct {
	repository Repository
	comments   CommentReader
	now        func() time.Time
}

type Repository interface {
	ListActiveEffects(ctx context.Context) ([]Effect, error)
	FindActiveEffectByID(ctx context.Context, effectID string) (Effect, error)
	GetOrCreatePointAccount(ctx context.Context, userID userdomain.UserID, initialBalance int, now time.Time) (PointAccount, error)
	ApplyCommentEffect(ctx context.Context, input ApplyCommentEffectRecordInput) (ApplyCommentEffectRecordResult, error)
}

type CommentReader interface {
	FindVisibleByID(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error)
}

type Effect struct {
	ID           string
	Name         string
	Description  string
	CostPoints   int
	AssetURL     string
	AnimationKey string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PointAccount struct {
	UserID         string
	Balance        int
	LifetimeEarned int
	LifetimeSpent  int
	UpdatedAt      time.Time
}

type CommentEffect struct {
	ID          string
	CommentID   string
	EffectID    string
	UserID      string
	PointsSpent int
	CreatedAt   time.Time
}

type ListEffectsCatalogResult struct {
	Effects []Effect
}

type GetMyPointsInput struct {
	ActorID userdomain.UserID
}

type GetMyPointsResult struct {
	Points PointAccount
}

type ApplyCommentEffectInput struct {
	ActorID   userdomain.UserID
	CommentID string
	EffectID  string
}

type ApplyCommentEffectResult struct {
	CommentEffect CommentEffect
	Points        PointAccount
}

type ApplyCommentEffectRecordInput struct {
	ID           string
	CommentID    commentdomain.CommentID
	EffectID     string
	UserID       userdomain.UserID
	PointsSpent  int
	InitialGrant int
	Now          time.Time
}

type ApplyCommentEffectRecordResult struct {
	CommentEffect CommentEffect
	Points        PointAccount
}

func NewUseCase(repository Repository, comments CommentReader, now func() time.Time) *UseCase {
	if now == nil {
		now = time.Now
	}
	return &UseCase{
		repository: repository,
		comments:   comments,
		now:        now,
	}
}

func (uc *UseCase) ListEffectsCatalog(ctx context.Context) (ListEffectsCatalogResult, error) {
	if uc.repository == nil {
		return ListEffectsCatalogResult{}, fmt.Errorf("effect repository is not configured")
	}
	effects, err := uc.repository.ListActiveEffects(ctx)
	if err != nil {
		return ListEffectsCatalogResult{}, fmt.Errorf("list effects catalog: %w", err)
	}
	return ListEffectsCatalogResult{Effects: effects}, nil
}

func (uc *UseCase) GetMyPoints(ctx context.Context, input GetMyPointsInput) (GetMyPointsResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return GetMyPointsResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.repository == nil {
		return GetMyPointsResult{}, fmt.Errorf("effect repository is not configured")
	}
	points, err := uc.repository.GetOrCreatePointAccount(ctx, input.ActorID, InitialPointBalance, uc.now().UTC())
	if err != nil {
		return GetMyPointsResult{}, fmt.Errorf("get point account: %w", err)
	}
	return GetMyPointsResult{Points: points}, nil
}

func (uc *UseCase) ApplyCommentEffect(ctx context.Context, input ApplyCommentEffectInput) (ApplyCommentEffectResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return ApplyCommentEffectResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	commentID, err := commentdomain.NewCommentID(input.CommentID)
	if err != nil {
		return ApplyCommentEffectResult{}, err
	}
	effectID, err := normalizeEffectID(input.EffectID)
	if err != nil {
		return ApplyCommentEffectResult{}, err
	}
	if uc.comments == nil {
		return ApplyCommentEffectResult{}, fmt.Errorf("comment reader is not configured")
	}
	if uc.repository == nil {
		return ApplyCommentEffectResult{}, fmt.Errorf("effect repository is not configured")
	}
	if _, err := uc.comments.FindVisibleByID(ctx, commentID); err != nil {
		return ApplyCommentEffectResult{}, fmt.Errorf("find visible comment: %w", err)
	}
	effect, err := uc.repository.FindActiveEffectByID(ctx, effectID)
	if err != nil {
		return ApplyCommentEffectResult{}, fmt.Errorf("find active effect: %w", err)
	}
	result, err := uc.repository.ApplyCommentEffect(ctx, ApplyCommentEffectRecordInput{
		ID:           uuid.NewString(),
		CommentID:    commentID,
		EffectID:     effect.ID,
		UserID:       input.ActorID,
		PointsSpent:  effect.CostPoints,
		InitialGrant: InitialPointBalance,
		Now:          uc.now().UTC(),
	})
	if err != nil {
		return ApplyCommentEffectResult{}, fmt.Errorf("apply comment effect: %w", err)
	}
	return ApplyCommentEffectResult{
		CommentEffect: result.CommentEffect,
		Points:        result.Points,
	}, nil
}

func normalizeEffectID(raw string) (string, error) {
	effectID := strings.ToLower(strings.TrimSpace(raw))
	if effectID == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "effect id is required")
	}
	if !effectIDPattern.MatchString(effectID) {
		return "", apperr.New(apperr.CodeInvalidArgument, "effect id is invalid")
	}
	return effectID, nil
}
