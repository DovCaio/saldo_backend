package application

import (
	"context"
	"encoding/csv"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DovCaio/saldo/internal/adapter/dto"
	"github.com/DovCaio/saldo/internal/transaction"
)

type fakeParser struct {
	transactions []transaction.Transaction
	err          error
	called       bool
}

func (f *fakeParser) Parse(
	parser BankParser,
	csvReader *csv.Reader,
) ([]transaction.Transaction, error) {
	f.called = true

	if f.err != nil {
		return nil, f.err
	}

	return f.transactions, nil
}

type fakeRepository struct {
	saved []transaction.Transaction
	err   error
	calls int
}

func (f *fakeRepository) SaveBatch(
	ctx context.Context,
	transactions []transaction.Transaction,
) ([]transaction.Transaction, error) {
	f.calls++

	if f.err != nil {
		return nil, f.err
	}

	f.saved = append(f.saved, transactions...)
	return transactions, nil
}

func (f *fakeRepository) FindAll(
	ctx context.Context,
	filter transaction.TransactionFilter,
) ([]transaction.Transaction, error) {
	return nil, nil
}

func (f *fakeRepository) Summary(
	ctx context.Context,
	filter transaction.TransactionSummaryFilter,
) (*dto.SummaryResponse, error) {
	return nil, nil
}

type fakePublisher struct {
	err       error
	published []transaction.Transaction
	calls     int
}

func (f *fakePublisher) Publish(
	ctx context.Context,
	tx transaction.Transaction,
) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	f.published = append(f.published, tx)
	return nil
}

func TestImportTransactionsExecute(t *testing.T) {
	validTransactions := []transaction.Transaction{
		{
			ID:          "tx-1",
			Date:        time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			Description: "Salario",
			AmountCents: 500000,
			Type:        transaction.Income,
		},
	}

	tests := []struct {
		name             string
		input            string
		transactions     []transaction.Transaction
		parserError      error
		repositoryError  error
		publisher        *fakePublisher
		useNilPublisher  bool
		wantError        bool
		wantErrorIs      error
		wantSaved        int
		wantRepoCalls    int
		wantParserCalled bool
		wantPublished    int
	}{
		{
			name:             "imports parsed transactions",
			input:            "Data,Descrição,Valor,Identificador\n",
			transactions:     validTransactions,
			publisher:        &fakePublisher{},
			wantSaved:        1,
			wantRepoCalls:    1,
			wantParserCalled: true,
			wantPublished:    1,
		},
		{
			name:             "propagates parser error",
			input:            "Data,Descrição,Valor,Identificador\n",
			parserError:      errors.New("invalid CSV"),
			publisher:        &fakePublisher{},
			wantError:        true,
			wantParserCalled: true,
		},
		{
			name:             "rejects empty import",
			input:            "Data,Descrição,Valor,Identificador\n",
			publisher:        &fakePublisher{},
			wantError:        true,
			wantParserCalled: true,
		},
		{
			name:             "propagates repository error",
			input:            "Data,Descrição,Valor,Identificador\n",
			transactions:     validTransactions,
			repositoryError:  errors.New("database unavailable"),
			publisher:        &fakePublisher{},
			wantError:        true,
			wantRepoCalls:    1,
			wantParserCalled: true,
		},
		{
			name:      "rejects empty CSV",
			input:     "",
			publisher: &fakePublisher{},
			wantError: true,
		},
		{
			name:      "rejects malformed CSV header",
			input:     "\"unterminated",
			publisher: &fakePublisher{},
			wantError: true,
		},
		{
			name:      "rejects unsupported bank header",
			input:     "foo,bar,baz,qux\n",
			publisher: &fakePublisher{},
			wantError: true,
		},
		{
			name:             "returns error when publisher is nil",
			input:            "Data,Descrição,Valor,Identificador\n",
			transactions:     validTransactions,
			useNilPublisher:  true,
			wantError:        true,
			wantErrorIs:      UninitializedPublisher,
			wantSaved:        1,
			wantRepoCalls:    1,
			wantParserCalled: true,
		},
		{
			name:             "propagates publisher error",
			input:            "Data,Descrição,Valor,Identificador\n",
			transactions:     validTransactions,
			publisher:        &fakePublisher{err: errors.New("broker unavailable")},
			wantError:        true,
			wantSaved:        1,
			wantRepoCalls:    1,
			wantParserCalled: true,
			wantPublished:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &fakeParser{
				transactions: tt.transactions,
				err:          tt.parserError,
			}

			repo := &fakeRepository{
				err: tt.repositoryError,
			}

			service := transaction.NewService(repo)

			var useCase *ImportTransactions
			if tt.useNilPublisher {
				useCase = NewImportTransactions(parser, service, nil)
			} else {
				useCase = NewImportTransactions(parser, service, tt.publisher)
			}
			err := useCase.Execute(
				context.Background(),
				strings.NewReader(tt.input),
			)

			if (err != nil) != tt.wantError {
				t.Fatalf("Execute() error = %v, wantError = %v",
					err, tt.wantError)
			}

			if tt.wantErrorIs != nil && !errors.Is(err, tt.wantErrorIs) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErrorIs)
			}

			if parser.called != tt.wantParserCalled {
				t.Errorf("parser.called = %v, want %v", parser.called, tt.wantParserCalled)
			}

			if len(repo.saved) != tt.wantSaved {
				t.Errorf("got %d saved transactions, want %d",
					len(repo.saved), tt.wantSaved)
			}

			if repo.calls != tt.wantRepoCalls {
				t.Errorf("got %d repository calls, want %d",
					repo.calls, tt.wantRepoCalls)
			}

			if tt.wantSaved > 0 &&
				!reflect.DeepEqual(repo.saved, tt.transactions) {
				t.Error("saved transactions differ from parsed transactions")
			}

			if tt.publisher != nil && len(tt.publisher.published) != tt.wantPublished {
				t.Errorf("got %d published transactions, want %d",
					len(tt.publisher.published), tt.wantPublished)
			}
		})
	}
}

func TestSendToMessagingPublishesAllTransactions(t *testing.T) {
	publisher := &fakePublisher{}
	useCase := NewImportTransactions(nil, nil, publisher)

	txs := []transaction.Transaction{
		{ID: "1", Description: "A"},
		{ID: "2", Description: "B"},
	}

	if err := useCase.sendToMessaging(context.Background(), txs); err != nil {
		t.Fatalf("sendToMessaging() error = %v", err)
	}

	if !reflect.DeepEqual(publisher.published, txs) {
		t.Errorf("published = %#v, want %#v", publisher.published, txs)
	}
}

func TestSendToMessagingStopsOnFirstPublishError(t *testing.T) {
	publisher := &fakePublisher{err: errors.New("nack")}
	useCase := NewImportTransactions(nil, nil, publisher)

	err := useCase.sendToMessaging(context.Background(), []transaction.Transaction{
		{ID: "1"},
		{ID: "2"},
	})
	if err == nil {
		t.Fatal("expected publish error")
	}
	if publisher.calls != 1 {
		t.Errorf("publisher calls = %d, want 1", publisher.calls)
	}
}
