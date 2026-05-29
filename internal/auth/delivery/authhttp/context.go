package authhttp

import (
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

const currentUserIDKey = "current_user_id"

func SetCurrentUserID(c *gin.Context, userID userdomain.UserID) {
	c.Set(currentUserIDKey, userID)
}

func CurrentUserID(c *gin.Context) (userID userdomain.UserID, ok bool) {
	value, ok := c.Get(currentUserIDKey)
	if !ok {
		return "", false
	}
	userID, ok = value.(userdomain.UserID)
	return userID, ok
}
