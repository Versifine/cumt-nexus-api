package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/Versifine/cumt-nexus-api/internal/media/mediausecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var _ mediausecase.ObjectStorage = (*R2ObjectStorage)(nil)

type R2ObjectStorage struct {
	client        *s3.Client
	bucket        string
	publicBaseURL string
}

func NewR2ObjectStorage(ctx context.Context, cfg config.ObjectStorageConfig) (*R2ObjectStorage, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load r2 client config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(cfg.Endpoint)
		options.UsePathStyle = cfg.ForcePathStyle
	})

	return &R2ObjectStorage{
		client:        client,
		bucket:        cfg.Bucket,
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
	}, nil
}

func (storage *R2ObjectStorage) PutObject(ctx context.Context, input mediausecase.PutObjectInput) (mediausecase.PutObjectResult, error) {
	if err := validateObjectKey(input.ObjectKey); err != nil {
		return mediausecase.PutObjectResult{}, err
	}

	_, err := storage.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(storage.bucket),
		Key:           aws.String(input.ObjectKey),
		Body:          input.Body,
		ContentLength: aws.Int64(input.SizeBytes),
		ContentType:   aws.String(input.ContentType),
	})
	if err != nil {
		return mediausecase.PutObjectResult{}, fmt.Errorf("put r2 object: %w", err)
	}

	return mediausecase.PutObjectResult{
		StorageProvider: "r2",
		Bucket:          storage.bucket,
		ObjectKey:       input.ObjectKey,
		PublicURL:       storage.publicURL(input.ObjectKey),
	}, nil
}

func (storage *R2ObjectStorage) publicURL(objectKey string) string {
	return storage.publicBaseURL + "/" + strings.TrimLeft(objectKey, "/")
}
