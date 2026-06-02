package searchusecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func TestSearchDefaultsScopeAndPagination(t *testing.T) {
	repository := &fakeRepository{
		communities: []CommunityResult{{ID: "community-1", Name: "Campus Life"}},
		posts:       []PostResult{{ID: "post-1", Title: "Campus notice"}},
	}
	uc := NewUseCase(repository)

	result, err := uc.Search(context.Background(), SearchInput{
		ActorID: userdomain.NewGeneratedUserID(),
		Query:   " campus ",
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if result.Query != "campus" || result.Scope != ScopeAll.String() {
		t.Fatalf("unexpected normalized query/scope: %#v", result)
	}
	if result.Limit != DefaultSearchLimit || result.Offset != 0 {
		t.Fatalf("unexpected pagination: %#v", result)
	}
	if !repository.communitiesCalled || !repository.postsCalled {
		t.Fatal("expected both repositories to be called")
	}
}

func TestSearchSupportsCommunityScope(t *testing.T) {
	repository := &fakeRepository{}
	uc := NewUseCase(repository)

	result, err := uc.Search(context.Background(), SearchInput{
		ActorID: userdomain.NewGeneratedUserID(),
		Query:   "campus",
		Scope:   "communities",
		Limit:   99,
		Offset:  3,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Scope != ScopeCommunities.String() || result.Limit != MaxSearchLimit || result.Offset != 3 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !repository.communitiesCalled || repository.postsCalled {
		t.Fatalf("unexpected repository calls: communities=%v posts=%v", repository.communitiesCalled, repository.postsCalled)
	}
}

func TestSearchSupportsPostScope(t *testing.T) {
	repository := &fakeRepository{}
	uc := NewUseCase(repository)

	result, err := uc.Search(context.Background(), SearchInput{
		ActorID: userdomain.NewGeneratedUserID(),
		Query:   "notice",
		Scope:   "POSTS",
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.Scope != ScopePosts.String() {
		t.Fatalf("unexpected scope: %#v", result)
	}
	if repository.communitiesCalled || !repository.postsCalled {
		t.Fatalf("unexpected repository calls: communities=%v posts=%v", repository.communitiesCalled, repository.postsCalled)
	}
}

func TestSearchRejectsInvalidInput(t *testing.T) {
	uc := NewUseCase(&fakeRepository{})

	tests := []struct {
		name  string
		input SearchInput
	}{
		{
			name:  "missing actor",
			input: SearchInput{Query: "campus"},
		},
		{
			name:  "blank query",
			input: SearchInput{ActorID: userdomain.NewGeneratedUserID(), Query: " "},
		},
		{
			name:  "long query",
			input: SearchInput{ActorID: userdomain.NewGeneratedUserID(), Query: strings.Repeat("a", MaxSearchQueryRunes+1)},
		},
		{
			name:  "invalid scope",
			input: SearchInput{ActorID: userdomain.NewGeneratedUserID(), Query: "campus", Scope: "comments"},
		},
		{
			name:  "negative limit",
			input: SearchInput{ActorID: userdomain.NewGeneratedUserID(), Query: "campus", Limit: -1},
		},
		{
			name:  "negative offset",
			input: SearchInput{ActorID: userdomain.NewGeneratedUserID(), Query: "campus", Offset: -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Search(context.Background(), tt.input)
			if !hasAppError(err) {
				t.Fatalf("expected app error, got %v", err)
			}
		})
	}
}

func TestSearchPropagatesRepositoryError(t *testing.T) {
	repository := &fakeRepository{
		communitiesErr: errors.New("database unavailable"),
	}
	uc := NewUseCase(repository)

	_, err := uc.Search(context.Background(), SearchInput{
		ActorID: userdomain.NewGeneratedUserID(),
		Query:   "campus",
		Scope:   "communities",
	})
	if err == nil {
		t.Fatal("expected repository error")
	}
}

type fakeRepository struct {
	communitiesCalled bool
	postsCalled       bool
	communities       []CommunityResult
	posts             []PostResult
	communitiesErr    error
	postsErr          error
}

func (f *fakeRepository) SearchCommunities(ctx context.Context, query string, limit int, offset int) ([]CommunityResult, error) {
	f.communitiesCalled = true
	return f.communities, f.communitiesErr
}

func (f *fakeRepository) SearchPosts(ctx context.Context, query string, limit int, offset int) ([]PostResult, error) {
	f.postsCalled = true
	return f.posts, f.postsErr
}

func hasAppError(err error) bool {
	var appError *apperr.Error
	return errors.As(err, &appError)
}
