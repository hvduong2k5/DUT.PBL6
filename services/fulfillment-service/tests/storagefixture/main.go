// Disposable acceptance setup: initialize a private, versioned S3 bucket.
package main

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, e := minio.New(os.Getenv("S3_ENDPOINT"), &minio.Options{Creds: credentials.NewStaticV4(os.Getenv("S3_ACCESS_KEY"), os.Getenv("S3_SECRET_KEY"), ""), Secure: false})
	if e != nil {
		log.Fatal(e)
	}
	bucket := os.Getenv("S3_BUCKET")
	if e = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); e != nil {
		log.Fatal(e)
	}
	if e = client.SetBucketVersioning(ctx, bucket, minio.BucketVersioningConfiguration{Status: "Enabled"}); e != nil {
		log.Fatal(e)
	}
	version, e := client.GetBucketVersioning(ctx, bucket)
	if e != nil || version.Status != "Enabled" {
		log.Fatal("S3 versioning unavailable")
	}
	log.Print("private versioned fixture bucket ready")
}
