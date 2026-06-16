package main

import (
	"context"

	"github.com/Versifine/cumt-nexus-api/internal/message/messageusecase"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/Versifine/cumt-nexus-api/internal/user/userusecase"
)

type publicUserDMCapabilityAdapter struct {
	messages *messageusecase.UseCase
}

func (adapter publicUserDMCapabilityAdapter) GetPublicUserDMCapability(ctx context.Context, viewerID userdomain.UserID, targetID userdomain.UserID) (userusecase.PublicUserDMCapability, error) {
	capability, err := adapter.messages.GetDMCapability(ctx, messageusecase.DMCapabilityInput{ViewerID: viewerID, TargetID: targetID})
	if err != nil {
		return userusecase.PublicUserDMCapability{}, err
	}
	return userusecase.PublicUserDMCapability{
		CanStart:             capability.CanStart,
		RequiresRequest:      capability.RequiresRequest,
		Reason:               capability.Reason,
		DirectConversationID: capability.DirectConversationID,
		ViewerRelation:       capability.ViewerRelation,
	}, nil
}
