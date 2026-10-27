package store

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/DouglasDans/status-trens-sp/backend/internal/board"
	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// fakeBoardDynamo keeps items by their "id" key.
type fakeBoardDynamo struct {
	items  map[string]map[string]types.AttributeValue
	err    error
	tables []string
}

func newFakeBoardDynamo() *fakeBoardDynamo {
	return &fakeBoardDynamo{items: map[string]map[string]types.AttributeValue{}}
}

func (f *fakeBoardDynamo) PutItem(_ context.Context, in *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.tables = append(f.tables, aws.ToString(in.TableName))
	f.items[in.Item["id"].(*types.AttributeValueMemberS).Value] = in.Item
	return &dynamodb.PutItemOutput{}, nil
}

func (f *fakeBoardDynamo) GetItem(_ context.Context, in *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.tables = append(f.tables, aws.ToString(in.TableName))
	return &dynamodb.GetItemOutput{Item: f.items[in.Key["id"].(*types.AttributeValueMemberS).Value]}, nil
}

var sampleBoard = board.Board{
	UpdatedAt: time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC),
	Sources:   []string{"proximotrem", "metro"},
	Lines: []board.Line{
		{Code: 3, Name: "Vermelha", ColorHex: "#ed2e38", State: status.Paralisada, Label: "Paralisada", Description: "Falha de energia.", SeenAt: time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)},
		{Code: 6, Name: "Laranja", State: status.Normal, Label: "Operação Normal", SeenAt: time.Date(2026, 10, 26, 11, 55, 0, 0, time.UTC)},
	},
}

func TestBoardStoresRoundTrip(t *testing.T) {
	stores := map[string]interface {
		SaveBoard(context.Context, board.Board) error
		LoadBoard(context.Context) (board.Board, bool, error)
	}{
		"in memory": NewInMemoryBoard(),
		"dynamodb":  NewDynamoBoard(newFakeBoardDynamo(), "boards"),
	}
	for name, s := range stores {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			if _, found, err := s.LoadBoard(ctx); err != nil || found {
				t.Fatalf("empty store: found=%v err=%v, want nothing", found, err)
			}
			if err := s.SaveBoard(ctx, sampleBoard); err != nil {
				t.Fatal(err)
			}
			got, found, err := s.LoadBoard(ctx)
			if err != nil || !found {
				t.Fatalf("found=%v err=%v", found, err)
			}
			if !reflect.DeepEqual(got, sampleBoard) {
				t.Errorf("got %+v, want %+v", got, sampleBoard)
			}
		})
	}
}

// R15.3: the board lives in its own table, never among the line memories
// that Load scans.
func TestDynamoBoardUsesItsOwnTable(t *testing.T) {
	db := newFakeBoardDynamo()
	s := NewDynamoBoard(db, "status-trens-sp-board")

	_ = s.SaveBoard(context.Background(), sampleBoard)
	_, _, _ = s.LoadBoard(context.Background())

	for _, table := range db.tables {
		if table != "status-trens-sp-board" {
			t.Errorf("used table %q, want only status-trens-sp-board", table)
		}
	}
}

// R16.4
func TestDynamoBoardRejectsACorruptItem(t *testing.T) {
	db := newFakeBoardDynamo()
	db.items["current"] = map[string]types.AttributeValue{
		"id":    &types.AttributeValueMemberS{Value: "current"},
		"board": &types.AttributeValueMemberS{Value: `{"updatedAt":`},
	}

	if _, _, err := NewDynamoBoard(db, "boards").LoadBoard(context.Background()); err == nil {
		t.Error("want error, got nil")
	}
}

func TestDynamoBoardReportsClientErrors(t *testing.T) {
	db := newFakeBoardDynamo()
	db.err = errors.New("throttled")
	s := NewDynamoBoard(db, "boards")

	if err := s.SaveBoard(context.Background(), sampleBoard); err == nil {
		t.Error("save: want error, got nil")
	}
	if _, _, err := s.LoadBoard(context.Background()); err == nil {
		t.Error("load: want error, got nil")
	}
}

func TestInMemoryBoardIsACopy(t *testing.T) {
	s := NewInMemoryBoard()
	b := sampleBoard
	b.Lines = append([]board.Line(nil), sampleBoard.Lines...)
	_ = s.SaveBoard(context.Background(), b)

	b.Lines[0].State = status.Normal
	got, _, _ := s.LoadBoard(context.Background())
	if got.Lines[0].State != status.Paralisada {
		t.Error("changing the saved board after SaveBoard changed the stored one")
	}
}
