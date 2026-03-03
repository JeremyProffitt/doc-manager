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

// DynamoSettingsStore implements SettingsStore using DynamoDB.
type DynamoSettingsStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewSettingsStore creates a new DynamoDB-backed SettingsStore.
func NewSettingsStore(client *dynamodb.Client, tableName string) *DynamoSettingsStore {
	return &DynamoSettingsStore{
		client:    client,
		tableName: tableName,
	}
}

// GetSetting retrieves a setting by key. Returns nil if the setting does not exist.
func (s *DynamoSettingsStore) GetSetting(key string) (*models.Setting, error) {
	result, err := s.client.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"key": &types.AttributeValueMemberS{Value: key},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("getting setting item: %w", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	var setting models.Setting
	if err := attributevalue.UnmarshalMap(result.Item, &setting); err != nil {
		return nil, fmt.Errorf("unmarshaling setting: %w", err)
	}
	return &setting, nil
}

// PutSetting creates or updates a setting in DynamoDB. Uses a condition
// expression to prevent overwriting existing settings.
func (s *DynamoSettingsStore) PutSetting(setting *models.Setting) error {
	item, err := attributevalue.MarshalMap(setting)
	if err != nil {
		return fmt.Errorf("marshaling setting: %w", err)
	}

	_, err = s.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(#k)"),
		ExpressionAttributeNames: map[string]string{
			"#k": "key",
		},
	})
	if err != nil {
		return fmt.Errorf("putting setting item: %w", err)
	}
	return nil
}
