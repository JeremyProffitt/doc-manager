package models

// Form represents an uploaded PDF form and its metadata.
type Form struct {
	ID          string `json:"id" dynamodbav:"id"`
	UserID      string `json:"userId" dynamodbav:"userId"`
	Name        string `json:"name" dynamodbav:"name"`
	Status      string `json:"status" dynamodbav:"status"` // uploading, uploaded, analyzing, analyzed, error
	S3Key       string `json:"s3Key" dynamodbav:"s3Key"`
	ContentType string `json:"contentType" dynamodbav:"contentType"`
	FontFamily  string `json:"fontFamily" dynamodbav:"fontFamily"`
	FontSize    int    `json:"fontSize" dynamodbav:"fontSize"`
	CreatedAt   string `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt   string `json:"updatedAt" dynamodbav:"updatedAt"`
}
