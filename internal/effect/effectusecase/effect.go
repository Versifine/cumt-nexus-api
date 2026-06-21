package effectusecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
)

const InitialPointBalance = 100
const DefaultPointTransactionListLimit = 20
const MaxPointTransactionListLimit = 50

const (
	PointSourceDailyActivity  = "daily_activity"
	PointSourcePostPublish    = "post_publish"
	PointSourceCommentPublish = "comment_publish"
	PointSourcePostUpvote     = "post_upvote_received"
	PointSourceCommentUpvote  = "comment_upvote_received"
	PointSourcePostSave       = "post_saved_received"
	PointSourceReportAccepted = "report_accepted"
	PointSourceQualityContent = "quality_content"
	PointReasonDailyLogin     = "daily_login"
	PointReasonPostPublish    = "post_publish"
	PointReasonCommentPublish = "comment_publish"
	PointReasonPostUpvote     = "post_upvote_received"
	PointReasonCommentUpvote  = "comment_upvote_received"
	PointReasonPostSave       = "post_saved_received"
	PointReasonReportAccepted = "report_accepted"
	PointReasonQualityContent = "quality_content"
)

var pointPolicies = map[string]PointPolicy{
	PointSourceDailyActivity:  {Delta: 5, DailyCap: 20, Reason: PointReasonDailyLogin},
	PointSourcePostPublish:    {Delta: 5, DailyCap: 25, Reason: PointReasonPostPublish},
	PointSourceCommentPublish: {Delta: 1, DailyCap: 15, Reason: PointReasonCommentPublish},
	PointSourcePostUpvote:     {Delta: 1, DailyCap: 50, Reason: PointReasonPostUpvote},
	PointSourceCommentUpvote:  {Delta: 1, DailyCap: 50, Reason: PointReasonCommentUpvote},
	PointSourcePostSave:       {Delta: 3, DailyCap: 45, Reason: PointReasonPostSave},
	PointSourceReportAccepted: {Delta: 5, DailyCap: 20, Reason: PointReasonReportAccepted},
	PointSourceQualityContent: {Delta: 20, DailyCap: 80, Reason: PointReasonQualityContent},
}

var effectIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type UseCase struct {
	repository Repository
	comments   CommentReader
	posts      PostReader
	now        func() time.Time
}

type Repository interface {
	ListActiveEffects(ctx context.Context) ([]Effect, error)
	FindActiveEffectByID(ctx context.Context, effectID string) (Effect, error)
	GetOrCreatePointAccount(ctx context.Context, userID userdomain.UserID, initialBalance int, now time.Time) (PointAccount, error)
	ListPointTransactions(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]PointTransaction, error)
	ApplyCommentEffect(ctx context.Context, input ApplyCommentEffectRecordInput) (ApplyCommentEffectRecordResult, error)
	ApplyPostEffect(ctx context.Context, input ApplyPostEffectRecordInput) (ApplyPostEffectRecordResult, error)
	GrantPoints(ctx context.Context, input GrantPointsRecordInput) (GrantPointsRecordResult, error)
}

type CommentReader interface {
	FindVisibleByID(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error)
}

type PostReader interface {
	FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error)
}

type Effect struct {
	ID           string
	Name         string
	Description  string
	CostPoints   int
	AssetURL     string
	AnimationKey string
	Emoji        string
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

type PointPolicy struct {
	Delta    int
	DailyCap int
	Reason   string
}

type CommentEffect struct {
	ID          string
	CommentID   string
	EffectID    string
	UserID      string
	PointsSpent int
	CreatedAt   time.Time
}

type PostEffect struct {
	ID          string
	PostID      string
	EffectID    string
	UserID      string
	PointsSpent int
	CreatedAt   time.Time
}

type PointTransaction struct {
	ID           string
	UserID       string
	Delta        int
	BalanceAfter int
	Reason       string
	SourceType   string
	SourceID     string
	CreatedAt    time.Time
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

type ListMyPointTransactionsInput struct {
	ActorID userdomain.UserID
	Limit   int
	Offset  int
}

type ListMyPointTransactionsResult struct {
	Transactions []PointTransaction
	Limit        int
	Offset       int
	NextOffset   int
	HasMore      bool
}

type ApplyCommentEffectInput struct {
	ActorID   userdomain.UserID
	CommentID string
	EffectID  string
}

type ApplyPostEffectInput struct {
	ActorID  userdomain.UserID
	PostID   string
	EffectID string
}

type GrantPointsInput struct {
	UserID     userdomain.UserID
	ActorID    userdomain.UserID
	SourceType string
	SourceID   string
}

type ApplyCommentEffectResult struct {
	CommentEffect CommentEffect
	Points        PointAccount
}

type ApplyPostEffectResult struct {
	PostEffect PostEffect
	Points     PointAccount
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

type ApplyPostEffectRecordInput struct {
	ID           string
	PostID       postdomain.PostID
	EffectID     string
	UserID       userdomain.UserID
	PointsSpent  int
	InitialGrant int
	Now          time.Time
}

type ApplyPostEffectRecordResult struct {
	PostEffect PostEffect
	Points     PointAccount
}

type GrantPointsRecordInput struct {
	TransactionID string
	UserID        userdomain.UserID
	ActorID       userdomain.UserID
	Delta         int
	DailyCap      int
	Reason        string
	SourceType    string
	SourceID      string
	InitialGrant  int
	CreatedAt     time.Time
}

type GrantPointsRecordResult struct {
	Transaction *PointTransaction
	Points      PointAccount
	Granted     bool
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

func (uc *UseCase) SetPostReader(posts PostReader) {
	uc.posts = posts
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
	now := uc.now().UTC()
	_ = uc.GrantPoints(ctx, GrantPointsInput{
		UserID:     input.ActorID,
		ActorID:    input.ActorID,
		SourceType: PointSourceDailyActivity,
		SourceID:   now.Format(time.DateOnly),
	})
	points, err := uc.repository.GetOrCreatePointAccount(ctx, input.ActorID, InitialPointBalance, now)
	if err != nil {
		return GetMyPointsResult{}, fmt.Errorf("get point account: %w", err)
	}
	return GetMyPointsResult{Points: points}, nil
}

func (uc *UseCase) ListMyPointTransactions(ctx context.Context, input ListMyPointTransactionsInput) (ListMyPointTransactionsResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return ListMyPointTransactionsResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return ListMyPointTransactionsResult{}, err
	}
	if uc.repository == nil {
		return ListMyPointTransactionsResult{}, fmt.Errorf("effect repository is not configured")
	}
	transactions, err := uc.repository.ListPointTransactions(ctx, input.ActorID, limit+1, offset)
	if err != nil {
		return ListMyPointTransactionsResult{}, fmt.Errorf("list point transactions: %w", err)
	}
	transactions, hasMore := trimPointTransactionPage(transactions, limit)
	return ListMyPointTransactionsResult{
		Transactions: transactions,
		Limit:        limit,
		Offset:       offset,
		NextOffset:   offset + len(transactions),
		HasMore:      hasMore,
	}, nil
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

func (uc *UseCase) ApplyPostEffect(ctx context.Context, input ApplyPostEffectInput) (ApplyPostEffectResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return ApplyPostEffectResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return ApplyPostEffectResult{}, err
	}
	effectID, err := normalizeEffectID(input.EffectID)
	if err != nil {
		return ApplyPostEffectResult{}, err
	}
	if uc.posts == nil {
		return ApplyPostEffectResult{}, fmt.Errorf("post reader is not configured")
	}
	if uc.repository == nil {
		return ApplyPostEffectResult{}, fmt.Errorf("effect repository is not configured")
	}
	if _, err := uc.posts.FindVisibleByID(ctx, postID); err != nil {
		return ApplyPostEffectResult{}, fmt.Errorf("find visible post: %w", err)
	}
	effect, err := uc.repository.FindActiveEffectByID(ctx, effectID)
	if err != nil {
		return ApplyPostEffectResult{}, fmt.Errorf("find active effect: %w", err)
	}
	result, err := uc.repository.ApplyPostEffect(ctx, ApplyPostEffectRecordInput{
		ID:           uuid.NewString(),
		PostID:       postID,
		EffectID:     effect.ID,
		UserID:       input.ActorID,
		PointsSpent:  effect.CostPoints,
		InitialGrant: InitialPointBalance,
		Now:          uc.now().UTC(),
	})
	if err != nil {
		return ApplyPostEffectResult{}, fmt.Errorf("apply post effect: %w", err)
	}
	return ApplyPostEffectResult{
		PostEffect: result.PostEffect,
		Points:     result.Points,
	}, nil
}

func (uc *UseCase) GrantPoints(ctx context.Context, input GrantPointsInput) error {
	if uc.repository == nil {
		return fmt.Errorf("effect repository is not configured")
	}
	policy, ok := pointPolicies[strings.TrimSpace(input.SourceType)]
	if !ok {
		return apperr.New(apperr.CodeInvalidArgument, "point source type is invalid")
	}
	if strings.TrimSpace(input.UserID.String()) == "" {
		return apperr.New(apperr.CodeInvalidArgument, "point user is required")
	}
	sourceID := strings.TrimSpace(input.SourceID)
	if sourceID == "" {
		return apperr.New(apperr.CodeInvalidArgument, "point source id is required")
	}
	_, err := uc.repository.GrantPoints(ctx, GrantPointsRecordInput{
		TransactionID: uuid.NewString(),
		UserID:        input.UserID,
		ActorID:       input.ActorID,
		Delta:         policy.Delta,
		DailyCap:      policy.DailyCap,
		Reason:        policy.Reason,
		SourceType:    input.SourceType,
		SourceID:      sourceID,
		InitialGrant:  InitialPointBalance,
		CreatedAt:     uc.now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("grant points: %w", err)
	}
	return nil
}

func normalizePagination(limit int, offset int) (int, int, error) {
	if limit == 0 {
		limit = DefaultPointTransactionListLimit
	}
	if limit < 0 || offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "pagination is invalid")
	}
	if limit > MaxPointTransactionListLimit {
		limit = MaxPointTransactionListLimit
	}
	return limit, offset, nil
}

func trimPointTransactionPage(transactions []PointTransaction, limit int) ([]PointTransaction, bool) {
	if len(transactions) <= limit {
		return transactions, false
	}
	return transactions[:limit], true
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
