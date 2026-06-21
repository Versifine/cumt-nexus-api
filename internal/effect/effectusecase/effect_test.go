package effectusecase

import (
	"context"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestListEffectsCatalogReturnsActiveEffects(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	repository := &fakeRepository{
		effects: []Effect{{
			ID:           "sparkle",
			Name:         "Sparkle",
			Description:  "Sparkle burst",
			CostPoints:   10,
			AnimationKey: "sparkle",
			IsActive:     true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}},
	}
	uc := NewUseCase(repository, &fakeCommentReader{}, time.Now)

	result, err := uc.ListEffectsCatalog(context.Background())
	if err != nil {
		t.Fatalf("ListEffectsCatalog returned error: %v", err)
	}
	if !repository.listCalled {
		t.Fatal("expected repository ListActiveEffects to be called")
	}
	if len(result.Effects) != 1 || result.Effects[0].ID != "sparkle" {
		t.Fatalf("unexpected effects: %#v", result.Effects)
	}
}

func TestGetMyPointsEnsuresAccount(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	repository := &fakeRepository{
		points: PointAccount{
			UserID:         userID.String(),
			Balance:        100,
			LifetimeEarned: 100,
			UpdatedAt:      now,
		},
	}
	uc := NewUseCase(repository, &fakeCommentReader{}, func() time.Time { return now })

	result, err := uc.GetMyPoints(context.Background(), GetMyPointsInput{ActorID: userID})
	if err != nil {
		t.Fatalf("GetMyPoints returned error: %v", err)
	}
	if !repository.getPointsCalled || repository.getPointsUserID != userID || repository.initialBalance != InitialPointBalance {
		t.Fatalf("unexpected get points call: %#v", repository)
	}
	if repository.grantPointsInput.SourceType != PointSourceDailyActivity || repository.grantPointsInput.SourceID != "2026-06-09" {
		t.Fatalf("expected get points to attempt daily activity grant, got %#v", repository.grantPointsInput)
	}
	if result.Points.Balance != 100 || result.Points.LifetimeEarned != 100 {
		t.Fatalf("unexpected points result: %#v", result.Points)
	}
}

func TestGetMyPointsRequiresAuth(t *testing.T) {
	uc := NewUseCase(&fakeRepository{}, &fakeCommentReader{}, time.Now)

	_, err := uc.GetMyPoints(context.Background(), GetMyPointsInput{})
	if !apperr.IsCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestListMyPointTransactionsReturnsPaginatedTransactions(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	repository := &fakeRepository{
		transactions: []PointTransaction{
			{ID: "1f37d0f3-f559-4ee0-ae11-f60f258a1c9a", UserID: userID.String(), Delta: 5, BalanceAfter: 105, Reason: "daily_visit", SourceType: "daily_activity", SourceID: "2026-06-14", CreatedAt: now},
			{ID: "27c03541-295e-4852-b801-c8ac6ad7911c", UserID: userID.String(), Delta: -10, BalanceAfter: 95, Reason: "comment_effect", SourceType: "comment_effect", SourceID: "effect-1", CreatedAt: now.Add(-time.Minute)},
		},
	}
	uc := NewUseCase(repository, &fakeCommentReader{}, time.Now)

	result, err := uc.ListMyPointTransactions(context.Background(), ListMyPointTransactionsInput{
		ActorID: userID,
		Limit:   1,
		Offset:  3,
	})
	if err != nil {
		t.Fatalf("ListMyPointTransactions returned error: %v", err)
	}
	if !repository.listTransactionsCalled || repository.listTransactionsUserID != userID || repository.listTransactionsLimit != 2 || repository.listTransactionsOffset != 3 {
		t.Fatalf("unexpected repository call: %#v", repository)
	}
	if len(result.Transactions) != 1 || !result.HasMore || result.NextOffset != 4 {
		t.Fatalf("unexpected paginated result: %#v", result)
	}
}

func TestListMyPointTransactionsRequiresAuth(t *testing.T) {
	uc := NewUseCase(&fakeRepository{}, &fakeCommentReader{}, time.Now)

	_, err := uc.ListMyPointTransactions(context.Background(), ListMyPointTransactionsInput{})
	if !apperr.IsCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestGrantPointsUsesPointPolicy(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	actorID := userdomain.NewGeneratedUserID()
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	repository := &fakeRepository{}
	uc := NewUseCase(repository, &fakeCommentReader{}, func() time.Time { return now })

	err := uc.GrantPoints(context.Background(), GrantPointsInput{
		UserID:     userID,
		ActorID:    actorID,
		SourceType: PointSourcePostPublish,
		SourceID:   "post-1",
	})
	if err != nil {
		t.Fatalf("GrantPoints returned error: %v", err)
	}
	if repository.grantPointsInput.UserID != userID || repository.grantPointsInput.ActorID != actorID {
		t.Fatalf("unexpected grant points user fields: %#v", repository.grantPointsInput)
	}
	if repository.grantPointsInput.Delta != 5 || repository.grantPointsInput.DailyCap != 25 || repository.grantPointsInput.Reason != PointReasonPostPublish {
		t.Fatalf("unexpected post publish point policy: %#v", repository.grantPointsInput)
	}
	if repository.grantPointsInput.InitialGrant != InitialPointBalance || !repository.grantPointsInput.CreatedAt.Equal(now) {
		t.Fatalf("unexpected point account metadata: %#v", repository.grantPointsInput)
	}
}

func TestGrantPointsRejectsInvalidSource(t *testing.T) {
	uc := NewUseCase(&fakeRepository{}, &fakeCommentReader{}, time.Now)

	err := uc.GrantPoints(context.Background(), GrantPointsInput{
		UserID:     userdomain.NewGeneratedUserID(),
		SourceType: "unknown",
		SourceID:   "source-1",
	})
	if !apperr.IsCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument, got %v", err)
	}
}

func TestApplyCommentEffectValidatesCommentAndDeductsPoints(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	commentID := commentdomain.NewGeneratedCommentID()
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	repository := &fakeRepository{
		effect: Effect{
			ID:           "sparkle",
			Name:         "Sparkle",
			CostPoints:   10,
			AnimationKey: "sparkle",
			IsActive:     true,
		},
		applyResult: ApplyCommentEffectRecordResult{
			CommentEffect: CommentEffect{
				ID:          "35d6502f-12c8-4235-96d0-e732575181ab",
				CommentID:   commentID.String(),
				EffectID:    "sparkle",
				UserID:      userID.String(),
				PointsSpent: 10,
				CreatedAt:   now,
			},
			Points: PointAccount{
				UserID:         userID.String(),
				Balance:        90,
				LifetimeEarned: 100,
				LifetimeSpent:  10,
				UpdatedAt:      now,
			},
		},
	}
	comments := &fakeCommentReader{
		comment: mustComment(t, commentID, userID, now),
	}
	uc := NewUseCase(repository, comments, func() time.Time { return now })

	result, err := uc.ApplyCommentEffect(context.Background(), ApplyCommentEffectInput{
		ActorID:   userID,
		CommentID: commentID.String(),
		EffectID:  "Sparkle",
	})
	if err != nil {
		t.Fatalf("ApplyCommentEffect returned error: %v", err)
	}
	if !comments.findCalled || comments.commentID != commentID {
		t.Fatalf("expected comment lookup for %q, got %#v", commentID.String(), comments)
	}
	if repository.findEffectID != "sparkle" {
		t.Fatalf("expected normalized effect id sparkle, got %q", repository.findEffectID)
	}
	if repository.applyInput.CommentID != commentID || repository.applyInput.EffectID != "sparkle" || repository.applyInput.PointsSpent != 10 {
		t.Fatalf("unexpected apply input: %#v", repository.applyInput)
	}
	if result.Points.Balance != 90 || result.CommentEffect.PointsSpent != 10 {
		t.Fatalf("unexpected apply result: %#v", result)
	}
}

func TestApplyCommentEffectRejectsInvalidInput(t *testing.T) {
	uc := NewUseCase(&fakeRepository{}, &fakeCommentReader{}, time.Now)

	tests := []struct {
		name  string
		input ApplyCommentEffectInput
		code  apperr.Code
	}{
		{name: "missing actor", input: ApplyCommentEffectInput{}, code: apperr.CodeUnauthenticated},
		{name: "invalid comment", input: ApplyCommentEffectInput{ActorID: userdomain.NewGeneratedUserID(), CommentID: "bad", EffectID: "sparkle"}, code: apperr.CodeInvalidArgument},
		{name: "invalid effect", input: ApplyCommentEffectInput{ActorID: userdomain.NewGeneratedUserID(), CommentID: commentdomain.NewGeneratedCommentID().String(), EffectID: "../x"}, code: apperr.CodeInvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.ApplyCommentEffect(context.Background(), tt.input)
			if !apperr.IsCode(err, tt.code) {
				t.Fatalf("expected %s, got %v", tt.code, err)
			}
		})
	}
}

func TestApplyCommentEffectPropagatesInsufficientPoints(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	commentID := commentdomain.NewGeneratedCommentID()
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	repository := &fakeRepository{
		effect: Effect{
			ID:         "sparkle",
			CostPoints: 10,
			IsActive:   true,
		},
		applyErr: apperr.New(apperr.CodeForbidden, "insufficient points"),
	}
	uc := NewUseCase(repository, &fakeCommentReader{comment: mustComment(t, commentID, userID, now)}, time.Now)

	_, err := uc.ApplyCommentEffect(context.Background(), ApplyCommentEffectInput{
		ActorID:   userID,
		CommentID: commentID.String(),
		EffectID:  "sparkle",
	})
	if !apperr.IsCode(err, apperr.CodeForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestApplyPostEffectValidatesPostAndDeductsPoints(t *testing.T) {
	userID := userdomain.NewGeneratedUserID()
	postID := postdomain.NewGeneratedPostID()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	repository := &fakeRepository{
		effect: Effect{
			ID:           "useful",
			Name:         "有用",
			CostPoints:   5,
			AnimationKey: "useful",
			Emoji:        "👍",
			IsActive:     true,
		},
		applyPostResult: ApplyPostEffectRecordResult{
			PostEffect: PostEffect{
				ID:          "e673c232-d76e-46ea-8bc2-1f9aefac5868",
				PostID:      postID.String(),
				EffectID:    "useful",
				UserID:      userID.String(),
				PointsSpent: 5,
				CreatedAt:   now,
			},
			Points: PointAccount{
				UserID:         userID.String(),
				Balance:        95,
				LifetimeEarned: 100,
				LifetimeSpent:  5,
				UpdatedAt:      now,
			},
		},
	}
	posts := &fakePostReader{
		post: mustPost(t, postID, userID, now),
	}
	uc := NewUseCase(repository, &fakeCommentReader{}, func() time.Time { return now })
	uc.SetPostReader(posts)

	result, err := uc.ApplyPostEffect(context.Background(), ApplyPostEffectInput{
		ActorID:  userID,
		PostID:   postID.String(),
		EffectID: "Useful",
	})
	if err != nil {
		t.Fatalf("ApplyPostEffect returned error: %v", err)
	}
	if !posts.findCalled || posts.postID != postID {
		t.Fatalf("expected post lookup for %q, got %#v", postID.String(), posts)
	}
	if repository.findEffectID != "useful" {
		t.Fatalf("expected normalized effect id useful, got %q", repository.findEffectID)
	}
	if repository.applyPostInput.PostID != postID || repository.applyPostInput.EffectID != "useful" || repository.applyPostInput.PointsSpent != 5 {
		t.Fatalf("unexpected apply post input: %#v", repository.applyPostInput)
	}
	if result.Points.Balance != 95 || result.PostEffect.PointsSpent != 5 {
		t.Fatalf("unexpected apply post result: %#v", result)
	}
}

func TestApplyPostEffectRejectsInvalidInput(t *testing.T) {
	uc := NewUseCase(&fakeRepository{}, &fakeCommentReader{}, time.Now)

	tests := []struct {
		name  string
		input ApplyPostEffectInput
		code  apperr.Code
	}{
		{name: "missing actor", input: ApplyPostEffectInput{}, code: apperr.CodeUnauthenticated},
		{name: "invalid post", input: ApplyPostEffectInput{ActorID: userdomain.NewGeneratedUserID(), PostID: "bad", EffectID: "useful"}, code: apperr.CodeInvalidArgument},
		{name: "invalid effect", input: ApplyPostEffectInput{ActorID: userdomain.NewGeneratedUserID(), PostID: postdomain.NewGeneratedPostID().String(), EffectID: "../x"}, code: apperr.CodeInvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.ApplyPostEffect(context.Background(), tt.input)
			if !apperr.IsCode(err, tt.code) {
				t.Fatalf("expected %s, got %v", tt.code, err)
			}
		})
	}
}

type fakeRepository struct {
	listCalled bool
	effects    []Effect
	listErr    error

	findEffectID string
	effect       Effect
	findErr      error

	getPointsCalled bool
	getPointsUserID userdomain.UserID
	initialBalance  int
	points          PointAccount
	getPointsErr    error

	listTransactionsCalled bool
	listTransactionsUserID userdomain.UserID
	listTransactionsLimit  int
	listTransactionsOffset int
	transactions           []PointTransaction
	listTransactionsErr    error

	applyInput  ApplyCommentEffectRecordInput
	applyResult ApplyCommentEffectRecordResult
	applyErr    error

	applyPostInput  ApplyPostEffectRecordInput
	applyPostResult ApplyPostEffectRecordResult
	applyPostErr    error

	grantPointsInput  GrantPointsRecordInput
	grantPointsResult GrantPointsRecordResult
	grantPointsErr    error
}

func (f *fakeRepository) ListActiveEffects(ctx context.Context) ([]Effect, error) {
	_ = ctx
	f.listCalled = true
	return f.effects, f.listErr
}

func (f *fakeRepository) FindActiveEffectByID(ctx context.Context, effectID string) (Effect, error) {
	_ = ctx
	f.findEffectID = effectID
	return f.effect, f.findErr
}

func (f *fakeRepository) GetOrCreatePointAccount(ctx context.Context, userID userdomain.UserID, initialBalance int, now time.Time) (PointAccount, error) {
	_ = ctx
	_ = now
	f.getPointsCalled = true
	f.getPointsUserID = userID
	f.initialBalance = initialBalance
	return f.points, f.getPointsErr
}

func (f *fakeRepository) ListPointTransactions(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]PointTransaction, error) {
	_ = ctx
	f.listTransactionsCalled = true
	f.listTransactionsUserID = userID
	f.listTransactionsLimit = limit
	f.listTransactionsOffset = offset
	return f.transactions, f.listTransactionsErr
}

func (f *fakeRepository) ApplyCommentEffect(ctx context.Context, input ApplyCommentEffectRecordInput) (ApplyCommentEffectRecordResult, error) {
	_ = ctx
	f.applyInput = input
	return f.applyResult, f.applyErr
}

func (f *fakeRepository) ApplyPostEffect(ctx context.Context, input ApplyPostEffectRecordInput) (ApplyPostEffectRecordResult, error) {
	_ = ctx
	f.applyPostInput = input
	return f.applyPostResult, f.applyPostErr
}

func (f *fakeRepository) GrantPoints(ctx context.Context, input GrantPointsRecordInput) (GrantPointsRecordResult, error) {
	_ = ctx
	f.grantPointsInput = input
	return f.grantPointsResult, f.grantPointsErr
}

type fakeCommentReader struct {
	findCalled bool
	commentID  commentdomain.CommentID
	comment    *commentdomain.Comment
	err        error
}

func (f *fakeCommentReader) FindVisibleByID(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
	_ = ctx
	f.findCalled = true
	f.commentID = id
	return f.comment, f.err
}

type fakePostReader struct {
	findCalled bool
	postID     postdomain.PostID
	post       *postdomain.Post
	err        error
}

func (f *fakePostReader) FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
	_ = ctx
	f.findCalled = true
	f.postID = id
	return f.post, f.err
}

func mustComment(t *testing.T, commentID commentdomain.CommentID, userID userdomain.UserID, now time.Time) *commentdomain.Comment {
	t.Helper()

	body, err := commentdomain.NewCommentBody("hello")
	if err != nil {
		t.Fatalf("new comment body: %v", err)
	}
	comment, err := commentdomain.NewComment(commentID, postdomain.NewGeneratedPostID(), userID, nil, body, now)
	if err != nil {
		t.Fatalf("new comment: %v", err)
	}
	return comment
}

func mustPost(t *testing.T, postID postdomain.PostID, userID userdomain.UserID, now time.Time) *postdomain.Post {
	t.Helper()

	title, err := postdomain.NewPostTitle("hello")
	if err != nil {
		t.Fatalf("new post title: %v", err)
	}
	body, err := postdomain.NewPostBody("post body")
	if err != nil {
		t.Fatalf("new post body: %v", err)
	}
	post, err := postdomain.NewPost(postID, communitydomain.NewGeneratedCommunityID(), userID, title, body, now)
	if err != nil {
		t.Fatalf("new post: %v", err)
	}
	return post
}
