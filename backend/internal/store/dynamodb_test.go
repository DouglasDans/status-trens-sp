package store

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// fakeDynamo keeps items by code and serves Scan in pages of pageSize.
type fakeDynamo struct {
	items    map[string]map[string]types.AttributeValue
	pageSize int
	err      error
	tables   []string
}

func newFakeDynamo() *fakeDynamo {
	return &fakeDynamo{items: map[string]map[string]types.AttributeValue{}, pageSize: 100}
}

func (f *fakeDynamo) PutItem(_ context.Context, in *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.tables = append(f.tables, aws.ToString(in.TableName))
	f.items[in.Item["code"].(*types.AttributeValueMemberN).Value] = in.Item
	return &dynamodb.PutItemOutput{}, nil
}

func (f *fakeDynamo) Scan(_ context.Context, in *dynamodb.ScanInput, _ ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.tables = append(f.tables, aws.ToString(in.TableName))
	var codes []int
	for k := range f.items {
		n, _ := strconv.Atoi(k)
		codes = append(codes, n)
	}
	slices.Sort(codes) // stable pages
	start := 0
	if in.ExclusiveStartKey != nil {
		last, _ := strconv.Atoi(in.ExclusiveStartKey["code"].(*types.AttributeValueMemberN).Value)
		for start < len(codes) && codes[start] <= last {
			start++
		}
	}
	end := min(start+f.pageSize, len(codes))
	out := &dynamodb.ScanOutput{}
	for _, c := range codes[start:end] {
		out.Items = append(out.Items, f.items[strconv.Itoa(c)])
	}
	if end < len(codes) {
		out.LastEvaluatedKey = map[string]types.AttributeValue{"code": &types.AttributeValueMemberN{Value: strconv.Itoa(codes[end-1])}}
	}
	return out, nil
}

func TestDynamoDBKeepsWhatWasSaved(t *testing.T) {
	ctx := context.Background()
	db := newFakeDynamo()
	s := NewDynamoDB(db, "lines")
	want := status.Memory{State: status.Paralisada, LastNotified: status.Paralisada}

	if err := s.Save(ctx, 9, want); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, 7, status.Memory{State: status.Normal}); err != nil {
		t.Fatal(err)
	}

	lines, err := s.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[9] != want || lines[7] != (status.Memory{State: status.Normal}) {
		t.Errorf("Load() = %v, want line 9 = %v and line 7 normal", lines, want)
	}
	for _, table := range db.tables {
		if table != "lines" {
			t.Errorf("used table %q, want %q", table, "lines")
		}
	}
}

func TestDynamoDBStartsEmpty(t *testing.T) {
	lines, err := NewDynamoDB(newFakeDynamo(), "lines").Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Errorf("Load() = %v, want empty", lines)
	}
}

func TestDynamoDBLoadsEveryPage(t *testing.T) {
	ctx := context.Background()
	db := newFakeDynamo()
	db.pageSize = 2
	s := NewDynamoDB(db, "lines")
	for code := 1; code <= 5; code++ {
		_ = s.Save(ctx, code, status.Memory{State: status.Normal})
	}

	lines, err := s.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 5 {
		t.Errorf("Load() returned %d lines, want 5 (all pages)", len(lines))
	}
}

func TestDynamoDBReportsFailures(t *testing.T) {
	ctx := context.Background()
	db := newFakeDynamo()
	db.err = errors.New("throttled")
	s := NewDynamoDB(db, "lines")

	if _, err := s.Load(ctx); !errors.Is(err, db.err) {
		t.Errorf("Load() error = %v, want it to wrap %v", err, db.err)
	}
	if err := s.Save(ctx, 9, status.Memory{State: status.Normal}); !errors.Is(err, db.err) {
		t.Errorf("Save() error = %v, want it to wrap %v", err, db.err)
	}
}

func TestDynamoDBRejectsAMalformedItem(t *testing.T) {
	db := newFakeDynamo()
	db.items["9"] = map[string]types.AttributeValue{"code": &types.AttributeValueMemberS{Value: "nove"}}

	if _, err := NewDynamoDB(db, "lines").Load(context.Background()); err == nil {
		t.Error("Load() accepted an item whose code is not a number")
	}
}
