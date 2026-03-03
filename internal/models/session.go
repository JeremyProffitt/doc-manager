package models

// Session represents an active user session, stored in DynamoDB with a TTL.
type Session struct {
	Token     string `json:"token" dynamodbav:"token"`
	UserEmail string `json:"userEmail" dynamodbav:"userEmail"`
	ExpiresAt int64  `json:"expiresAt" dynamodbav:"expiresAt"`
}
