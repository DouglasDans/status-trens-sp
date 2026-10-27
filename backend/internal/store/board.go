package store

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/DouglasDans/status-trens-sp/backend/internal/board"
)

// InMemoryBoard keeps the last board for local runs and tests.
type InMemoryBoard struct {
	mu    sync.Mutex
	board *board.Board
}

func NewInMemoryBoard() *InMemoryBoard { return &InMemoryBoard{} }

func (s *InMemoryBoard) SaveBoard(_ context.Context, b board.Board) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b.Lines = slices.Clone(b.Lines)
	b.Sources = slices.Clone(b.Sources)
	s.board = &b
	return nil
}

// LoadBoard reports found=false until the first SaveBoard.
func (s *InMemoryBoard) LoadBoard(context.Context) (board.Board, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.board == nil {
		return board.Board{}, false, nil
	}
	b := *s.board
	b.Lines = slices.Clone(b.Lines)
	b.Sources = slices.Clone(b.Sources)
	return b, true, nil
}

// BoardAPI is the part of the DynamoDB client the board store uses.
type BoardAPI interface {
	GetItem(ctx context.Context, in *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, in *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// boardID is the key of the single item: the whole board is written in one
// PutItem, so a reader never sees half of a cycle.
const boardID = "current"

// DynamoBoard uses a table of its own (key "id", string) holding one
// item whose "board" attribute is the board as JSON. Kept apart from the line
// memory table, which Load scans in full.
type DynamoBoard struct {
	client BoardAPI
	table  string
}

func NewDynamoBoard(client BoardAPI, table string) *DynamoBoard {
	return &DynamoBoard{client: client, table: table}
}

func (s *DynamoBoard) SaveBoard(ctx context.Context, b board.Board) error {
	data, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("encoding board: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item: map[string]types.AttributeValue{
			"id":    &types.AttributeValueMemberS{Value: boardID},
			"board": &types.AttributeValueMemberS{Value: string(data)},
		},
	})
	if err != nil {
		return fmt.Errorf("saving board in %s: %w", s.table, err)
	}
	return nil
}

func (s *DynamoBoard) LoadBoard(ctx context.Context) (board.Board, bool, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key:       map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: boardID}},
	})
	if err != nil {
		return board.Board{}, false, fmt.Errorf("reading board from %s: %w", s.table, err)
	}
	if out.Item == nil {
		return board.Board{}, false, nil
	}
	data, ok := out.Item["board"].(*types.AttributeValueMemberS)
	if !ok {
		return board.Board{}, false, fmt.Errorf("board item in %s has no board attribute", s.table)
	}
	var b board.Board
	if err := json.Unmarshal([]byte(data.Value), &b); err != nil {
		return board.Board{}, false, fmt.Errorf("decoding board from %s: %w", s.table, err)
	}
	return b, true, nil
}
