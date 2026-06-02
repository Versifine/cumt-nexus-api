package searchusecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

const (
	DefaultSearchLimit  = 20
	MaxSearchLimit      = 50
	MaxSearchQueryRunes = 100
)

type Scope string

const (
	ScopeAll         Scope = "all"
	ScopeCommunities Scope = "communities"
	ScopePosts       Scope = "posts"
)

type UseCase struct {
	repository Repository
}

type SearchInput struct {
	ActorID userdomain.UserID
	Query   string
	Scope   string
	Limit   int
	Offset  int
}

type SearchResult struct {
	Query       string
	Scope       string
	Limit       int
	Offset      int
	Communities []CommunityResult
	Posts       []PostResult
}

func NewUseCase(repository Repository) *UseCase {
	return &UseCase{
		repository: repository,
	}
}

func (uc *UseCase) Search(ctx context.Context, input SearchInput) (SearchResult, error) {
	if strings.TrimSpace(input.ActorID.String()) == "" {
		return SearchResult{}, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	query, err := normalizeQuery(input.Query)
	if err != nil {
		return SearchResult{}, err
	}
	scope, err := normalizeScope(input.Scope)
	if err != nil {
		return SearchResult{}, err
	}
	limit, offset, err := normalizePagination(input.Limit, input.Offset)
	if err != nil {
		return SearchResult{}, err
	}

	result := SearchResult{
		Query:  query,
		Scope:  scope.String(),
		Limit:  limit,
		Offset: offset,
	}

	if scope == ScopeAll || scope == ScopeCommunities {
		communities, err := uc.repository.SearchCommunities(ctx, query, limit, offset)
		if err != nil {
			return SearchResult{}, fmt.Errorf("search communities: %w", err)
		}
		result.Communities = communities
	}
	if scope == ScopeAll || scope == ScopePosts {
		posts, err := uc.repository.SearchPosts(ctx, query, limit, offset)
		if err != nil {
			return SearchResult{}, fmt.Errorf("search posts: %w", err)
		}
		result.Posts = posts
	}

	return result, nil
}

func normalizeQuery(raw string) (string, error) {
	query := strings.TrimSpace(raw)
	if query == "" {
		return "", apperr.New(apperr.CodeInvalidArgument, "search query is required")
	}
	if len([]rune(query)) > MaxSearchQueryRunes {
		return "", apperr.New(apperr.CodeInvalidArgument, "search query is too long")
	}
	return query, nil
}

func normalizeScope(raw string) (Scope, error) {
	if strings.TrimSpace(raw) == "" {
		return ScopeAll, nil
	}
	switch Scope(strings.TrimSpace(strings.ToLower(raw))) {
	case ScopeAll:
		return ScopeAll, nil
	case ScopeCommunities:
		return ScopeCommunities, nil
	case ScopePosts:
		return ScopePosts, nil
	default:
		return "", apperr.New(apperr.CodeInvalidArgument, "search scope is invalid")
	}
}

func (scope Scope) String() string {
	return string(scope)
}

func normalizePagination(limit int, offset int) (int, int, error) {
	if limit < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "limit must be non-negative")
	}
	if offset < 0 {
		return 0, 0, apperr.New(apperr.CodeInvalidArgument, "offset must be non-negative")
	}
	if limit == 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}
	return limit, offset, nil
}
