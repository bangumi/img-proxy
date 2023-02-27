package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/samber/lo"
)

func newS3Client() *s3.S3 {
	// Initialize s3 client object.
	c := credentials.NewStaticCredentials(s3accessKey, s3secretKey, "")
	s := lo.Must(session.NewSession(&aws.Config{
		Credentials:      c,
		Endpoint:         &s3entryPoint,
		Region:           lo.ToPtr("us-east-1"),
		DisableSSL:       lo.ToPtr(true),
		S3ForcePathStyle: lo.ToPtr(true),
	}))
	svc := s3.New(s)

	return svc
}
