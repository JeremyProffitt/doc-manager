package services

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// mockS3Client implements S3Client for testing.
type mockS3Client struct {
	putObjectFn    func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	getObjectFn    func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	deleteObjectFn func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putObjectFn != nil {
		return m.putObjectFn(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.getObjectFn != nil {
		return m.getObjectFn(ctx, params, optFns...)
	}
	return &s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader("test content")),
	}, nil
}

func (m *mockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if m.deleteObjectFn != nil {
		return m.deleteObjectFn(ctx, params, optFns...)
	}
	return &s3.DeleteObjectOutput{}, nil
}

// mockPresignClient implements S3PresignClient for testing.
type mockPresignClient struct {
	presignPutFn func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
	presignGetFn func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

func (m *mockPresignClient) PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	if m.presignPutFn != nil {
		return m.presignPutFn(ctx, params, optFns...)
	}
	return &v4.PresignedHTTPRequest{
		URL: "https://s3.amazonaws.com/test-bucket/test-key?X-Amz-Signature=abc123",
	}, nil
}

func (m *mockPresignClient) PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	if m.presignGetFn != nil {
		return m.presignGetFn(ctx, params, optFns...)
	}
	return &v4.PresignedHTTPRequest{
		URL: "https://s3.amazonaws.com/test-bucket/test-key?X-Amz-Signature=def456",
	}, nil
}

func TestS3Service(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, svc *S3Service)
	}{
		{
			name: "GenerateUploadURL returns non-empty URL",
			run: func(t *testing.T, svc *S3Service) {
				url, s3Key, err := svc.GenerateUploadURL("form-123", "test.pdf", "application/pdf")
				if err != nil {
					t.Fatalf("GenerateUploadURL failed: %v", err)
				}
				if url == "" {
					t.Error("expected non-empty URL")
				}
				if s3Key == "" {
					t.Error("expected non-empty s3Key")
				}
				if !strings.Contains(s3Key, "form-123") {
					t.Errorf("expected s3Key to contain form ID, got %s", s3Key)
				}
			},
		},
		{
			name: "GenerateDownloadURL returns non-empty URL",
			run: func(t *testing.T, svc *S3Service) {
				url, err := svc.GenerateDownloadURL("forms/form-123/test.pdf")
				if err != nil {
					t.Fatalf("GenerateDownloadURL failed: %v", err)
				}
				if url == "" {
					t.Error("expected non-empty URL")
				}
			},
		},
		{
			name: "GetObject returns content",
			run: func(t *testing.T, svc *S3Service) {
				data, err := svc.GetObject("forms/form-123/test.pdf")
				if err != nil {
					t.Fatalf("GetObject failed: %v", err)
				}
				if len(data) == 0 {
					t.Error("expected non-empty data")
				}
			},
		},
		{
			name: "DeleteObject succeeds",
			run: func(t *testing.T, svc *S3Service) {
				err := svc.DeleteObject("forms/form-123/test.pdf")
				if err != nil {
					t.Fatalf("DeleteObject failed: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewS3Service(&mockS3Client{}, &mockPresignClient{}, "test-bucket")
			tt.run(t, svc)
		})
	}
}

func TestS3ServiceErrors(t *testing.T) {
	errTest := errors.New("s3 error")

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "GenerateUploadURL returns error when presign fails",
			run: func(t *testing.T) {
				svc := NewS3Service(&mockS3Client{}, &mockPresignClient{
					presignPutFn: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
						return nil, errTest
					},
				}, "test-bucket")
				_, _, err := svc.GenerateUploadURL("form-1", "test.pdf", "application/pdf")
				if err == nil {
					t.Fatal("expected error")
				}
			},
		},
		{
			name: "GenerateDownloadURL returns error when presign fails",
			run: func(t *testing.T) {
				svc := NewS3Service(&mockS3Client{}, &mockPresignClient{
					presignGetFn: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
						return nil, errTest
					},
				}, "test-bucket")
				_, err := svc.GenerateDownloadURL("forms/form-1/test.pdf")
				if err == nil {
					t.Fatal("expected error")
				}
			},
		},
		{
			name: "GetObject returns error when S3 fails",
			run: func(t *testing.T) {
				svc := NewS3Service(&mockS3Client{
					getObjectFn: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
						return nil, errTest
					},
				}, &mockPresignClient{}, "test-bucket")
				_, err := svc.GetObject("forms/form-1/test.pdf")
				if err == nil {
					t.Fatal("expected error")
				}
			},
		},
		{
			name: "DeleteObject returns error when S3 fails",
			run: func(t *testing.T) {
				svc := NewS3Service(&mockS3Client{
					deleteObjectFn: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
						return nil, errTest
					},
				}, &mockPresignClient{}, "test-bucket")
				err := svc.DeleteObject("forms/form-1/test.pdf")
				if err == nil {
					t.Fatal("expected error")
				}
			},
		},
		{
			name: "GenerateUploadURL sets correct S3 key format",
			run: func(t *testing.T) {
				svc := NewS3Service(&mockS3Client{}, &mockPresignClient{}, "my-bucket")
				_, s3Key, err := svc.GenerateUploadURL("abc-123", "document.pdf", "application/pdf")
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				expected := "forms/abc-123/document.pdf"
				if s3Key != expected {
					t.Errorf("expected s3Key %q, got %q", expected, s3Key)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}
