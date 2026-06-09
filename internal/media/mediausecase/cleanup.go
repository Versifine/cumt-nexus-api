package mediausecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/media/mediadomain"
)

const (
	defaultCleanupUnboundTTL = 24 * time.Hour
	defaultCleanupFailedTTL  = 24 * time.Hour
	defaultCleanupBatchLimit = 100
	maxCleanupBatchLimit     = 1000
)

func (uc *UseCase) CleanupExpiredAttachments(ctx context.Context, input CleanupExpiredAttachmentsInput) (CleanupExpiredAttachmentsResult, error) {
	if uc.attachments == nil {
		return CleanupExpiredAttachmentsResult{}, fmt.Errorf("media attachment repository is not configured")
	}
	if uc.storage == nil && !input.DryRun {
		return CleanupExpiredAttachmentsResult{}, fmt.Errorf("object storage is not configured")
	}

	unboundTTL := normalizeCleanupTTL(input.UnboundTTL, defaultCleanupUnboundTTL)
	failedTTL := normalizeCleanupTTL(input.FailedTTL, defaultCleanupFailedTTL)
	limit := normalizeCleanupLimit(input.Limit)
	now := uc.now().UTC()
	result := CleanupExpiredAttachmentsResult{
		DryRun:                input.DryRun,
		UnboundReadyBefore:    now.Add(-unboundTTL),
		FailedOrBlockedBefore: now.Add(-failedTTL),
		UnboundTTL:            unboundTTL.String(),
		FailedTTL:             failedTTL.String(),
	}

	var candidates []mediadomain.Attachment
	var err error
	if input.DryRun {
		candidates, err = uc.attachments.ListCleanupCandidates(ctx, result.UnboundReadyBefore, result.FailedOrBlockedBefore, limit)
	} else {
		candidates, err = uc.attachments.TakeCleanupCandidates(ctx, result.UnboundReadyBefore, result.FailedOrBlockedBefore, limit)
	}
	if err != nil {
		return CleanupExpiredAttachmentsResult{}, fmt.Errorf("list media cleanup candidates: %w", err)
	}

	result.Candidates = len(candidates)
	if input.DryRun {
		return result, nil
	}
	result.AttachmentsDeleted = len(candidates)

	for _, attachment := range candidates {
		deletedObjects, err := uc.deleteAttachmentObjects(ctx, attachment)
		result.ObjectsDeleted += deletedObjects
		if err != nil {
			result.Failures++
			continue
		}
	}
	return result, nil
}

func (uc *UseCase) deleteAttachmentObjects(ctx context.Context, attachment mediadomain.Attachment) (int, error) {
	objectKeys := cleanupObjectKeys(attachment)
	deletedObjects := 0
	for _, objectKey := range objectKeys {
		if err := uc.storage.DeleteObject(ctx, objectKey); err != nil {
			return deletedObjects, fmt.Errorf("delete media object %q: %w", objectKey, err)
		}
		deletedObjects++
	}
	return deletedObjects, nil
}

func cleanupObjectKeys(attachment mediadomain.Attachment) []string {
	objectKey := strings.TrimSpace(attachment.ObjectKey())
	thumbnailObjectKey := strings.TrimSpace(attachment.ThumbnailObjectKey())
	if thumbnailObjectKey == "" || thumbnailObjectKey == objectKey {
		return []string{objectKey}
	}
	return []string{objectKey, thumbnailObjectKey}
}

func normalizeCleanupTTL(value time.Duration, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}

func normalizeCleanupLimit(value int) int {
	if value <= 0 {
		return defaultCleanupBatchLimit
	}
	if value > maxCleanupBatchLimit {
		return maxCleanupBatchLimit
	}
	return value
}
