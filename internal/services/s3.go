package services

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client defines the subset of the S3 API used by S3Service.
type S3Client interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// S3PresignClient defines the presigning methods used by S3Service.
type S3PresignClient interface {
	PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
	PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// S3Service provides operations for interacting with S3 for form storage.
type S3Service struct {
	client        S3Client
	presignClient S3PresignClient
	bucketName    string
}

// NewS3Service creates a new S3Service with the given clients and bucket.
func NewS3Service(client S3Client, presignClient S3PresignClient, bucketName string) *S3Service {
	return &S3Service{
		client:        client,
		presignClient: presignClient,
		bucketName:    bucketName,
	}
}

// GenerateUploadURL generates a pre-signed PUT URL for uploading a form PDF.
// Returns the URL, the S3 key, and an error.
func (s *S3Service) GenerateUploadURL(formId, filename, contentType string) (string, string, error) {
	s3Key := fmt.Sprintf("forms/%s/%s", formId, filename)

	req, err := s.presignClient.PresignPutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(s3Key),
		ContentType: aws.String(contentType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 5 * time.Minute
	})
	if err != nil {
		return "", "", fmt.Errorf("presigning put object: %w", err)
	}

	return req.URL, s3Key, nil
}

// GenerateDownloadURL generates a pre-signed GET URL for downloading a form PDF.
func (s *S3Service) GenerateDownloadURL(s3Key string) (string, error) {
	req, err := s.presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 15 * time.Minute
	})
	if err != nil {
		return "", fmt.Errorf("presigning get object: %w", err)
	}

	return req.URL, nil
}

// GetObject retrieves the raw bytes of an S3 object.
func (s *S3Service) GetObject(s3Key string) ([]byte, error) {
	result, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return nil, fmt.Errorf("getting s3 object: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("reading s3 object body: %w", err)
	}
	return data, nil
}

// DeleteObject deletes an object from S3.
func (s *S3Service) DeleteObject(s3Key string) error {
	_, err := s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("deleting s3 object: %w", err)
	}
	return nil
}
