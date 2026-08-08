package storage

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithymiddleware "github.com/aws/smithy-go/middleware"
)

func NewS3Client(endpoint, region, accessKeyID, secretAccessKey string) (*s3.Client, error) {
	if endpoint == "" || region == "" || accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("s3 client: endpoint, region, and credentials are required")
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load s3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
		o.APIOptions = append(o.APIOptions, removeDisableGzip)
	})

	return client, nil
}

// Supabase S3 compatibility: Go SDK v2 adds DisableAcceptEncodingGzip middleware that breaks signing.
func removeDisableGzip(stack *smithymiddleware.Stack) error {
	_, err := stack.Finalize.Remove("DisableAcceptEncodingGzip")
	return err
}
