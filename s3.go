package main

import (
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func newS3Client() *s3.Client {
	// Initialize s3 client object.
	svc := s3.New(s3.Options{
		BaseEndpoint: &s3entryPoint,
		Region:       "us-east-1",
		UsePathStyle: true,
		Credentials:  credentials.NewStaticCredentialsProvider(s3accessKey, s3secretKey, ""),
	})

	return svc
}
