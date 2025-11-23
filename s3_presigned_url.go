package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignedClient := s3.NewPresignClient(s3Client)
	presignedHTTPRequest, err := presignedClient.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		},
		s3.WithPresignExpires(expireTime),
	)
	if err != nil {
		return "", fmt.Errorf("Could't generate presigned requests: %v", err)
	}

	return presignedHTTPRequest.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return database.Video{}, fmt.Errorf("VideoUrl is nil")
	}
	splitUrl := strings.Split(*video.VideoURL, ",")
	if len(splitUrl) != 2 {
		return database.Video{}, fmt.Errorf("No bucket and key could be extracted")
	}

	bucket := splitUrl[0]
	key := splitUrl[1]

	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute*3)
	if err != nil {
		return database.Video{}, err
	}

	video.VideoURL = &presignedURL
	return video, nil
}
