package authcontext

import (
	"context"

	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type currentUserIDKey struct{}

func WithCurrentUserID(ctx context.Context, userID userdomain.UserID) context.Context {
	return context.WithValue(ctx, currentUserIDKey{}, userID)
}

func CurrentUserID(ctx context.Context) (userdomain.UserID, bool) {
	value := ctx.Value(currentUserIDKey{})
	userID, ok := value.(userdomain.UserID)
	return userID, ok
}
