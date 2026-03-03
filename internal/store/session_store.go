package store

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// DynamoSessionStore implements SessionStore using DynamoDB.
type DynamoSessionStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewSessionStore creates a new DynamoSessionStore.
func NewSessionStore(client *dynamodb.Client, tableName string) *DynamoSessionStore {
	return &DynamoSessionStore{client: client, tableName: tableName}
}

// CreateSession stores a new session with TTL.
func (s *DynamoSessionStore) CreateSession(session *models.Session) error {
	item, err := attributevalue.MarshalMap(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	_, err = s.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// GetSession retrieves a session. Returns nil, nil if not found or expired.
func (s *DynamoSessionStore) GetSession(token string) (*models.Session, error) {
	result, err := s.client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"token": &types.AttributeValueMemberS{Value: token},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	var session models.Session
	if err := attributevalue.UnmarshalMap(result.Item, &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	// Belt-and-suspenders expiration check alongside DynamoDB TTL
	if session.ExpiresAt < time.Now().Unix() {
		return nil, nil
	}

	return &session, nil
}

// DeleteSession removes a session.
func (s *DynamoSessionStore) DeleteSession(token string) error {
	_, err := s.client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"token": &types.AttributeValueMemberS{Value: token},
		},
	})
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
