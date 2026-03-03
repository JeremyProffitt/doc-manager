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

// NewUserStore creates a new DynamoDB-backed UserStore.
func NewUserStore(client *dynamodb.Client, tableName string) *DynamoUserStore {
	return &DynamoUserStore{
		client:    client,
		tableName: tableName,
	}
}

// CreateUser creates a new user in DynamoDB. Returns an error if the user
// already exists (uses a condition expression to prevent overwrites).
func (s *DynamoUserStore) CreateUser(user *models.User) error {
	item, err := attributevalue.MarshalMap(user)
	if err != nil {
		return fmt.Errorf("marshaling user: %w", err)
	}

	_, err = s.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(email)"),
	})
	if err != nil {
		return fmt.Errorf("putting user item: %w", err)
	}
	return nil
}

// GetUser retrieves a user by email. Returns nil if the user does not exist.
func (s *DynamoUserStore) GetUser(email string) (*models.User, error) {
	result, err := s.client.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"email": &types.AttributeValueMemberS{Value: email},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("getting user item: %w", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	var user models.User
	if err := attributevalue.UnmarshalMap(result.Item, &user); err != nil {
		return nil, fmt.Errorf("unmarshaling user: %w", err)
	}
	return &user, nil
}
