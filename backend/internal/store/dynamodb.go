package store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// DynamoAPI is the part of the DynamoDB client the store uses.
type DynamoAPI interface {
	Scan(ctx context.Context, in *dynamodb.ScanInput, opts ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	PutItem(ctx context.Context, in *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// DynamoDB keeps one item per line: code (number, the key), state and
// lastNotified. It survives restarts and deploys, unlike InMemory.
type DynamoDB struct {
	client DynamoAPI
	table  string
}

func NewDynamoDB(client DynamoAPI, table string) *DynamoDB {
	return &DynamoDB{client: client, table: table}
}

// Load scans the whole table: it holds one small item per line.
func (s *DynamoDB) Load(ctx context.Context) (map[int]status.Memory, error) {
	lines := map[int]status.Memory{}
	in := &dynamodb.ScanInput{TableName: aws.String(s.table)}
	for {
		out, err := s.client.Scan(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("scanning %s: %w", s.table, err)
		}
		for _, item := range out.Items {
			code, m, err := decode(item)
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", s.table, err)
			}
			lines[code] = m
		}
		if out.LastEvaluatedKey == nil {
			return lines, nil
		}
		in.ExclusiveStartKey = out.LastEvaluatedKey
	}
}

func (s *DynamoDB) Save(ctx context.Context, code int, m status.Memory) error {
	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item: map[string]types.AttributeValue{
			"code":         &types.AttributeValueMemberN{Value: strconv.Itoa(code)},
			"state":        &types.AttributeValueMemberS{Value: string(m.State)},
			"lastNotified": &types.AttributeValueMemberS{Value: string(m.LastNotified)},
		},
	})
	if err != nil {
		return fmt.Errorf("saving line %d in %s: %w", code, s.table, err)
	}
	return nil
}

func decode(item map[string]types.AttributeValue) (int, status.Memory, error) {
	n, ok := item["code"].(*types.AttributeValueMemberN)
	if !ok {
		return 0, status.Memory{}, fmt.Errorf("item without a numeric code: %v", item)
	}
	code, err := strconv.Atoi(n.Value)
	if err != nil {
		return 0, status.Memory{}, fmt.Errorf("code %q: %w", n.Value, err)
	}
	return code, status.Memory{State: text(item, "state"), LastNotified: text(item, "lastNotified")}, nil
}

// text reads a string attribute; a missing one is the empty state, which is
// what Memory uses for "not known yet" (e.g. an item edited by hand).
func text(item map[string]types.AttributeValue, name string) status.State {
	if s, ok := item[name].(*types.AttributeValueMemberS); ok {
		return status.State(s.Value)
	}
	return ""
}
