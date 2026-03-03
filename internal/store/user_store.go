package store

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// DynamoUserStore implements UserStore using DynamoDB.
type DynamoUserStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewUserStore creates a new DynamoUserStore.
func NewUserStore(client *dynamodb.Client, tableName string) *DynamoUserStore {
	return &DynamoUserStore{client: client, tableName: tableName}
}

// GetUser fetches a user by email. Returns nil, nil if not found.
func (s *DynamoUserStore) GetUser(email string) (*models.User, error) {
	result, err := s.client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"email": &types.AttributeValueMemberS{Value: email},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	var user models.User
	if err := attributevalue.UnmarshalMap(result.Item, &user); err != nil {
		return nil, fmt.Errorf("unmarshal user: %w", err)
	}
	return &user, nil
}

// CreateUser creates a new user. Returns error if email already exists.
func (s *DynamoUserStore) CreateUser(user *models.User) error {
	item, err := attributevalue.MarshalMap(user)
	if err != nil {
		return fmt.Errorf("marshal user: %w", err)
	}

	_, err = s.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(email)"),
	})
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}
