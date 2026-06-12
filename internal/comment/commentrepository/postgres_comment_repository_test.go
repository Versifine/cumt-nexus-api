package commentrepository

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentusecase"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCommentRepositoryCreateFindListAndNotFound(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresCommentRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "comment-"+randomSuffix())
	postID := insertTestPost(ctx, t, pool, communityID, authorID, "Comment Post")
	otherPostID := insertTestPost(ctx, t, pool, communityID, authorID, "Other Post")

	parentComment := mustComment(t, postID, authorID, nil, "Parent", now)
	if err := repo.Create(ctx, *parentComment); err != nil {
		t.Fatalf("Create parent comment returned error: %v", err)
	}
	cleanupComment(ctx, t, pool, parentComment.ID())

	parentID := parentComment.ID()
	newerComment := mustComment(t, postID, authorID, &parentID, "Reply", now.Add(time.Minute))
	otherComment := mustComment(t, otherPostID, authorID, nil, "Other", now.Add(2*time.Minute))
	if err := repo.Create(ctx, *newerComment); err != nil {
		t.Fatalf("Create newer comment returned error: %v", err)
	}
	cleanupComment(ctx, t, pool, newerComment.ID())
	if err := repo.Create(ctx, *otherComment); err != nil {
		t.Fatalf("Create other comment returned error: %v", err)
	}
	cleanupComment(ctx, t, pool, otherComment.ID())

	got, err := repo.FindVisibleByID(ctx, newerComment.ID())
	if err != nil {
		t.Fatalf("FindVisibleByID returned error: %v", err)
	}
	if got.ID() != newerComment.ID() {
		t.Fatalf("expected comment %q, got %q", newerComment.ID().String(), got.ID().String())
	}
	if gotParentID, ok := got.ParentID(); !ok || gotParentID != parentID {
		t.Fatalf("expected parent %q, got %q present=%t", parentID.String(), gotParentID.String(), ok)
	}

	comments, err := repo.ListVisibleByPost(ctx, postID, commentusecase.CommentListSortNew, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleByPost returned error: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected two comments, got %d", len(comments))
	}
	if comments[0].ID() != newerComment.ID() || comments[1].ID() != parentComment.ID() {
		t.Fatalf("expected newest-first ordering, got %#v", []commentdomain.CommentID{comments[0].ID(), comments[1].ID()})
	}

	treeComments, err := repo.ListVisibleTreeByPost(ctx, postID)
	if err != nil {
		t.Fatalf("ListVisibleTreeByPost returned error: %v", err)
	}
	if len(treeComments) != 2 {
		t.Fatalf("expected two tree comments, got %d", len(treeComments))
	}
	if treeComments[0].ID() != newerComment.ID() || treeComments[1].ID() != parentComment.ID() {
		t.Fatalf("expected tree comments newest-first, got %#v", []commentdomain.CommentID{treeComments[0].ID(), treeComments[1].ID()})
	}

	if _, err := repo.FindVisibleByID(ctx, commentdomain.NewGeneratedCommentID()); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing comment, got %v", err)
	}
}

func TestPostgresCommentRepositoryMapsForeignKeyFailure(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresCommentRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	comment := mustComment(t, postdomain.NewGeneratedPostID(), authorID, nil, "Missing post", now)
	if err := repo.Create(ctx, *comment); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing related post, got %v", err)
	}
}

func TestPostgresCommentRepositoryLoadMetadataByCommentIDsReturnsAuthorProfile(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresCommentRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	updateTestUserProfile(ctx, t, pool, authorID, "Alice", "https://example.com/avatar.jpg", "Backend builder")
	communityID := insertTestCommunity(ctx, t, pool, authorID, "comment-meta-"+randomSuffix())
	postID := insertTestPost(ctx, t, pool, communityID, authorID, "Comment metadata post")
	comment := mustComment(t, postID, authorID, nil, "Metadata comment", now)
	if err := repo.Create(ctx, *comment); err != nil {
		t.Fatalf("Create comment returned error: %v", err)
	}
	cleanupComment(ctx, t, pool, comment.ID())

	metadata, err := repo.LoadMetadataByCommentIDs(ctx, []commentdomain.CommentID{comment.ID()})
	if err != nil {
		t.Fatalf("LoadMetadataByCommentIDs returned error: %v", err)
	}
	got := metadata[comment.ID()]
	if got.Author.DisplayName != "Alice" || got.Author.AvatarURL != "https://example.com/avatar.jpg" || got.Author.Headline != "Backend builder" {
		t.Fatalf("expected author profile fields, got %#v", got.Author)
	}
}

func TestPostgresCommentRepositoryListCommentsByCommunityForManagement(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresCommentRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "manage-comment-"+randomSuffix())
	otherCommunityID := insertTestCommunity(ctx, t, pool, authorID, "manage-comment-"+randomSuffix())
	postID := insertTestPost(ctx, t, pool, communityID, authorID, "Manage Comment Post")
	otherPostID := insertTestPost(ctx, t, pool, otherCommunityID, authorID, "Other Manage Comment Post")

	visibleComment := mustComment(t, postID, authorID, nil, "Visible manage comment", now)
	removedComment := mustCommentWithStatus(t, postID, authorID, nil, "Removed manage comment", commentdomain.CommentStatusRemoved, now.Add(time.Minute))
	otherComment := mustCommentWithStatus(t, otherPostID, authorID, nil, "Other removed manage comment", commentdomain.CommentStatusRemoved, now.Add(2*time.Minute))
	for _, comment := range []*commentdomain.Comment{visibleComment, removedComment, otherComment} {
		if err := repo.Create(ctx, *comment); err != nil {
			t.Fatalf("Create comment %q returned error: %v", comment.Body().String(), err)
		}
		cleanupComment(ctx, t, pool, comment.ID())
	}

	status := commentdomain.CommentStatusRemoved
	comments, err := repo.ListCommentsByCommunityForManagement(ctx, communityID, &status, 20, 0)
	if err != nil {
		t.Fatalf("ListCommentsByCommunityForManagement returned error: %v", err)
	}
	if len(comments) != 1 || comments[0].ID() != removedComment.ID() {
		t.Fatalf("expected only removed comment, got %#v", comments)
	}

	allComments, err := repo.ListCommentsByCommunityForManagement(ctx, communityID, nil, 20, 0)
	if err != nil {
		t.Fatalf("ListCommentsByCommunityForManagement all returned error: %v", err)
	}
	if len(allComments) != 2 || allComments[0].ID() != removedComment.ID() || allComments[1].ID() != visibleComment.ID() {
		t.Fatalf("expected newest-first comments in community, got %#v", allComments)
	}
}

func TestPostgresCommentRepositoryUpdateContentAndMarkDeleted(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresCommentRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "comment-update-"+randomSuffix())
	postID := insertTestPost(ctx, t, pool, communityID, authorID, "Comment Update Post")
	comment := mustComment(t, postID, authorID, nil, "Original", now)
	if err := repo.Create(ctx, *comment); err != nil {
		t.Fatalf("Create comment returned error: %v", err)
	}
	cleanupComment(ctx, t, pool, comment.ID())

	if err := comment.EditBody(mustCommentBody(t, "Updated body"), now.Add(time.Minute)); err != nil {
		t.Fatalf("EditBody returned error: %v", err)
	}
	if err := repo.UpdateContent(ctx, *comment); err != nil {
		t.Fatalf("UpdateContent returned error: %v", err)
	}

	updated, err := repo.FindVisibleByID(ctx, comment.ID())
	if err != nil {
		t.Fatalf("FindVisibleByID after update returned error: %v", err)
	}
	if updated.Body().String() != "Updated body" {
		t.Fatalf("expected updated body, got %q", updated.Body().String())
	}

	if err := comment.MarkDeleted(now.Add(2 * time.Minute)); err != nil {
		t.Fatalf("MarkDeleted returned error: %v", err)
	}
	if err := repo.MarkDeleted(ctx, *comment); err != nil {
		t.Fatalf("MarkDeleted repository returned error: %v", err)
	}
	if _, err := repo.FindVisibleByID(ctx, comment.ID()); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found after delete, got %v", err)
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM comments WHERE id = $1::uuid`, comment.ID().String()).Scan(&status); err != nil {
		t.Fatalf("query deleted comment status: %v", err)
	}
	if status != commentdomain.CommentStatusDeleted.String() {
		t.Fatalf("expected deleted status, got %q", status)
	}
}

func TestPostgresCommentRepositoryListVisibleByPostSortsByVotes(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresCommentRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "comment-sort-"+randomSuffix())
	postID := insertTestPost(ctx, t, pool, communityID, authorID, "Comment Sort Post")

	lowScore := mustComment(t, postID, authorID, nil, "Low score", now.Add(2*time.Minute))
	topScore := mustComment(t, postID, authorID, nil, "Top score", now.Add(time.Minute))
	controversial := mustComment(t, postID, authorID, nil, "Controversial", now)
	for _, comment := range []*commentdomain.Comment{lowScore, topScore, controversial} {
		if err := repo.Create(ctx, *comment); err != nil {
			t.Fatalf("Create comment returned error: %v", err)
		}
		cleanupComment(ctx, t, pool, comment.ID())
	}

	insertTestCommentVote(ctx, t, pool, lowScore.ID(), insertTestUser(ctx, t, pool), 1)
	for i := 0; i < 5; i++ {
		insertTestCommentVote(ctx, t, pool, topScore.ID(), insertTestUser(ctx, t, pool), 1)
	}
	insertTestCommentVote(ctx, t, pool, topScore.ID(), insertTestUser(ctx, t, pool), -1)
	for i := 0; i < 3; i++ {
		insertTestCommentVote(ctx, t, pool, controversial.ID(), insertTestUser(ctx, t, pool), 1)
		insertTestCommentVote(ctx, t, pool, controversial.ID(), insertTestUser(ctx, t, pool), -1)
	}

	topComments, err := repo.ListVisibleByPost(ctx, postID, commentusecase.CommentListSortTop, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleByPost top returned error: %v", err)
	}
	if len(topComments) != 3 || topComments[0].ID() != topScore.ID() {
		t.Fatalf("expected top score comment first, got %#v", commentIDs(topComments))
	}

	controversialComments, err := repo.ListVisibleByPost(ctx, postID, commentusecase.CommentListSortControversial, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleByPost controversial returned error: %v", err)
	}
	if len(controversialComments) != 3 || controversialComments[0].ID() != controversial.ID() {
		t.Fatalf("expected controversial comment first, got %#v", commentIDs(controversialComments))
	}

	oldComments, err := repo.ListVisibleByPost(ctx, postID, commentusecase.CommentListSortOld, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleByPost old returned error: %v", err)
	}
	if len(oldComments) != 3 || oldComments[0].ID() != controversial.ID() || oldComments[2].ID() != lowScore.ID() {
		t.Fatalf("expected old comment order, got %#v", commentIDs(oldComments))
	}
}

func TestPostgresCommentRepositoryReplaceAndListContentRefs(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresCommentRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "comment-content-refs-"+randomSuffix())
	postID := insertTestPost(ctx, t, pool, communityID, authorID, "Comment content refs post")
	comment := mustComment(t, postID, authorID, nil, "Comment content refs", now)
	if err := repo.Create(ctx, *comment); err != nil {
		t.Fatalf("Create comment returned error: %v", err)
	}
	cleanupComment(ctx, t, pool, comment.ID())

	refs := []postusecase.ContentRef{
		{Kind: postusecase.ContentRefKindLink, RefID: "https://example.com/comment-one"},
		{Kind: postusecase.ContentRefKindEmbed, RefID: "https://www.youtube.com/watch?v=comment-one"},
	}
	if err := repo.ReplaceCommentContentRefs(ctx, comment.ID(), refs, now); err != nil {
		t.Fatalf("ReplaceCommentContentRefs returned error: %v", err)
	}
	got, err := repo.ListCommentContentRefsByCommentIDs(ctx, []commentdomain.CommentID{comment.ID(), commentdomain.NewGeneratedCommentID()})
	if err != nil {
		t.Fatalf("ListCommentContentRefsByCommentIDs returned error: %v", err)
	}
	assertCommentRepositoryContentRefs(t, got[comment.ID()], refs)

	replacement := []postusecase.ContentRef{
		{Kind: postusecase.ContentRefKindImage, RefID: "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a"},
	}
	if err := repo.ReplaceCommentContentRefs(ctx, comment.ID(), replacement, now.Add(time.Minute)); err != nil {
		t.Fatalf("ReplaceCommentContentRefs replacement returned error: %v", err)
	}
	got, err = repo.ListCommentContentRefsByCommentIDs(ctx, []commentdomain.CommentID{comment.ID()})
	if err != nil {
		t.Fatalf("ListCommentContentRefsByCommentIDs after replace returned error: %v", err)
	}
	assertCommentRepositoryContentRefs(t, got[comment.ID()], replacement)

	if err := repo.ReplaceCommentContentRefs(ctx, comment.ID(), nil, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ReplaceCommentContentRefs clear returned error: %v", err)
	}
	got, err = repo.ListCommentContentRefsByCommentIDs(ctx, []commentdomain.CommentID{comment.ID()})
	if err != nil {
		t.Fatalf("ListCommentContentRefsByCommentIDs after clear returned error: %v", err)
	}
	if len(got[comment.ID()]) != 0 {
		t.Fatalf("expected cleared comment content refs, got %#v", got[comment.ID()])
	}
}

func newTestPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	pool, err := db.Open(ctx, testPostgresConfig())
	if err != nil {
		t.Skipf("skip repository integration test: open postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.Ping(ctx, pool); err != nil {
		t.Skipf("skip repository integration test: ping postgres: %v", err)
	}

	requireCommentSchema(ctx, t, pool)

	return ctx, pool
}

func requireCommentSchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, table := range []string{"users", "communities", "posts", "comments", "comment_votes", "comment_content_refs"} {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'public'
					AND table_name = $1
			)
		`, table).Scan(&exists); err != nil {
			t.Fatalf("check table %s exists: %v", table, err)
		}
		if !exists {
			t.Skipf("%s table does not exist; run go run ./cmd/migrate up before repository tests", table)
		}
	}
}

func testPostgresConfig() config.PostgresConfig {
	return config.PostgresConfig{
		Host:            envString("POSTGRES_HOST", "localhost"),
		Port:            envInt("POSTGRES_PORT", 5432),
		User:            envString("POSTGRES_USER", "postgres"),
		Password:        envString("POSTGRES_PASSWORD", "postgres"),
		Database:        envString("POSTGRES_DATABASE", "cumt_nexus"),
		SSLMode:         envString("POSTGRES_SSL_MODE", "disable"),
		MaxConns:        5,
		MaxConnLifetime: time.Minute,
		MaxConnIdleTime: time.Minute,
	}
}

func envString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func insertTestUser(ctx context.Context, t *testing.T, pool *pgxpool.Pool) userdomain.UserID {
	t.Helper()

	id := userdomain.NewGeneratedUserID()
	username := "comment_repo_" + randomSuffix()
	_, err := pool.Exec(ctx, `
		INSERT INTO users (
			id,
			username,
			password_hash,
			status,
			is_platform_staff,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2, $3, 'active', false, $4, $4)
	`, id.String(), username, "hashed-password-"+username, testNow())
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup test user %q: %v", id.String(), err)
		}
	})

	return id
}

func updateTestUserProfile(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID userdomain.UserID, displayName string, avatarURL string, headline string) {
	t.Helper()

	if _, err := pool.Exec(ctx, `
		UPDATE users
		SET display_name = $2, avatar_url = $3, headline = $4
		WHERE id = $1::uuid
	`, userID.String(), displayName, avatarURL, headline); err != nil {
		t.Fatalf("update test user profile: %v", err)
	}
}

func insertTestCommunity(ctx context.Context, t *testing.T, pool *pgxpool.Pool, createdBy userdomain.UserID, rawSlug string) communitydomain.CommunityID {
	t.Helper()

	id := communitydomain.NewGeneratedCommunityID()
	_, err := pool.Exec(ctx, `
		INSERT INTO communities (
			id,
			slug,
			name,
			description,
			kind,
			status,
			visibility,
			created_by,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2, $3, '', 'user_created', 'active', 'public', $4::uuid, $5, $5)
	`, id.String(), rawSlug, "Comment Repo "+rawSlug, createdBy.String(), testNow())
	if err != nil {
		t.Fatalf("insert test community: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM communities WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup test community %q: %v", id.String(), err)
		}
	})

	return id
}

func insertTestPost(ctx context.Context, t *testing.T, pool *pgxpool.Pool, communityID communitydomain.CommunityID, authorID userdomain.UserID, title string) postdomain.PostID {
	t.Helper()

	id := postdomain.NewGeneratedPostID()
	_, err := pool.Exec(ctx, `
		INSERT INTO posts (
			id,
			community_id,
			author_id,
			title,
			body,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, 'visible', $6, $6)
	`, id.String(), communityID.String(), authorID.String(), title, "Body for "+title, testNow())
	if err != nil {
		t.Fatalf("insert test post: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM posts WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup test post %q: %v", id.String(), err)
		}
	})

	return id
}

func insertTestCommentVote(ctx context.Context, t *testing.T, pool *pgxpool.Pool, commentID commentdomain.CommentID, userID userdomain.UserID, value int) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO comment_votes (
			comment_id,
			user_id,
			value,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $4)
	`, commentID.String(), userID.String(), value, testNow())
	if err != nil {
		t.Fatalf("insert test comment vote: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `
			DELETE FROM comment_votes
			WHERE comment_id = $1::uuid
				AND user_id = $2::uuid
		`, commentID.String(), userID.String()); err != nil {
			t.Fatalf("cleanup test comment vote comment=%q user=%q: %v", commentID.String(), userID.String(), err)
		}
	})
}

func cleanupComment(ctx context.Context, t *testing.T, pool *pgxpool.Pool, id commentdomain.CommentID) {
	t.Helper()

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM comments WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup comment %q: %v", id.String(), err)
		}
	})
}

func mustComment(t *testing.T, postID postdomain.PostID, authorID userdomain.UserID, parentID *commentdomain.CommentID, body string, now time.Time) *commentdomain.Comment {
	t.Helper()

	commentBody, err := commentdomain.NewCommentBody(body)
	if err != nil {
		t.Fatalf("NewCommentBody returned error: %v", err)
	}
	comment, err := commentdomain.NewComment(commentdomain.NewGeneratedCommentID(), postID, authorID, parentID, commentBody, now)
	if err != nil {
		t.Fatalf("NewComment returned error: %v", err)
	}
	return comment
}

func mustCommentWithStatus(t *testing.T, postID postdomain.PostID, authorID userdomain.UserID, parentID *commentdomain.CommentID, body string, status commentdomain.CommentStatus, now time.Time) *commentdomain.Comment {
	t.Helper()

	commentBody, err := commentdomain.NewCommentBody(body)
	if err != nil {
		t.Fatalf("NewCommentBody returned error: %v", err)
	}
	comment, err := commentdomain.RehydrateComment(commentdomain.NewGeneratedCommentID(), postID, authorID, parentID, commentBody, status, now, now)
	if err != nil {
		t.Fatalf("RehydrateComment returned error: %v", err)
	}
	return comment
}

func commentIDs(comments []commentdomain.Comment) []commentdomain.CommentID {
	ids := make([]commentdomain.CommentID, 0, len(comments))
	for _, comment := range comments {
		ids = append(ids, comment.ID())
	}
	return ids
}

func assertCommentRepositoryContentRefs(t *testing.T, got []postusecase.ContentRef, want []postusecase.ContentRef) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d content refs, got %d: %#v", len(want), len(got), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected content ref at %d: got %#v want %#v", index, got[index], want[index])
		}
	}
}

func mustCommentBody(t *testing.T, raw string) commentdomain.CommentBody {
	t.Helper()

	body, err := commentdomain.NewCommentBody(raw)
	if err != nil {
		t.Fatalf("NewCommentBody returned error: %v", err)
	}
	return body
}

func testNow() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

func randomSuffix() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
}

func hasAppCode(err error, code apperr.Code) bool {
	if err == nil {
		return false
	}

	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code() == code
}
