package models

// User represents an authenticated user of the Doc-Manager application.
type User struct {
	Email        string `json:"email" dynamodbav:"email"`
	PasswordHash string `json:"-" dynamodbav:"passwordHash"`
	Name         string `json:"name" dynamodbav:"name"`
	CreatedAt    string `json:"createdAt" dynamodbav:"createdAt"`
}
