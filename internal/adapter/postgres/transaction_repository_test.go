package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/DovCaio/saldo/internal/transaction"
)

func newRepo(t *testing.T) (*TransactionRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	return NewTransactionRepository(db), mock
}

func TestSaveBatchRejectsEmptyImport(t *testing.T) {
	repo, _ := newRepo(t)

	got, err := repo.SaveBatch(context.Background(), nil)
	if !errors.Is(err, transaction.ErrEmptyImport) {
		t.Fatalf("error = %v, want %v", err, transaction.ErrEmptyImport)
	}
	if got != nil {
		t.Errorf("result = %#v, want nil", got)
	}
}

func TestSaveBatchPersistsTransactions(t *testing.T) {
	repo, mock := newRepo(t)
	date := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO transactions`).
		WithArgs("2026-10-14", "Mercado", int64(12550), "EXPENSE", "ext-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "transaction_date", "description", "amount_cents", "type", "external_id",
		}).AddRow("tx-1", date, "Mercado", int64(12550), "EXPENSE", "ext-1"))
	mock.ExpectCommit()

	got, err := repo.SaveBatch(context.Background(), []transaction.Transaction{{
		Date:        date,
		Description: "Mercado",
		AmountCents: 12550,
		Type:        transaction.Expense,
		ExternalID:  "ext-1",
	}})
	if err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "tx-1" {
		t.Errorf("saved = %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveBatchBeginError(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	_, err := repo.SaveBatch(context.Background(), []transaction.Transaction{{
		Date:        time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC),
		Description: "Mercado",
		AmountCents: 100,
		Type:        transaction.Expense,
	}})
	if err == nil {
		t.Fatal("expected begin error")
	}
}

func TestSaveBatchInsertError(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO transactions`).
		WillReturnError(errors.New("constraint"))
	mock.ExpectRollback()

	_, err := repo.SaveBatch(context.Background(), []transaction.Transaction{{
		Date:        time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC),
		Description: "Mercado",
		AmountCents: 100,
		Type:        transaction.Expense,
	}})
	if err == nil {
		t.Fatal("expected insert error")
	}
}

func TestSaveBatchCommitError(t *testing.T) {
	repo, mock := newRepo(t)
	date := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO transactions`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "transaction_date", "description", "amount_cents", "type", "external_id",
		}).AddRow("tx-1", date, "Mercado", int64(100), "EXPENSE", ""))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	_, err := repo.SaveBatch(context.Background(), []transaction.Transaction{{
		Date:        date,
		Description: "Mercado",
		AmountCents: 100,
		Type:        transaction.Expense,
	}})
	if err == nil {
		t.Fatal("expected commit error")
	}
}

func TestFindAllAppliesFiltersAndDefaults(t *testing.T) {
	repo, mock := newRepo(t)
	date := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	txType := transaction.Income
	minAmount := int64(100)
	maxAmount := int64(5000)
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{
		"id", "transaction_date", "description", "amount_cents", "type", "external_id", "category", "category_status",
	}).AddRow("tx-1", date, "Salario", int64(500000), "INCOME", "ext-1", "salary", "completed")

	mock.ExpectQuery(`SELECT`).
		WithArgs(from, to, minAmount, maxAmount, "INCOME", 100, 0).
		WillReturnRows(rows)

	got, err := repo.FindAll(context.Background(), transaction.TransactionFilter{
		DateFrom:  &from,
		DateTo:    &to,
		MinAmount: &minAmount,
		MaxAmount: &maxAmount,
		Type:      &txType,
		Limit:     200,
		Offset:    -3,
	})
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d transactions, want 1", len(got))
	}
	if got[0].Category != "salary" {
		t.Errorf("category = %q", got[0].Category)
	}
	if got[0].CategoryStatus != transaction.Completed {
		t.Errorf("category status = %q", got[0].CategoryStatus)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindAllDefaultLimit(t *testing.T) {
	repo, mock := newRepo(t)

	mock.ExpectQuery(`SELECT`).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "transaction_date", "description", "amount_cents", "type", "external_id", "category", "category_status",
		}))

	got, err := repo.FindAll(context.Background(), transaction.TransactionFilter{})
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected empty slice, got nil")
	}
}

func TestFindAllQueryError(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("query failed"))

	_, err := repo.FindAll(context.Background(), transaction.TransactionFilter{Limit: 10})
	if err == nil {
		t.Fatal("expected query error")
	}
}

func TestFindAllScanError(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "transaction_date", "description", "amount_cents", "type", "external_id", "category", "category_status",
		}).AddRow("tx-1", "not-a-date", "x", int64(1), "INCOME", "", "", "pending"))

	_, err := repo.FindAll(context.Background(), transaction.TransactionFilter{Limit: 10})
	if err == nil {
		t.Fatal("expected scan error")
	}
}

func TestFindAllRowsError(t *testing.T) {
	repo, mock := newRepo(t)
	rows := sqlmock.NewRows([]string{
		"id", "transaction_date", "description", "amount_cents", "type", "external_id", "category", "category_status",
	}).AddRow("tx-1", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), "x", int64(1), "INCOME", "", "", "pending").
		RowError(0, errors.New("row error"))

	mock.ExpectQuery(`SELECT`).WillReturnRows(rows)

	_, err := repo.FindAll(context.Background(), transaction.TransactionFilter{Limit: 10})
	if err == nil {
		t.Fatal("expected rows error")
	}
}

func TestSummaryInvalidGrouping(t *testing.T) {
	repo, _ := newRepo(t)

	got, err := repo.Summary(context.Background(), transaction.TransactionSummaryFilter{
		GroupBy: transaction.SummaryGroupBy("week"),
	})
	if err == nil {
		t.Fatal("expected invalid grouping error")
	}
	if got != nil {
		t.Errorf("result = %#v, want nil", got)
	}
}

func TestSummaryDefaultGroupByAndFilters(t *testing.T) {
	repo, mock := newRepo(t)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{"date", "type", "total"}).
		AddRow(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), transaction.Income, int64(10000)).
		AddRow(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), transaction.Expense, int64(2500)).
		AddRow(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), transaction.Income, int64(5000))

	mock.ExpectQuery(`SELECT`).
		WithArgs("MONTH", from, to).
		WillReturnRows(rows)

	got, err := repo.Summary(context.Background(), transaction.TransactionSummaryFilter{
		DateFrom: &from,
		DateTo:   &to,
	})
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if got.TotalIncome != 15000 || got.TotalExpense != 2500 || got.Balance != 12500 {
		t.Errorf("totals = %+v", got)
	}
	if len(got.Series) != 2 {
		t.Fatalf("series length = %d, want 2", len(got.Series))
	}
	if got.Series[0].Date != "2026-01" {
		t.Errorf("first series date = %q", got.Series[0].Date)
	}
}

func TestSummaryGroupByDayAndYear(t *testing.T) {
	tests := []struct {
		groupBy transaction.SummaryGroupBy
		wantKey string
	}{
		{groupBy: transaction.DAY, wantKey: "2026-03-05"},
		{groupBy: transaction.YEAR, wantKey: "2026"},
	}

	for _, tt := range tests {
		t.Run(string(tt.groupBy), func(t *testing.T) {
			repo, mock := newRepo(t)
			rows := sqlmock.NewRows([]string{"date", "type", "total"}).
				AddRow(time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC), transaction.Expense, int64(1000))

			mock.ExpectQuery(`SELECT`).
				WithArgs(string(tt.groupBy)).
				WillReturnRows(rows)

			got, err := repo.Summary(context.Background(), transaction.TransactionSummaryFilter{
				GroupBy: tt.groupBy,
			})
			if err != nil {
				t.Fatalf("Summary() error = %v", err)
			}
			if len(got.Series) != 1 || got.Series[0].Date != tt.wantKey {
				t.Errorf("series = %#v, want date %q", got.Series, tt.wantKey)
			}
		})
	}
}

func TestSummaryQueryError(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("query failed"))

	_, err := repo.Summary(context.Background(), transaction.TransactionSummaryFilter{GroupBy: transaction.DAY})
	if err == nil {
		t.Fatal("expected query error")
	}
}

func TestSummaryScanError(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{"date", "type", "total"}).
			AddRow("bad-date", transaction.Income, int64(1)))

	_, err := repo.Summary(context.Background(), transaction.TransactionSummaryFilter{GroupBy: transaction.DAY})
	if err == nil {
		t.Fatal("expected scan error")
	}
}

func TestSummaryUnexpectedType(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{"date", "type", "total"}).
			AddRow(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "TRANSFER", int64(1)))

	_, err := repo.Summary(context.Background(), transaction.TransactionSummaryFilter{GroupBy: transaction.DAY})
	if err == nil {
		t.Fatal("expected unexpected type error")
	}
}

func TestSummaryRowsError(t *testing.T) {
	repo, mock := newRepo(t)
	rows := sqlmock.NewRows([]string{"date", "type", "total"}).
		AddRow(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), transaction.Income, int64(1)).
		RowError(0, errors.New("row error"))
	mock.ExpectQuery(`SELECT`).WillReturnRows(rows)

	_, err := repo.Summary(context.Background(), transaction.TransactionSummaryFilter{GroupBy: transaction.DAY})
	if err == nil {
		t.Fatal("expected rows error")
	}
}

func TestUpdateClassification(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectExec(`UPDATE transactions`).
		WithArgs("food", "completed", "tx-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateClassification(context.Background(), "tx-1", "food", 0.9, 1); err != nil {
		t.Fatalf("UpdateClassification() error = %v", err)
	}
}

func TestUpdateClassificationExecError(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectExec(`UPDATE transactions`).
		WillReturnError(errors.New("update failed"))

	err := repo.UpdateClassification(context.Background(), "tx-1", "food", 0.9, 1)
	if err == nil {
		t.Fatal("expected exec error")
	}
}

func TestUpdateClassificationNotFound(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectExec(`UPDATE transactions`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UpdateClassification(context.Background(), "tx-1", "food", 0.9, 1)
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestUpdateClassificationRowsAffectedError(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectExec(`UPDATE transactions`).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected failed")))

	err := repo.UpdateClassification(context.Background(), "tx-1", "food", 0.9, 1)
	if err == nil {
		t.Fatal("expected rows affected error")
	}
}

func TestNewTransactionRepository(t *testing.T) {
	db := &sql.DB{}
	repo := NewTransactionRepository(db)
	if repo.db != db {
		t.Error("repository was not initialized with the given db")
	}
}
