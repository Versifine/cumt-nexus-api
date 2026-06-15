package communityusecase

import (
	"context"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/comment/commentdomain"
	"github.com/Versifine/cumt-nexus-api/internal/community/communitydomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationdomain"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/post/postdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type CommunityRepository interface {
	Create(ctx context.Context, community communitydomain.Community) error
	FindByID(ctx context.Context, id communitydomain.CommunityID) (*communitydomain.Community, error)
	FindBySlug(ctx context.Context, slug communitydomain.CommunitySlug) (*communitydomain.Community, error)
	ListActivePublic(ctx context.Context, limit int, offset int) ([]communitydomain.Community, error)
}

type CommunitySettingsRepository interface {
	UpdateDetails(ctx context.Context, community communitydomain.Community) error
}

type CommunityStatsRepository interface {
	LoadPublicStatsByCommunityIDs(ctx context.Context, communityIDs []communitydomain.CommunityID) (map[communitydomain.CommunityID]CommunityStats, error)
}

type CommunityFollowRepository interface {
	FollowCommunity(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, now time.Time) error
	DeleteCommunityFollow(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID) error
	ListFollowedActivePublic(ctx context.Context, userID userdomain.UserID, limit int, offset int) ([]communitydomain.Community, error)
	FindFollowedCommunityIDsByUser(ctx context.Context, communityIDs []communitydomain.CommunityID, userID userdomain.UserID) (map[communitydomain.CommunityID]bool, error)
}

type CommunityMembershipRepository interface {
	CommunityMembershipReadRepository
	Create(ctx context.Context, membership communitydomain.CommunityMembership) error
	FindActiveMemberByUserID(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID) (CommunityMember, error)
	FindActiveMemberByUsername(ctx context.Context, communityID communitydomain.CommunityID, username string) (CommunityMember, error)
	FindActiveUserByUsername(ctx context.Context, username string) (CommunityMember, error)
	FindActiveUserByID(ctx context.Context, userID userdomain.UserID) (CommunityMember, error)
	CountActiveMembers(ctx context.Context, communityID communitydomain.CommunityID) (int, error)
	CountActiveModerators(ctx context.Context, communityID communitydomain.CommunityID) (int, error)
	UpsertActiveMemberRole(ctx context.Context, communityID communitydomain.CommunityID, userID userdomain.UserID, role communitydomain.MembershipRole, now time.Time) (CommunityMember, error)
	CreateOwnerTransfer(ctx context.Context, transfer CommunityOwnerTransferRecord) error
	FindCurrentOwnerTransfer(ctx context.Context, communityID communitydomain.CommunityID, now time.Time) (CommunityOwnerTransferRecord, error)
	ListOwnerTransfersByTarget(ctx context.Context, targetUserID userdomain.UserID, status string, now time.Time, limit int, offset int) ([]CommunityOwnerTransferListRecord, error)
	FindOwnerTransferByID(ctx context.Context, transferID string) (CommunityOwnerTransferRecord, error)
	FindOwnerTransferForUpdate(ctx context.Context, transferID string) (CommunityOwnerTransferRecord, error)
	AcceptOwnerTransfer(ctx context.Context, transferID string, acceptedAt time.Time) error
	CancelOwnerTransfer(ctx context.Context, transferID string, cancelledAt time.Time) error
	TransferOwner(ctx context.Context, communityID communitydomain.CommunityID, newOwnerID userdomain.UserID, now time.Time) (CommunityOwnerChange, error)
}

type CommunityMembershipReadRepository interface {
	FindActiveRolesByUser(ctx context.Context, communityIDs []communitydomain.CommunityID, userID userdomain.UserID) (map[communitydomain.CommunityID]communitydomain.MembershipRole, error)
	ListActiveMembers(ctx context.Context, communityID communitydomain.CommunityID, limit int, offset int) ([]CommunityMember, error)
}

type CommunityManagePostRepository interface {
	ListPostsByCommunityForManagement(ctx context.Context, communityID communitydomain.CommunityID, status *postdomain.PostStatus, limit int, offset int) ([]postdomain.Post, error)
}

type CommunityManageCommentRepository interface {
	ListCommentsByCommunityForManagement(ctx context.Context, communityID communitydomain.CommunityID, status *commentdomain.CommentStatus, limit int, offset int) ([]commentdomain.Comment, error)
}

type CommunityManageReportRepository interface {
	ListReportsByCommunityForManagement(ctx context.Context, communityID communitydomain.CommunityID, status moderationdomain.ReportStatus, limit int, offset int) ([]moderationusecase.ContentReportRecord, error)
}

type CommunityRuleRepository interface {
	ListRules(ctx context.Context, communityID communitydomain.CommunityID) ([]communitydomain.CommunityRule, error)
	FindRuleByID(ctx context.Context, id communitydomain.CommunityRuleID) (*communitydomain.CommunityRule, error)
	CreateRule(ctx context.Context, rule communitydomain.CommunityRule) error
	UpdateRule(ctx context.Context, rule communitydomain.CommunityRule) error
	DeleteRule(ctx context.Context, id communitydomain.CommunityRuleID, communityID communitydomain.CommunityID) error
}

type CommunityApplicationRepository interface {
	Create(ctx context.Context, application communitydomain.CommunityApplication) error
	FindByID(ctx context.Context, id communitydomain.CommunityApplicationID) (*communitydomain.CommunityApplication, error)
	FindByIDForUpdate(ctx context.Context, id communitydomain.CommunityApplicationID) (*communitydomain.CommunityApplication, error)
	ListByStatus(ctx context.Context, status communitydomain.ApplicationStatus, limit int, offset int) ([]communitydomain.CommunityApplication, error)
	Save(ctx context.Context, application communitydomain.CommunityApplication) error
}

type PlatformStaffRepository interface {
	IsPlatformStaff(ctx context.Context, userID userdomain.UserID) (bool, error)
}

type PlatformOwnerRepository interface {
	IsPlatformOwner(ctx context.Context, userID userdomain.UserID) (bool, error)
}

type CommunityTransactionManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, repositories CommunityRepositories) error) error
}

type CommunityRepositories interface {
	Communities() CommunityRepository
	Memberships() CommunityMembershipRepository
	Applications() CommunityApplicationRepository
}
