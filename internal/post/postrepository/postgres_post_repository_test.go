package postrepository

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresPostRepositoryCreateFindListAndNotFound(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "post-"+randomSuffix())
	otherCommunityID := insertTestCommunity(ctx, t, pool, authorID, "post-"+randomSuffix())

	olderPost := mustPost(t, communityID, authorID, "Older", now)
	newerPost := mustPost(t, communityID, authorID, "Newer", now.Add(time.Minute))
	otherPost := mustPost(t, otherCommunityID, authorID, "Other", now.Add(2*time.Minute))

	if err := repo.Create(ctx, *olderPost); err != nil {
		t.Fatalf("Create older post returned error: %v", err)
	}
	cleanupPost(ctx, t, pool, olderPost.ID())
	if err := repo.Create(ctx, *newerPost); err != nil {
		t.Fatalf("Create newer post returned error: %v", err)
	}
	cleanupPost(ctx, t, pool, newerPost.ID())
	if err := repo.Create(ctx, *otherPost); err != nil {
		t.Fatalf("Create other post returned error: %v", err)
	}
	cleanupPost(ctx, t, pool, otherPost.ID())

	got, err := repo.FindVisibleByID(ctx, olderPost.ID())
	if err != nil {
		t.Fatalf("FindVisibleByID returned error: %v", err)
	}
	if got.ID() != olderPost.ID() || got.Title() != olderPost.Title() {
		t.Fatalf("unexpected post: got id=%q title=%q", got.ID().String(), got.Title().String())
	}

	posts, err := repo.ListVisibleByCommunity(ctx, communityID, postusecase.PostListSortNew, nil, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleByCommunity returned error: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected two posts, got %d", len(posts))
	}
	if posts[0].ID() != newerPost.ID() || posts[1].ID() != olderPost.ID() {
		t.Fatalf("expected newest-first ordering, got %#v", []postdomain.PostID{posts[0].ID(), posts[1].ID()})
	}

	if _, err := repo.FindVisibleByID(ctx, postdomain.NewGeneratedPostID()); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing post, got %v", err)
	}
}

func TestPostgresPostRepositoryListPostsByCommunityForManagement(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "manage-post-"+randomSuffix())
	otherCommunityID := insertTestCommunity(ctx, t, pool, authorID, "manage-post-"+randomSuffix())

	visiblePost := mustPost(t, communityID, authorID, "Visible manage post", now)
	removedPost := mustPostWithStatus(t, communityID, authorID, "Removed manage post", postdomain.PostStatusRemoved, now.Add(time.Minute))
	otherPost := mustPostWithStatus(t, otherCommunityID, authorID, "Other removed manage post", postdomain.PostStatusRemoved, now.Add(2*time.Minute))
	for _, post := range []*postdomain.Post{visiblePost, removedPost, otherPost} {
		if err := repo.Create(ctx, *post); err != nil {
			t.Fatalf("Create post %q returned error: %v", post.Title().String(), err)
		}
		cleanupPost(ctx, t, pool, post.ID())
	}

	status := postdomain.PostStatusRemoved
	posts, err := repo.ListPostsByCommunityForManagement(ctx, communityID, &status, 20, 0)
	if err != nil {
		t.Fatalf("ListPostsByCommunityForManagement returned error: %v", err)
	}
	if len(posts) != 1 || posts[0].ID() != removedPost.ID() {
		t.Fatalf("expected only removed post, got %#v", posts)
	}

	allPosts, err := repo.ListPostsByCommunityForManagement(ctx, communityID, nil, 20, 0)
	if err != nil {
		t.Fatalf("ListPostsByCommunityForManagement all returned error: %v", err)
	}
	if len(allPosts) != 2 || allPosts[0].ID() != removedPost.ID() || allPosts[1].ID() != visiblePost.ID() {
		t.Fatalf("expected newest-first posts in community, got %#v", allPosts)
	}
}

func TestPostgresPostRepositoryLoadMetadataByPostIDsReturnsViewerCommunityContext(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	updateTestUserProfile(ctx, t, pool, authorID, "Alice", "https://example.com/avatar.jpg", "Backend builder")
	viewerID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "metadata-"+randomSuffix())
	insertTestCommunityFollow(ctx, t, pool, communityID, viewerID)
	insertTestCommunityMembership(ctx, t, pool, communityID, viewerID, communitydomain.MembershipRoleModerator)
	post := mustPost(t, communityID, authorID, "Metadata viewer context", now)
	if err := repo.Create(ctx, *post); err != nil {
		t.Fatalf("Create post returned error: %v", err)
	}
	cleanupPost(ctx, t, pool, post.ID())

	metadata, err := repo.LoadMetadataByPostIDs(ctx, []postdomain.PostID{post.ID()}, viewerID)
	if err != nil {
		t.Fatalf("LoadMetadataByPostIDs returned error: %v", err)
	}
	got := metadata[post.ID()]
	if !got.Community.ViewerIsFollowing {
		t.Fatal("expected viewer_is_following=true")
	}
	if got.Community.ViewerRole != communitydomain.MembershipRoleModerator.String() {
		t.Fatalf("expected moderator role, got %q", got.Community.ViewerRole)
	}
	if !got.Community.ViewerPermissions.CanPost || got.Community.ViewerPermissions.CanManage || !got.Community.ViewerPermissions.CanModerate {
		t.Fatalf("unexpected community viewer permissions: %#v", got.Community.ViewerPermissions)
	}
	if got.Author.DisplayName != "Alice" || got.Author.AvatarURL != "https://example.com/avatar.jpg" || got.Author.Headline != "Backend builder" {
		t.Fatalf("expected author profile fields, got %#v", got.Author)
	}
}

func TestPostgresPostRepositoryListVisibleInPublicCommunities(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "latest-"+randomSuffix())
	otherCommunityID := insertTestCommunity(ctx, t, pool, authorID, "latest-"+randomSuffix())
	suspendedCommunityID := insertTestCommunityWithStatus(ctx, t, pool, authorID, "latest-"+randomSuffix(), "suspended")

	olderPost := mustPost(t, communityID, authorID, "Older latest", now)
	newerPost := mustPost(t, otherCommunityID, authorID, "Newer latest", now.Add(time.Minute))
	suspendedPost := mustPost(t, suspendedCommunityID, authorID, "Suspended latest", now.Add(2*time.Minute))
	removedPost := mustPostWithStatus(t, communityID, authorID, "Removed latest", postdomain.PostStatusRemoved, now.Add(3*time.Minute))

	for _, post := range []*postdomain.Post{olderPost, newerPost, suspendedPost, removedPost} {
		if err := repo.Create(ctx, *post); err != nil {
			t.Fatalf("Create post %q returned error: %v", post.Title().String(), err)
		}
		cleanupPost(ctx, t, pool, post.ID())
	}

	posts, err := repo.ListVisibleInPublicCommunities(ctx, postusecase.PostListSortNew, nil, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleInPublicCommunities returned error: %v", err)
	}

	var gotIDs []postdomain.PostID
	for _, post := range posts {
		if post.ID() == newerPost.ID() || post.ID() == olderPost.ID() || post.ID() == suspendedPost.ID() || post.ID() == removedPost.ID() {
			gotIDs = append(gotIDs, post.ID())
		}
	}

	if len(gotIDs) != 2 {
		t.Fatalf("expected two visible public posts from this test, got %#v", gotIDs)
	}
	if gotIDs[0] != newerPost.ID() || gotIDs[1] != olderPost.ID() {
		t.Fatalf("expected newest-first visible public posts, got %#v", gotIDs)
	}
}

func TestPostgresPostRepositoryListFollowingInPublicCommunities(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	viewerID := insertTestUser(ctx, t, pool)
	communityAuthorID := insertTestUser(ctx, t, pool)
	followedAuthorID := insertTestUser(ctx, t, pool)
	bothFollowedAuthorID := insertTestUser(ctx, t, pool)
	disabledFollowedAuthorID := insertTestUser(ctx, t, pool)
	unfollowedAuthorID := insertTestUser(ctx, t, pool)

	followedCommunityID := insertTestCommunity(ctx, t, pool, communityAuthorID, "followed-"+randomSuffix())
	authorOnlyCommunityID := insertTestCommunity(ctx, t, pool, followedAuthorID, "author-"+randomSuffix())
	bothFollowedCommunityID := insertTestCommunity(ctx, t, pool, bothFollowedAuthorID, "both-"+randomSuffix())
	disabledAuthorCommunityID := insertTestCommunity(ctx, t, pool, disabledFollowedAuthorID, "disabled-author-"+randomSuffix())
	unfollowedCommunityID := insertTestCommunity(ctx, t, pool, unfollowedAuthorID, "unfollowed-"+randomSuffix())
	suspendedCommunityID := insertTestCommunityWithStatus(ctx, t, pool, followedAuthorID, "suspended-following-"+randomSuffix(), "suspended")

	insertTestCommunityFollow(ctx, t, pool, followedCommunityID, viewerID)
	insertTestCommunityFollow(ctx, t, pool, bothFollowedCommunityID, viewerID)
	insertTestUserFollow(ctx, t, pool, viewerID, followedAuthorID)
	insertTestUserFollow(ctx, t, pool, viewerID, bothFollowedAuthorID)
	insertTestUserFollow(ctx, t, pool, viewerID, disabledFollowedAuthorID)
	updateTestUserStatus(ctx, t, pool, disabledFollowedAuthorID, "disabled")

	followedCommunityPost := mustPost(t, followedCommunityID, communityAuthorID, "Followed community latest", now.Add(time.Minute))
	followedAuthorPost := mustPost(t, authorOnlyCommunityID, followedAuthorID, "Followed author latest", now.Add(2*time.Minute))
	bothFollowedPost := mustPost(t, bothFollowedCommunityID, bothFollowedAuthorID, "Both followed latest", now.Add(3*time.Minute))
	disabledAuthorPost := mustPost(t, disabledAuthorCommunityID, disabledFollowedAuthorID, "Disabled author latest", now.Add(4*time.Minute))
	suspendedCommunityPost := mustPost(t, suspendedCommunityID, followedAuthorID, "Suspended community latest", now.Add(5*time.Minute))
	unfollowedPost := mustPost(t, unfollowedCommunityID, unfollowedAuthorID, "Unfollowed latest", now.Add(6*time.Minute))
	for _, post := range []*postdomain.Post{followedCommunityPost, followedAuthorPost, bothFollowedPost, disabledAuthorPost, suspendedCommunityPost, unfollowedPost} {
		if err := repo.Create(ctx, *post); err != nil {
			t.Fatalf("Create post %q returned error: %v", post.Title().String(), err)
		}
		cleanupPost(ctx, t, pool, post.ID())
	}

	posts, err := repo.ListFollowingInPublicCommunities(ctx, viewerID, postusecase.PostListSortNew, nil, 20, 0)
	if err != nil {
		t.Fatalf("ListFollowingInPublicCommunities returned error: %v", err)
	}

	var gotIDs []postdomain.PostID
	for _, post := range posts {
		switch post.ID() {
		case followedCommunityPost.ID(), followedAuthorPost.ID(), bothFollowedPost.ID(), disabledAuthorPost.ID(), suspendedCommunityPost.ID(), unfollowedPost.ID():
			gotIDs = append(gotIDs, post.ID())
		}
	}
	wantIDs := []postdomain.PostID{bothFollowedPost.ID(), followedAuthorPost.ID(), followedCommunityPost.ID()}
	if !samePostIDs(gotIDs, wantIDs) {
		t.Fatalf("expected following feed ids %#v, got %#v", wantIDs, gotIDs)
	}
}

func TestPostgresPostRepositoryListVisibleInPublicCommunitiesCreatedAfter(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "latest-window-"+randomSuffix())

	oldPost := mustPost(t, communityID, authorID, "Old latest window", now.Add(-2*time.Hour))
	newPost := mustPost(t, communityID, authorID, "New latest window", now.Add(-10*time.Minute))
	for _, post := range []*postdomain.Post{oldPost, newPost} {
		if err := repo.Create(ctx, *post); err != nil {
			t.Fatalf("Create post %q returned error: %v", post.Title().String(), err)
		}
		cleanupPost(ctx, t, pool, post.ID())
	}

	createdAfter := now.Add(-30 * time.Minute)
	posts, err := repo.ListVisibleInPublicCommunities(ctx, postusecase.PostListSortNew, &createdAfter, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleInPublicCommunities returned error: %v", err)
	}

	var gotIDs []postdomain.PostID
	for _, post := range posts {
		switch post.ID() {
		case oldPost.ID(), newPost.ID():
			gotIDs = append(gotIDs, post.ID())
		}
	}
	if len(gotIDs) != 1 || gotIDs[0] != newPost.ID() {
		t.Fatalf("expected only new post in time window, got %#v", gotIDs)
	}
}

func TestPostgresPostRepositoryUpdateContentAndMarkDeleted(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "post-update-"+randomSuffix())
	post := mustPost(t, communityID, authorID, "Original update", now)
	if err := repo.Create(ctx, *post); err != nil {
		t.Fatalf("Create post returned error: %v", err)
	}
	cleanupPost(ctx, t, pool, post.ID())

	if err := post.Edit(mustPostTitle(t, "Updated update"), mustPostBody(t, "Updated body"), now.Add(time.Minute)); err != nil {
		t.Fatalf("Edit post returned error: %v", err)
	}
	if err := repo.UpdateContent(ctx, *post); err != nil {
		t.Fatalf("UpdateContent returned error: %v", err)
	}

	updated, err := repo.FindVisibleByID(ctx, post.ID())
	if err != nil {
		t.Fatalf("FindVisibleByID after update returned error: %v", err)
	}
	if updated.Title().String() != "Updated update" || updated.Body().String() != "Updated body" {
		t.Fatalf("unexpected updated content: title=%q body=%q", updated.Title().String(), updated.Body().String())
	}

	if err := post.MarkDeleted(now.Add(2 * time.Minute)); err != nil {
		t.Fatalf("MarkDeleted returned error: %v", err)
	}
	if err := repo.MarkDeleted(ctx, *post); err != nil {
		t.Fatalf("MarkDeleted repository returned error: %v", err)
	}
	if _, err := repo.FindVisibleByID(ctx, post.ID()); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found after delete, got %v", err)
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM posts WHERE id = $1::uuid`, post.ID().String()).Scan(&status); err != nil {
		t.Fatalf("query deleted post status: %v", err)
	}
	if status != postdomain.PostStatusDeleted.String() {
		t.Fatalf("expected deleted status, got %q", status)
	}
}

func TestPostgresPostRepositoryListVisibleByCommunityHotSort(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "hot-community-"+randomSuffix())

	simplePost := mustPost(t, communityID, authorID, "Simple hot", now.Add(2*time.Minute))
	balancedPost := mustPost(t, communityID, authorID, "Balanced hot", now)
	coldPost := mustPost(t, communityID, authorID, "Cold hot", now.Add(3*time.Minute))

	for _, post := range []*postdomain.Post{simplePost, balancedPost, coldPost} {
		if err := repo.Create(ctx, *post); err != nil {
			t.Fatalf("Create post %q returned error: %v", post.Title().String(), err)
		}
		cleanupPost(ctx, t, pool, post.ID())
	}

	insertTestPostVote(ctx, t, pool, simplePost.ID(), insertTestUser(ctx, t, pool), 1)
	insertTestPostVote(ctx, t, pool, balancedPost.ID(), insertTestUser(ctx, t, pool), 1)
	insertTestPostVote(ctx, t, pool, balancedPost.ID(), insertTestUser(ctx, t, pool), 1)
	insertTestPostVote(ctx, t, pool, balancedPost.ID(), insertTestUser(ctx, t, pool), 1)

	posts, err := repo.ListVisibleByCommunity(ctx, communityID, postusecase.PostListSortHot, nil, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleByCommunity hot returned error: %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("expected three posts, got %d", len(posts))
	}

	gotIDs := []postdomain.PostID{posts[0].ID(), posts[1].ID(), posts[2].ID()}
	wantIDs := []postdomain.PostID{balancedPost.ID(), simplePost.ID(), coldPost.ID()}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("expected hot order %#v, got %#v", wantIDs, gotIDs)
		}
	}
}

func TestPostgresPostRepositoryListVisibleByCommunityRankingSorts(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "ranking-community-"+randomSuffix())

	lowConfidencePost := mustPost(t, communityID, authorID, "Low confidence", now.Add(2*time.Minute))
	bestPost := mustPost(t, communityID, authorID, "Best confidence", now.Add(time.Minute))
	topPost := mustPost(t, communityID, authorID, "Top score", now)

	for _, post := range []*postdomain.Post{lowConfidencePost, bestPost, topPost} {
		if err := repo.Create(ctx, *post); err != nil {
			t.Fatalf("Create post %q returned error: %v", post.Title().String(), err)
		}
		cleanupPost(ctx, t, pool, post.ID())
	}

	insertTestPostVote(ctx, t, pool, lowConfidencePost.ID(), insertTestUser(ctx, t, pool), 1)
	for i := 0; i < 5; i++ {
		insertTestPostVote(ctx, t, pool, bestPost.ID(), insertTestUser(ctx, t, pool), 1)
	}
	insertTestPostVote(ctx, t, pool, bestPost.ID(), insertTestUser(ctx, t, pool), -1)
	for i := 0; i < 8; i++ {
		insertTestPostVote(ctx, t, pool, topPost.ID(), insertTestUser(ctx, t, pool), 1)
	}
	for i := 0; i < 3; i++ {
		insertTestPostVote(ctx, t, pool, topPost.ID(), insertTestUser(ctx, t, pool), -1)
	}

	topPosts, err := repo.ListVisibleByCommunity(ctx, communityID, postusecase.PostListSortTop, nil, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleByCommunity top returned error: %v", err)
	}
	if len(topPosts) != 3 || topPosts[0].ID() != topPost.ID() {
		t.Fatalf("expected top score post first, got %#v", postIDs(topPosts))
	}

	bestPosts, err := repo.ListVisibleByCommunity(ctx, communityID, postusecase.PostListSortBest, nil, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleByCommunity best returned error: %v", err)
	}
	if len(bestPosts) != 3 || bestPosts[0].ID() != bestPost.ID() {
		t.Fatalf("expected best confidence post first, got %#v", postIDs(bestPosts))
	}

	risingPosts, err := repo.ListVisibleByCommunity(ctx, communityID, postusecase.PostListSortRising, nil, 20, 0)
	if err != nil {
		t.Fatalf("ListVisibleByCommunity rising returned error: %v", err)
	}
	if len(risingPosts) != 3 || risingPosts[0].ID() != topPost.ID() {
		t.Fatalf("expected highest recent interaction post first, got %#v", postIDs(risingPosts))
	}
}

func TestPostgresPostRepositoryListVisibleInPublicCommunitiesHotSort(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "global-hot-"+randomSuffix())
	suspendedCommunityID := insertTestCommunityWithStatus(ctx, t, pool, authorID, "global-hot-"+randomSuffix(), "suspended")

	hotPost := mustPost(t, communityID, authorID, "Global hot", now)
	warmPost := mustPost(t, communityID, authorID, "Global warm", now.Add(time.Minute))
	suspendedPost := mustPost(t, suspendedCommunityID, authorID, "Suspended hot", now.Add(2*time.Minute))
	removedPost := mustPostWithStatus(t, communityID, authorID, "Removed hot", postdomain.PostStatusRemoved, now.Add(3*time.Minute))

	for _, post := range []*postdomain.Post{hotPost, warmPost, suspendedPost, removedPost} {
		if err := repo.Create(ctx, *post); err != nil {
			t.Fatalf("Create post %q returned error: %v", post.Title().String(), err)
		}
		cleanupPost(ctx, t, pool, post.ID())
	}

	for i := 0; i < 8; i++ {
		insertTestPostVote(ctx, t, pool, hotPost.ID(), insertTestUser(ctx, t, pool), 1)
	}
	for i := 0; i < 3; i++ {
		insertTestPostVote(ctx, t, pool, warmPost.ID(), insertTestUser(ctx, t, pool), 1)
	}
	for i := 0; i < 10; i++ {
		insertTestPostVote(ctx, t, pool, suspendedPost.ID(), insertTestUser(ctx, t, pool), 1)
		insertTestPostVote(ctx, t, pool, removedPost.ID(), insertTestUser(ctx, t, pool), 1)
	}

	posts, err := repo.ListVisibleInPublicCommunities(ctx, postusecase.PostListSortHot, nil, 200, 0)
	if err != nil {
		t.Fatalf("ListVisibleInPublicCommunities hot returned error: %v", err)
	}

	var gotIDs []postdomain.PostID
	for _, post := range posts {
		switch post.ID() {
		case hotPost.ID(), warmPost.ID(), suspendedPost.ID(), removedPost.ID():
			gotIDs = append(gotIDs, post.ID())
		}
	}

	if len(gotIDs) != 2 {
		t.Fatalf("expected only two visible public test posts, got %#v", gotIDs)
	}
	if gotIDs[0] != hotPost.ID() || gotIDs[1] != warmPost.ID() {
		t.Fatalf("expected public hot order [%s %s], got %#v", hotPost.ID().String(), warmPost.ID().String(), gotIDs)
	}
}

func TestPostgresPostRepositoryListRecommendedInPublicCommunitiesDiversifiesAndBoostsViewer(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	viewerID := insertTestUser(ctx, t, pool)
	followedCommunityID := insertTestCommunity(ctx, t, pool, authorID, "recommended-followed-"+randomSuffix())
	otherCommunityID := insertTestCommunity(ctx, t, pool, authorID, "recommended-other-"+randomSuffix())

	followedOlderPost := mustPost(t, followedCommunityID, authorID, "Recommended followed older", now)
	otherPost := mustPost(t, otherCommunityID, authorID, "Recommended other", now.Add(30*time.Second))
	followedNewerPost := mustPost(t, followedCommunityID, authorID, "Recommended followed newer", now.Add(time.Minute))

	for _, post := range []*postdomain.Post{followedOlderPost, otherPost, followedNewerPost} {
		if err := repo.Create(ctx, *post); err != nil {
			t.Fatalf("Create post %q returned error: %v", post.Title().String(), err)
		}
		cleanupPost(ctx, t, pool, post.ID())
	}
	insertTestCommunityFollow(ctx, t, pool, followedCommunityID, viewerID)

	posts, err := repo.ListRecommendedInPublicCommunities(ctx, viewerID, postusecase.PostListSortNew, nil, 200, 0)
	if err != nil {
		t.Fatalf("ListRecommendedInPublicCommunities returned error: %v", err)
	}

	var gotIDs []postdomain.PostID
	for _, post := range posts {
		switch post.ID() {
		case followedOlderPost.ID(), otherPost.ID(), followedNewerPost.ID():
			gotIDs = append(gotIDs, post.ID())
		}
	}

	wantIDs := []postdomain.PostID{followedNewerPost.ID(), otherPost.ID(), followedOlderPost.ID()}
	if len(gotIDs) != len(wantIDs) {
		t.Fatalf("expected recommended test posts %#v, got %#v", wantIDs, gotIDs)
	}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("expected recommended order %#v, got %#v", wantIDs, gotIDs)
		}
	}
}

func TestPostgresPostRepositoryMapsForeignKeyFailure(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	post := mustPost(t, communitydomain.NewGeneratedCommunityID(), authorID, "Missing community", now)
	if err := repo.Create(ctx, *post); !hasAppCode(err, apperr.CodeNotFound) {
		t.Fatalf("expected not_found for missing related community, got %v", err)
	}
}

func TestPostgresPostRepositoryReplaceAndListContentRefs(t *testing.T) {
	ctx, pool := newTestPool(t)
	repo := NewPostgresPostRepository(pool)
	now := testNow()

	authorID := insertTestUser(ctx, t, pool)
	communityID := insertTestCommunity(ctx, t, pool, authorID, "post-content-refs-"+randomSuffix())
	post := mustPost(t, communityID, authorID, "Post content refs", now)
	if err := repo.Create(ctx, *post); err != nil {
		t.Fatalf("Create post returned error: %v", err)
	}
	cleanupPost(ctx, t, pool, post.ID())

	refs := []postusecase.ContentRef{
		{Kind: postusecase.ContentRefKindLink, RefID: "https://example.com/one"},
		{Kind: postusecase.ContentRefKindEmbed, RefID: "ba9d3ef9-29e4-4bb9-b2da-1c7ba55e2702"},
	}
	if err := repo.ReplacePostContentRefs(ctx, post.ID(), refs, now); err != nil {
		t.Fatalf("ReplacePostContentRefs returned error: %v", err)
	}
	got, err := repo.ListPostContentRefsByPostIDs(ctx, []postdomain.PostID{post.ID(), postdomain.NewGeneratedPostID()})
	if err != nil {
		t.Fatalf("ListPostContentRefsByPostIDs returned error: %v", err)
	}
	assertPostRepositoryContentRefs(t, got[post.ID()], refs)

	replacement := []postusecase.ContentRef{
		{Kind: postusecase.ContentRefKindImage, RefID: "98fb2f1e-72a8-4f3a-9a38-787aeed6ac9a"},
	}
	if err := repo.ReplacePostContentRefs(ctx, post.ID(), replacement, now.Add(time.Minute)); err != nil {
		t.Fatalf("ReplacePostContentRefs replacement returned error: %v", err)
	}
	got, err = repo.ListPostContentRefsByPostIDs(ctx, []postdomain.PostID{post.ID()})
	if err != nil {
		t.Fatalf("ListPostContentRefsByPostIDs after replace returned error: %v", err)
	}
	assertPostRepositoryContentRefs(t, got[post.ID()], replacement)

	if err := repo.ReplacePostContentRefs(ctx, post.ID(), nil, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("ReplacePostContentRefs clear returned error: %v", err)
	}
	got, err = repo.ListPostContentRefsByPostIDs(ctx, []postdomain.PostID{post.ID()})
	if err != nil {
		t.Fatalf("ListPostContentRefsByPostIDs after clear returned error: %v", err)
	}
	if len(got[post.ID()]) != 0 {
		t.Fatalf("expected cleared post content refs, got %#v", got[post.ID()])
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

	requirePostSchema(ctx, t, pool)

	return ctx, pool
}

func requirePostSchema(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, table := range []string{"users", "communities", "posts", "comments", "post_votes", "post_saves", "community_follows", "user_follows", "post_content_refs"} {
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
	username := "post_repo_" + randomSuffix()

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

func updateTestUserStatus(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userID userdomain.UserID, status string) {
	t.Helper()

	if _, err := pool.Exec(ctx, `
		UPDATE users
		SET status = $2, updated_at = $3
		WHERE id = $1::uuid
	`, userID.String(), status, testNow()); err != nil {
		t.Fatalf("update test user status: %v", err)
	}
}

func insertTestCommunity(ctx context.Context, t *testing.T, pool *pgxpool.Pool, createdBy userdomain.UserID, rawSlug string) communitydomain.CommunityID {
	return insertTestCommunityWithStatus(ctx, t, pool, createdBy, rawSlug, "active")
}

func insertTestCommunityWithStatus(ctx context.Context, t *testing.T, pool *pgxpool.Pool, createdBy userdomain.UserID, rawSlug string, status string) communitydomain.CommunityID {
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
		VALUES ($1::uuid, $2, $3, '', 'user_created', $4, 'public', $5::uuid, $6, $6)
	`, id.String(), rawSlug, "Post Repo "+rawSlug, status, createdBy.String(), testNow())
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

func insertTestPostVote(ctx context.Context, t *testing.T, pool *pgxpool.Pool, postID postdomain.PostID, userID userdomain.UserID, value int) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO post_votes (
			post_id,
			user_id,
			value,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $4)
	`, postID.String(), userID.String(), value, testNow())
	if err != nil {
		t.Fatalf("insert test post vote: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `
			DELETE FROM post_votes
			WHERE post_id = $1::uuid
				AND user_id = $2::uuid
		`, postID.String(), userID.String()); err != nil {
			t.Fatalf("cleanup test post vote post=%q user=%q: %v", postID.String(), userID.String(), err)
		}
	})
}

func insertTestCommunityFollow(ctx context.Context, t *testing.T, pool *pgxpool.Pool, communityID communitydomain.CommunityID, userID userdomain.UserID) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO community_follows (
			community_id,
			user_id,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, $3)
	`, communityID.String(), userID.String(), testNow())
	if err != nil {
		t.Fatalf("insert test community follow: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `
			DELETE FROM community_follows
			WHERE community_id = $1::uuid
				AND user_id = $2::uuid
		`, communityID.String(), userID.String()); err != nil {
			t.Fatalf("cleanup test community follow community=%q user=%q: %v", communityID.String(), userID.String(), err)
		}
	})
}

func insertTestUserFollow(ctx context.Context, t *testing.T, pool *pgxpool.Pool, followerID userdomain.UserID, followingID userdomain.UserID) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO user_follows (
			follower_id,
			following_id,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, $3)
	`, followerID.String(), followingID.String(), testNow())
	if err != nil {
		t.Fatalf("insert test user follow: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `
			DELETE FROM user_follows
			WHERE follower_id = $1::uuid
				AND following_id = $2::uuid
		`, followerID.String(), followingID.String()); err != nil {
			t.Fatalf("cleanup test user follow follower=%q following=%q: %v", followerID.String(), followingID.String(), err)
		}
	})
}

func insertTestCommunityMembership(ctx context.Context, t *testing.T, pool *pgxpool.Pool, communityID communitydomain.CommunityID, userID userdomain.UserID, role communitydomain.MembershipRole) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO community_memberships (
			community_id,
			user_id,
			role,
			status,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3, 'active', $4, $4)
	`, communityID.String(), userID.String(), role.String(), testNow())
	if err != nil {
		t.Fatalf("insert test community membership: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `
			DELETE FROM community_memberships
			WHERE community_id = $1::uuid
				AND user_id = $2::uuid
		`, communityID.String(), userID.String()); err != nil {
			t.Fatalf("cleanup test community membership community=%q user=%q: %v", communityID.String(), userID.String(), err)
		}
	})
}

func cleanupPost(ctx context.Context, t *testing.T, pool *pgxpool.Pool, id postdomain.PostID) {
	t.Helper()

	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM posts WHERE id = $1::uuid`, id.String()); err != nil {
			t.Fatalf("cleanup post %q: %v", id.String(), err)
		}
	})
}

func postIDs(posts []postdomain.Post) []postdomain.PostID {
	ids := make([]postdomain.PostID, 0, len(posts))
	for _, post := range posts {
		ids = append(ids, post.ID())
	}
	return ids
}

func samePostIDs(got []postdomain.PostID, want []postdomain.PostID) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func assertPostRepositoryContentRefs(t *testing.T, got []postusecase.ContentRef, want []postusecase.ContentRef) {
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

func mustPost(t *testing.T, communityID communitydomain.CommunityID, authorID userdomain.UserID, title string, now time.Time) *postdomain.Post {
	return mustPostWithStatus(t, communityID, authorID, title, postdomain.PostStatusVisible, now)
}

func mustPostWithStatus(t *testing.T, communityID communitydomain.CommunityID, authorID userdomain.UserID, title string, status postdomain.PostStatus, now time.Time) *postdomain.Post {
	t.Helper()

	postTitle, err := postdomain.NewPostTitle(title)
	if err != nil {
		t.Fatalf("NewPostTitle returned error: %v", err)
	}
	body, err := postdomain.NewPostBody("Body for " + title)
	if err != nil {
		t.Fatalf("NewPostBody returned error: %v", err)
	}
	post, err := postdomain.RehydratePost(postdomain.NewGeneratedPostID(), communityID, authorID, postTitle, body, status, now, now)
	if err != nil {
		t.Fatalf("NewPost returned error: %v", err)
	}
	return post
}

func mustPostTitle(t *testing.T, raw string) postdomain.PostTitle {
	t.Helper()

	title, err := postdomain.NewPostTitle(raw)
	if err != nil {
		t.Fatalf("NewPostTitle returned error: %v", err)
	}
	return title
}

func mustPostBody(t *testing.T, raw string) postdomain.PostBody {
	t.Helper()

	body, err := postdomain.NewPostBody(raw)
	if err != nil {
		t.Fatalf("NewPostBody returned error: %v", err)
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
