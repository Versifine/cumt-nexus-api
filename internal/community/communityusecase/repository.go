package communityusecase

import (
	"context"

	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type CommunityRepository interface {
	Create(ctx context.Context, community communitydomain.Community) error
	FindBySlug(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error)
	ListActivePublic(ctx context.Context) ([]communitydomain.Community, error)
}

type CommunityMembershipRepository interface {
	Create(ctx context.Context, membership communitydomain.CommunityMembership) error
}

type CommunityApplicationRepository interface {
	Create(ctx context.Context, application communitydomain.CommunityApplication) error
	FindByID(ctx context.Context, id communitydomain.CommunityApplicationID) (*communitydomain.CommunityApplication, error)
	Save(ctx context.Context, application communitydomain.CommunityApplication) error
}

type PlatformStaffRepository interface {
	IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error)
}
