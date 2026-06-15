package authusecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/progression/progressionusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

type XPRecorder interface {
	GrantXP(ctx context.Context, input progressionusecase.GrantXPInput) error
}

func grantDailyLoginXP(ctx context.Context, recorder XPRecorder, userID userdomain.UserID, now time.Time) error {
	if recorder == nil {
		return nil
	}
	if err := recorder.GrantXP(ctx, progressionusecase.GrantXPInput{
		UserID:     userID,
		ActorID:    userID,
		SourceType: progressionusecase.XPSourceDailyLogin,
		SourceID:   now.UTC().Format(time.DateOnly),
	}); err != nil {
		return fmt.Errorf("grant daily login xp: %w", err)
	}
	return nil
}
