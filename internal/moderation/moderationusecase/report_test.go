package moderationusecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestReportPostCreatesPendingReport(t *testing.T) {
	now := testNow()
	post := mustPost(t, now)
	reporterID := userdomain.NewGeneratedUserID()
	var saved moderationdomain.ContentReport
	reports := &fakeContentReportRepository{
		createFunc: func(ctx context.Context, report moderationdomain.ContentReport) error {
			saved = report
			return nil
		},
	}
	posts := &fakePostRepository{
		findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
			if id != post.ID() {
				t.Fatalf("expected post %q, got %q", post.ID().String(), id.String())
			}
			return post, nil
		},
	}
	uc := NewReportUseCase(reports, posts, &fakeCommentRepository{}, func() time.Time { return now })

	result, err := uc.ReportPost(context.Background(), ReportPostInput{
		PostID:     post.ID().String(),
		ReporterID: reporterID,
		Reason:     "spam",
	})
	if err != nil {
		t.Fatalf("ReportPost returned error: %v", err)
	}

	if !reports.createCalled {
		t.Fatal("expected report repository to be called")
	}
	if result.Report.TargetType != moderationdomain.TargetTypePost.String() || result.Report.PostID != post.ID().String() {
		t.Fatalf("unexpected report target: %#v", result.Report)
	}
	if result.Report.ReporterID != reporterID.String() || result.Report.Status != moderationdomain.ReportStatusPending.String() {
		t.Fatalf("unexpected report dto: %#v", result.Report)
	}
	if saved.Reason().String() != "spam" {
		t.Fatalf("expected saved reason spam, got %q", saved.Reason().String())
	}
}

func TestReportCommentCreatesPendingReport(t *testing.T) {
	now := testNow()
	comment := mustComment(t, now)
	reporterID := userdomain.NewGeneratedUserID()
	reports := &fakeContentReportRepository{}
	comments := &fakeCommentRepository{
		findVisibleByIDFunc: func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
			if id != comment.ID() {
				t.Fatalf("expected comment %q, got %q", comment.ID().String(), id.String())
			}
			return comment, nil
		},
	}
	uc := NewReportUseCase(reports, &fakePostRepository{}, comments, func() time.Time { return now })

	result, err := uc.ReportComment(context.Background(), ReportCommentInput{
		CommentID:  comment.ID().String(),
		ReporterID: reporterID,
		Reason:     "abuse",
	})
	if err != nil {
		t.Fatalf("ReportComment returned error: %v", err)
	}

	if result.Report.TargetType != moderationdomain.TargetTypeComment.String() || result.Report.CommentID != comment.ID().String() {
		t.Fatalf("unexpected comment report: %#v", result.Report)
	}
}

func TestReportRejectsMissingReporterAndInvalidInput(t *testing.T) {
	uc := NewReportUseCase(&fakeContentReportRepository{}, &fakePostRepository{}, &fakeCommentRepository{}, time.Now)

	_, err := uc.ReportPost(context.Background(), ReportPostInput{
		PostID: postdomain.NewGeneratedPostID().String(),
		Reason: "spam",
	})
	if !hasAppCode(err, apperr.CodeUnauthenticated) {
		t.Fatalf("expected unauthenticated for missing reporter, got %v", err)
	}

	_, err = uc.ReportPost(context.Background(), ReportPostInput{
		PostID:     postdomain.NewGeneratedPostID().String(),
		ReporterID: userdomain.NewGeneratedUserID(),
		Reason:     " ",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for blank reason, got %v", err)
	}

	_, err = uc.ReportComment(context.Background(), ReportCommentInput{
		CommentID:  "not-a-uuid",
		ReporterID: userdomain.NewGeneratedUserID(),
		Reason:     "spam",
	})
	if !hasAppCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument for invalid comment id, got %v", err)
	}
}

func TestReportPropagatesTargetAndRepositoryErrors(t *testing.T) {
	postID := postdomain.NewGeneratedPostID()
	uc := NewReportUseCase(
		&fakeContentReportRepository{},
		&fakePostRepository{
			findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
				return nil, apperr.New(apperr.CodeNotFound, "post not found")
			},
		},
		&fakeCommentRepository{},
		time.Now,
	)

	_, err := uc.ReportPost(context.Background(), ReportPostInput{
		PostID:     postID.String(),
		ReporterID: userdomain.NewGeneratedUserID(),
		Reason:     "spam",
	})
	if !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found from post repository, got %v", err)
	}

	uc = NewReportUseCase(
		&fakeContentReportRepository{
			createFunc: func(ctx context.Context, report moderationdomain.ContentReport) error {
				return apperr.New(apperr.CodeConflict, "content report already exists")
			},
		},
		&fakePostRepository{
			findVisibleByIDFunc: func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
				return mustPost(t, testNow()), nil
			},
		},
		&fakeCommentRepository{},
		time.Now,
	)

	_, err = uc.ReportPost(context.Background(), ReportPostInput{
		PostID:     postID.String(),
		ReporterID: userdomain.NewGeneratedUserID(),
		Reason:     "spam",
	})
	if !hasAppCode(err, apperr.CodeConflict) {
		t.Fatalf("expected conflict from report repository, got %v", err)
	}
}

type fakeContentReportRepository struct {
	createCalled bool
	createFunc   func(ctx context.Context, report moderationdomain.ContentReport) error
}

func (f *fakeContentReportRepository) CreateReport(ctx context.Context, report moderationdomain.ContentReport) error {
	f.createCalled = true
	if f.createFunc != nil {
		return f.createFunc(ctx, report)
	}
	return nil
}

type fakePostRepository struct {
	findVisibleByIDFunc func(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error)
}

func (f *fakePostRepository) FindVisibleByID(ctx context.Context, id postdomain.PostID) (*postdomain.Post, error) {
	if f.findVisibleByIDFunc != nil {
		return f.findVisibleByIDFunc(ctx, id)
	}
	return nil, apperr.New(apperr.CodeNotFound, "post not found")
}

type fakeCommentRepository struct {
	findVisibleByIDFunc func(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error)
}

func (f *fakeCommentRepository) FindVisibleByID(ctx context.Context, id commentdomain.CommentID) (*commentdomain.Comment, error) {
	if f.findVisibleByIDFunc != nil {
		return f.findVisibleByIDFunc(ctx, id)
	}
	return nil, apperr.New(apperr.CodeNotFound, "comment not found")
}

func mustPost(t *testing.T, now time.Time) *postdomain.Post {
	t.Helper()
	title, err := postdomain.NewPostTitle("Post")
	if err != nil {
		t.Fatalf("NewPostTitle returned error: %v", err)
	}
	body, err := postdomain.NewPostBody("Post body")
	if err != nil {
		t.Fatalf("NewPostBody returned error: %v", err)
	}
	post, err := postdomain.NewPost(postdomain.NewGeneratedPostID(), communitydomain.NewGeneratedCommunityID(), userdomain.NewGeneratedUserID(), title, body, now)
	if err != nil {
		t.Fatalf("NewPost returned error: %v", err)
	}
	return post
}

func mustComment(t *testing.T, now time.Time) *commentdomain.Comment {
	t.Helper()
	body, err := commentdomain.NewCommentBody("Comment body")
	if err != nil {
		t.Fatalf("NewCommentBody returned error: %v", err)
	}
	comment, err := commentdomain.NewComment(commentdomain.NewGeneratedCommentID(), postdomain.NewGeneratedPostID(), userdomain.NewGeneratedUserID(), nil, body, now)
	if err != nil {
		t.Fatalf("NewComment returned error: %v", err)
	}
	return comment
}

func testNow() time.Time {
	return time.Date(2026, 6, 2, 4, 0, 0, 0, time.UTC)
}

func hasAppCode(err error, code apperr.Code) bool {
	if err == nil {
		return false
	}

	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return appErr.Code() == code
	}
	return false
}
