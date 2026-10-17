package storage

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPresignedURLsUsePublicEndpoint(t *testing.T) {
	storage, err := NewS3(S3Config{
		Endpoint:       "care-object-storage:9000",
		PublicEndpoint: "localhost:9010",
		AccessKey:      "test-access-key",
		SecretKey:      "test-secret-key",
		Bucket:         "care-private",
		Region:         "us-east-1",
	})
	require.NoError(t, err)

	uploadURL, _, err := storage.PresignUpload(
		context.Background(),
		"ticket/test-file",
		"application/pdf",
		"",
		15*time.Minute,
	)
	require.NoError(t, err)
	require.Equal(t, "localhost:9010", mustParseURL(t, uploadURL).Host)

	downloadURL, err := storage.PresignDownload(
		context.Background(),
		"ticket/test-file",
		10*time.Minute,
	)
	require.NoError(t, err)
	require.Equal(t, "localhost:9010", mustParseURL(t, downloadURL).Host)
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)
	return parsed
}
