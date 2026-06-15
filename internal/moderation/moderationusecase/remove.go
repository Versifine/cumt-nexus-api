package moderationusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type RemoveUseCase struct {
	removals    ContentRemovalRepository
	staff       PlatformStaffRepository
	communities CommunityRepository
	roles       CommunityRoleRepository
	posts       PostRepository
	comments    CommentRepository
	now         func() time.Time
}

func NewRemoveUseCase(
	removals ContentRemovalRepository,
	staff PlatformStaffRepository,
	communities CommunityRepository,
	roles CommunityRoleRepository,
	posts PostRepository,
	comments CommentRepository,
	now func() time.Time,
) *RemoveUseCase {
	if now == nil {
		now = time.Now
	}

	return &RemoveUseCase{
		removals:    removals,
		staff:       staff,
		communities: communities,
		roles:       roles,
		posts:       posts,
		comments:    comments,
		now:         now,
	}
}

type RemovePostInput struct {
	PostID  string
	ActorID userdomain.UserID
	Reason  string
}

type RemoveCommentInput struct {
	CommentID string
	ActorID   userdomain.UserID
	Reason    string
}

type RemoveCommunityPostInput struct {
	CommunitySlug string
	PostID        string
	ActorID       userdomain.UserID
	Reason        string
}

type RemoveCommunityCommentInput struct {
	CommunitySlug string
	CommentID     string
	ActorID       userdomain.UserID
	Reason        string
}

type RemoveContentResult struct {
	Action ModerationAction
}

type ModerationAction struct {
	ID         string
	TargetType string
	PostID     string
	CommentID  string
	ActorID    string
	Action     string
	Reason     string
	CreatedAt  time.Time
}

func (uc *RemoveUseCase) RemovePost(ctx context.Context, input RemovePostInput) (RemoveContentResult, error) {
	if err := uc.ensureActorCanModerate(ctx, input.ActorID); err != nil {
		return RemoveContentResult{}, err
	}
	reason, err := moderationdomain.NewReason(input.Reason)
	if err != nil {
		return RemoveContentResult{}, err
	}
	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return RemoveContentResult{}, err
	}
	target, err := moderationdomain.NewPostTarget(postID)
	if err != nil {
		return RemoveContentResult{}, err
	}
	action, err := uc.newRemoveAction(target, input.ActorID, reason)
	if err != nil {
		return RemoveContentResult{}, err
	}

	if err := uc.removals.RemovePostWithAction(ctx, *action); err != nil {
		return RemoveContentResult{}, fmt.Errorf("remove post with moderation action: %w", err)
	}

	return RemoveContentResult{
		Action: toModerationActionDTO(*action),
	}, nil
}

func (uc *RemoveUseCase) RemoveComment(ctx context.Context, input RemoveCommentInput) (RemoveContentResult, error) {
	if err := uc.ensureActorCanModerate(ctx, input.ActorID); err != nil {
		return RemoveContentResult{}, err
	}
	reason, err := moderationdomain.NewReason(input.Reason)
	if err != nil {
		return RemoveContentResult{}, err
	}
	commentID, err := commentdomain.NewCommentID(input.CommentID)
	if err != nil {
		return RemoveContentResult{}, err
	}
	target, err := moderationdomain.NewCommentTarget(commentID)
	if err != nil {
		return RemoveContentResult{}, err
	}
	action, err := uc.newRemoveAction(target, input.ActorID, reason)
	if err != nil {
		return RemoveContentResult{}, err
	}

	if err := uc.removals.RemoveCommentWithAction(ctx, *action); err != nil {
		return RemoveContentResult{}, fmt.Errorf("remove comment with moderation action: %w", err)
	}

	return RemoveContentResult{
		Action: toModerationActionDTO(*action),
	}, nil
}

func (uc *RemoveUseCase) RemoveCommunityPost(ctx context.Context, input RemoveCommunityPostInput) (RemoveContentResult, error) {
	reason, err := moderationdomain.NewReason(input.Reason)
	if err != nil {
		return RemoveContentResult{}, err
	}
	postID, err := postdomain.NewPostID(input.PostID)
	if err != nil {
		return RemoveContentResult{}, err
	}
	community, err := uc.loadModeratedCommunity(ctx, input.CommunitySlug, input.ActorID)
	if err != nil {
		return RemoveContentResult{}, err
	}
	post, err := uc.posts.FindVisibleByID(ctx, postID)
	if err != nil {
		return RemoveContentResult{}, fmt.Errorf("find community moderation post: %w", err)
	}
	if post.CommunityID() != community.ID() {
		return RemoveContentResult{}, apperr.New(apperr.CodeNotFound, "post not found")
	}
	target, err := moderationdomain.NewPostTarget(post.ID())
	if err != nil {
		return RemoveContentResult{}, err
	}
	action, err := uc.newRemoveAction(target, input.ActorID, reason)
	if err != nil {
		return RemoveContentResult{}, err
	}
	if err := uc.removals.RemovePostWithAction(ctx, *action); err != nil {
		return RemoveContentResult{}, fmt.Errorf("remove community post with moderation action: %w", err)
	}
	return RemoveContentResult{Action: toModerationActionDTO(*action)}, nil
}

func (uc *RemoveUseCase) RemoveCommunityComment(ctx context.Context, input RemoveCommunityCommentInput) (RemoveContentResult, error) {
	reason, err := moderationdomain.NewReason(input.Reason)
	if err != nil {
		return RemoveContentResult{}, err
	}
	commentID, err := commentdomain.NewCommentID(input.CommentID)
	if err != nil {
		return RemoveContentResult{}, err
	}
	community, err := uc.loadModeratedCommunity(ctx, input.CommunitySlug, input.ActorID)
	if err != nil {
		return RemoveContentResult{}, err
	}
	comment, err := uc.comments.FindVisibleByID(ctx, commentID)
	if err != nil {
		return RemoveContentResult{}, fmt.Errorf("find community moderation comment: %w", err)
	}
	post, err := uc.posts.FindVisibleByID(ctx, comment.PostID())
	if err != nil {
		return RemoveContentResult{}, fmt.Errorf("find community moderation comment post: %w", err)
	}
	if post.CommunityID() != community.ID() {
		return RemoveContentResult{}, apperr.New(apperr.CodeNotFound, "comment not found")
	}
	target, err := moderationdomain.NewCommentTarget(comment.ID())
	if err != nil {
		return RemoveContentResult{}, err
	}
	action, err := uc.newRemoveAction(target, input.ActorID, reason)
	if err != nil {
		return RemoveContentResult{}, err
	}
	if err := uc.removals.RemoveCommentWithAction(ctx, *action); err != nil {
		return RemoveContentResult{}, fmt.Errorf("remove community comment with moderation action: %w", err)
	}
	return RemoveContentResult{Action: toModerationActionDTO(*action)}, nil
}

func (uc *RemoveUseCase) ensureActorCanModerate(ctx context.Context, actorID userdomain.UserID) error {
	if strings.TrimSpace(actorID.String()) == "" {
		return apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}

	isStaff, err := uc.staff.IsPlatformStaff(ctx, actorID)
	if err != nil {
		return fmt.Errorf("check platform staff: %w", err)
	}
	if !isStaff {
		return apperr.New(apperr.CodeForbidden, "platform staff required")
	}
	return nil
}

func (uc *RemoveUseCase) loadModeratedCommunity(ctx context.Context, rawSlug string, actorID userdomain.UserID) (*communitydomain.Community, error) {
	if strings.TrimSpace(actorID.String()) == "" {
		return nil, apperr.New(apperr.CodeUnauthenticated, "authentication required")
	}
	if uc.communities == nil || uc.roles == nil || uc.posts == nil || uc.comments == nil {
		return nil, apperr.New(apperr.CodeInternal, "community moderation is not configured")
	}
	slug, err := communitydomain.NewCommunitySlug(rawSlug)
	if err != nil {
		return nil, err
	}
	community, err := uc.communities.FindBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("find community for moderation: %w", err)
	}
	if community.Status() != communitydomain.CommunityStatusActive {
		return nil, apperr.New(apperr.CodeNotFound, "community not found")
	}
	roles, err := uc.roles.FindActiveRolesByUser(ctx, []communitydomain.CommunityID{community.ID()}, actorID)
	if err != nil {
		return nil, fmt.Errorf("find community moderation role: %w", err)
	}
	role, ok := roles[community.ID()]
	if !ok || (role != communitydomain.MembershipRoleOwner && role != communitydomain.MembershipRoleModerator) {
		return nil, apperr.New(apperr.CodeForbidden, "community moderator required")
	}
	return community, nil
}

func (uc *RemoveUseCase) newRemoveAction(target moderationdomain.Target, actorID userdomain.UserID, reason moderationdomain.Reason) (*moderationdomain.ModerationAction, error) {
	return moderationdomain.NewModerationAction(
		moderationdomain.NewGeneratedModerationActionID(),
		target,
		actorID,
		moderationdomain.ActionTypeRemove,
		reason,
		uc.now().UTC(),
	)
}

func toModerationActionDTO(action moderationdomain.ModerationAction) ModerationAction {
	target := action.Target()
	postID := ""
	if id, ok := target.PostID(); ok {
		postID = id.String()
	}
	commentID := ""
	if id, ok := target.CommentID(); ok {
		commentID = id.String()
	}

	return ModerationAction{
		ID:         action.ID().String(),
		TargetType: target.Type().String(),
		PostID:     postID,
		CommentID:  commentID,
		ActorID:    action.ActorID().String(),
		Action:     action.Action().String(),
		Reason:     action.Reason().String(),
		CreatedAt:  action.CreatedAt(),
	}
}
