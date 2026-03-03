package store

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// DynamoFieldStore implements FieldStore using DynamoDB.
// The table uses formId as partition key and version as sort key.
type DynamoFieldStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewFieldStore creates a new DynamoDB-backed FieldStore.
func NewFieldStore(client *dynamodb.Client, tableName string) *DynamoFieldStore {
	return &DynamoFieldStore{
		client:    client,
		tableName: tableName,
	}
}

// SaveFieldPlacement saves a new field placement version. It queries for the
// max existing version and auto-increments. Returns the new version number.
func (s *DynamoFieldStore) SaveFieldPlacement(placement *models.FieldPlacement) (int, error) {
	// Find the max existing version
	result, err := s.client.Query(context.Background(), &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("formId = :fid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":fid": &types.AttributeValueMemberS{Value: placement.FormID},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(1),
		ProjectionExpression: aws.String("#v"),
		ExpressionAttributeNames: map[string]string{
			"#v": "version",
		},
	})
	if err != nil {
		return 0, fmt.Errorf("querying max version: %w", err)
	}

	maxVersion := 0
	if len(result.Items) > 0 {
		var existing models.FieldPlacement
		if err := attributevalue.UnmarshalMap(result.Items[0], &existing); err != nil {
			return 0, fmt.Errorf("unmarshaling version: %w", err)
		}
		maxVersion = existing.Version
	}

	newVersion := maxVersion + 1
	placement.Version = newVersion

	item, err := attributevalue.MarshalMap(placement)
	if err != nil {
		return 0, fmt.Errorf("marshaling field placement: %w", err)
	}

	_, err = s.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return 0, fmt.Errorf("putting field placement: %w", err)
	}

	return newVersion, nil
}

// GetLatestFieldPlacement retrieves the most recent version of field placements
// for a form by querying with ScanIndexForward=false and Limit=1.
func (s *DynamoFieldStore) GetLatestFieldPlacement(formId string) (*models.FieldPlacement, error) {
	result, err := s.client.Query(context.Background(), &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("formId = :fid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":fid": &types.AttributeValueMemberS{Value: formId},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("querying latest field placement: %w", err)
	}
	if len(result.Items) == 0 {
		return nil, nil
	}

	var placement models.FieldPlacement
	if err := attributevalue.UnmarshalMap(result.Items[0], &placement); err != nil {
		return nil, fmt.Errorf("unmarshaling field placement: %w", err)
	}
	return &placement, nil
}

// GetFieldPlacement retrieves a specific version of field placements.
func (s *DynamoFieldStore) GetFieldPlacement(formId string, version int) (*models.FieldPlacement, error) {
	result, err := s.client.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"formId":  &types.AttributeValueMemberS{Value: formId},
			"version": &types.AttributeValueMemberN{Value: strconv.Itoa(version)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("getting field placement: %w", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	var placement models.FieldPlacement
	if err := attributevalue.UnmarshalMap(result.Item, &placement); err != nil {
		return nil, fmt.Errorf("unmarshaling field placement: %w", err)
	}
	return &placement, nil
}

// ListVersions returns metadata for all versions of field placements for a form,
// sorted by version descending. The Fields array is stripped from results.
func (s *DynamoFieldStore) ListVersions(formId string) ([]models.FieldPlacement, error) {
	result, err := s.client.Query(context.Background(), &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("formId = :fid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":fid": &types.AttributeValueMemberS{Value: formId},
		},
		ProjectionExpression: aws.String("formId, #v, createdAt, #src"),
		ExpressionAttributeNames: map[string]string{
			"#v":   "version",
			"#src": "source",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("querying field placement versions: %w", err)
	}

	var placements []models.FieldPlacement
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &placements); err != nil {
		return nil, fmt.Errorf("unmarshaling field placement versions: %w", err)
	}

	// Sort by version descending
	sort.Slice(placements, func(i, j int) bool {
		return placements[i].Version > placements[j].Version
	})

	return placements, nil
}

// DeleteAllVersions removes all field placement versions for a form.
// It queries all versions, then uses BatchWriteItem to delete them.
func (s *DynamoFieldStore) DeleteAllVersions(formId string) error {
	// Query all versions
	result, err := s.client.Query(context.Background(), &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("formId = :fid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":fid": &types.AttributeValueMemberS{Value: formId},
		},
		ProjectionExpression: aws.String("formId, #v"),
		ExpressionAttributeNames: map[string]string{
			"#v": "version",
		},
	})
	if err != nil {
		return fmt.Errorf("querying versions for deletion: %w", err)
	}

	if len(result.Items) == 0 {
		return nil
	}

	// BatchWriteItem can handle up to 25 items at a time
	for i := 0; i < len(result.Items); i += 25 {
		end := i + 25
		if end > len(result.Items) {
			end = len(result.Items)
		}

		var requests []types.WriteRequest
		for _, item := range result.Items[i:end] {
			requests = append(requests, types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{
					Key: map[string]types.AttributeValue{
						"formId":  item["formId"],
						"version": item["version"],
					},
				},
			})
		}

		_, err := s.client.BatchWriteItem(context.Background(), &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				s.tableName: requests,
			},
		})
		if err != nil {
			return fmt.Errorf("batch deleting versions: %w", err)
		}
	}

	return nil
}
