package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/DovCaio/saldo/internal/adapter/dto"
	"github.com/DovCaio/saldo/internal/transaction"
)

type transactionSummaryMock struct {
	gotContext context.Context
	gotFilter  transaction.TransactionFilter
	err        error
	called     bool
	result     *dto.SummaryResponse
}

func (m *transactionSummaryMock) ImportTransactions(ctx context.Context, transactions []transaction.Transaction) ([]transaction.Transaction, error) {
	return nil, nil
}

func (m *transactionSummaryMock) FindAll(
	ctx context.Context,
	filter transaction.TransactionFilter,
) ([]dto.TransactionResponse, error) {
	return nil, nil
}

func (m *transactionSummaryMock) Summary(
	ctx context.Context,
	filter transaction.TransactionSummaryFilter,
) (*dto.SummaryResponse, error) {
	if m.err != nil {
		return nil, m.err
	}

	m.called = true

	return m.result, nil
}

func TestSucessfulSummary(t *testing.T) {
	expectedSummary := &dto.SummaryResponse{
		TotalIncome:  1000,
		TotalExpense: 800,
		Balance:      200,
		Series: []dto.Serie{
			{Date: "2026-10-02", Income: 1000, Expense: 800},
		},
	}

	mock := &transactionSummaryMock{
		gotContext: context.Background(),
		called:     false,
		result:     expectedSummary,
	}

	summaryTransaction := NewSummaryTransactions(mock)

	dateFrom := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)

	filter := transaction.TransactionSummaryFilter{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
		GroupBy:  "Day",
	}

	result, err := summaryTransaction.Execute(mock.gotContext, filter)

	if mock.gotContext != context.Background() {
		t.Errorf("the context was not propageded")
	}

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !mock.called {
		t.Errorf("service was not called")
	}

	if !reflect.DeepEqual(expectedSummary, result) {
		t.Errorf("expected %#v, got %#v", expectedSummary, result)
	}
}

func TestSummaryPropagatesError(t *testing.T) {
	expected := errors.New("database unavailable")
	mock := &transactionSummaryMock{err: expected}
	useCase := NewSummaryTransactions(mock)

	got, err := useCase.Execute(context.Background(), transaction.TransactionSummaryFilter{})
	if !errors.Is(err, expected) {
		t.Fatalf("error = %v, want %v", err, expected)
	}
	if got != nil {
		t.Errorf("result = %#v, want nil", got)
	}
}
