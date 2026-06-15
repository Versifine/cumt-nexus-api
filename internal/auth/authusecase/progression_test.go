package authusecase

import (
	"context"

	"github.com/Versifine/cumt-nexus-api/internal/progression/progressionusecase"
)

type fakeXPRecorder struct {
	inputs []progressionusecase.GrantXPInput
	err    error
}

func (f *fakeXPRecorder) GrantXP(ctx context.Context, input progressionusecase.GrantXPInput) error {
	f.inputs = append(f.inputs, input)
	return f.err
}
