package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const ImageSourceTypeS3 ImageSourceType = "s3"

func init() {
	RegisterSource(ImageSourceTypeS3, NewS3ImageSource)
}

func newS3Session(region string) (*session.Session, error) {
	return session.NewSession(&aws.Config{
		Region: &region,
		Credentials: credentials.NewStaticCredentials(
			os.Getenv("S3_KEY"),
			os.Getenv("S3_KEY_SECRET"),
			"",
		),
	})
}

type S3ImageSource struct {
	Config *SourceConfig
}

func NewS3ImageSource(config *SourceConfig) ImageSource {
	return &S3ImageSource{Config: config}
}

func (s *S3ImageSource) Matches(r *http.Request) bool {
	return r.Method == http.MethodGet && parseS3Key(r) != ""
}

func (s *S3ImageSource) GetImage(req *http.Request) ([]byte, error) {
	key, bucket, region := parseS3Key(req), parseS3Bucket(req), parseS3Region(req)

	session, err := newS3Session(region)
	if err != nil {
		return nil, fmt.Errorf("failed to create s3 session: %w", err)
	}

	buffer := aws.NewWriteAtBuffer([]byte{})
	buffer.GrowthCoeff = 1.5
	if _, err := s3manager.NewDownloader(session).
		Download(
			buffer,
			&s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    &key,
			}); err != nil {
		return nil, fmt.Errorf("failed to download file, %w", err)
	}

	return buffer.Bytes(), nil
}

func uploadBufferToS3(buffer []byte, outputKey, bucket, region string) error {
	sess, err := newS3Session(region)
	if err != nil {
		return fmt.Errorf("failed to create s3 session: %w", err)
	}

	if _, err := s3manager.NewUploader(sess).
		Upload(&s3manager.UploadInput{
			Bucket: aws.String(bucket),
			Key:    &outputKey,
			Body:   bytes.NewReader(buffer),
		}); err != nil {
		return fmt.Errorf("failed to upload file, %w", err)
	}

	return nil
}

func parseS3Key(request *http.Request) string {
	return request.URL.Query().Get("s3key")
}

func parseS3OutputKey(request *http.Request) string {
	return request.URL.Query().Get("outputKey")
}

func parseS3Bucket(request *http.Request) string {
	return request.URL.Query().Get("bucket")
}

func parseS3Region(request *http.Request) string {
	return request.URL.Query().Get("region")
}

type S3Source struct {
	Zone string
}

func (s *S3Source) DownloadImage(container, imageKey string) ([]byte, error) {
	session, err := newS3Session(s.Zone)
	if err != nil {
		return nil, fmt.Errorf("failed to create s3 session: %w", err)
	}

	buffer := aws.NewWriteAtBuffer([]byte{})
	buffer.GrowthCoeff = 1.5
	if _, err := s3manager.NewDownloader(session).
		Download(
			buffer,
			&s3.GetObjectInput{
				Bucket: aws.String(container),
				Key:    &imageKey,
			}); err != nil {
		return nil, fmt.Errorf("sources3: failed to download image, %w", err)
	}

	return buffer.Bytes(), nil
}

func (s *S3Source) UploadImage(data []byte, fileKey, container string) error {
	sess, err := newS3Session(s.Zone)
	if err != nil {
		return fmt.Errorf("failed to create s3 session: %w", err)
	}

	if _, err := s3manager.NewUploader(sess).
		Upload(&s3manager.UploadInput{
			Bucket: aws.String(container),
			Key:    &fileKey,
			Body:   bytes.NewReader(data),
		}); err != nil {
		return fmt.Errorf("failed to upload file, %w", err)
	}

	return nil
}
