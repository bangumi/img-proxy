package main

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
)

func s3() *minio.Client {
	// Initialize minio client object.
	minioClient, err := minio.New(s3entryPoint, &minio.Options{
		Creds: credentials.NewStaticV4(s3accessKey, s3secretKey, ""),
	})
	if err != nil {
		log.Fatalln(err)
	}

	return minioClient
}
