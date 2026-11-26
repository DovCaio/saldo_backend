package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/DovCaio/saldo/internal/adapter/dto"
	"github.com/DovCaio/saldo/internal/transaction"
)

type listTransactionsUseCaseMock struct {
	transactions []dto.TransactionResponse
	err          error
	gotFilter    transaction.TransactionFilter
	called       bool
}

func (m *listTransactionsUseCaseMock) Execute(
	ctx context.Context,
	filter transaction.TransactionFilter,
) ([]dto.TransactionResponse, error) {
	m.called = true
	m.gotFilter = filter
	return m.transactions, m.err
}

func TestListTransactionsHandler_Success(t *testing.T) {
	expected := []dto.TransactionResponse{
		{
			ID:          "transaction-1",
			Date:        time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC),
			Description: "Salário",
			AmountCents: 500000,
			Type:        dto.Income,
		},
	}

	mock := &listTransactionsUseCaseMock{
		transactions: expected,
	}

	handler := NewListTransactionsHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected application/json, got %q", contentType)
	}

	var got []dto.TransactionResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %+v, got %+v", expected, got)
	}

	if !mock.called {
		t.Fatal("expected use case to be called")
	}
}

func TestListTransactionsHandler_ParsesFilters(t *testing.T) {
	mock := &listTransactionsUseCaseMock{}
	handler := NewListTransactionsHandler(mock)

	req := httptest.NewRequest(
		http.MethodGet,
		"/transactions?type=EXPENSE&min_amount=1000&max_amount=5000&date_from=2026-10-01&date_to=2026-10-14&limit=10&offset=5",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	filter := mock.gotFilter

	if filter.Type == nil || *filter.Type != transaction.Expense {
		t.Errorf("expected type EXPENSE, got %v", filter.Type)
	}

	if filter.MinAmount == nil || *filter.MinAmount != 1000 {
		t.Errorf("expected minAmount 1000, got %v", filter.MinAmount)
	}

	if filter.MaxAmount == nil || *filter.MaxAmount != 5000 {
		t.Errorf("expected maxAmount 5000, got %v", filter.MaxAmount)
	}

	if filter.DateFrom == nil || filter.DateFrom.Format("2006-01-02") != "2026-10-01" {
		t.Errorf("unexpected dateFrom: %v", filter.DateFrom)
	}

	if filter.DateTo == nil || filter.DateTo.Format("2006-01-02") != "2026-10-14" {
		t.Errorf("unexpected dateTo: %v", filter.DateTo)
	}

	if filter.Limit != 10 {
		t.Errorf("expected limit 10, got %d", filter.Limit)
	}

	if filter.Offset != 5 {
		t.Errorf("expected offset 5, got %d", filter.Offset)
	}
}

func TestListTransactionsHandler_InvalidType(t *testing.T) {
	mock := &listTransactionsUseCaseMock{}
	handler := NewListTransactionsHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/transactions?type=UNKNOWN", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	if mock.called {
		t.Error("use case should not be called for invalid filters")
	}
}

func TestListTransactionsHandler_InvalidAmount(t *testing.T) {
	mock := &listTransactionsUseCaseMock{}
	handler := NewListTransactionsHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/transactions?min_amount=abc", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	if mock.called {
		t.Error("use case should not be called for invalid filters")
	}
}

func TestListTransactionsHandler_InvalidDateRange(t *testing.T) {
	mock := &listTransactionsUseCaseMock{}
	handler := NewListTransactionsHandler(mock)

	req := httptest.NewRequest(
		http.MethodGet,
		"/transactions?date_from=2026-10-15&date_to=2026-10-01",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	if mock.called {
		t.Error("use case should not be called for invalid filters")
	}
}

func TestListTransactionsHandler_InvalidAmountRange(t *testing.T) {
	mock := &listTransactionsUseCaseMock{}
	handler := NewListTransactionsHandler(mock)

	req := httptest.NewRequest(
		http.MethodGet,
		"/transactions?min_amount=5000&max_amount=1000",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	if mock.called {
		t.Error("use case should not be called for invalid filters")
	}
}

func TestListTransactionsHandler_UseCaseError(t *testing.T) {
	mock := &listTransactionsUseCaseMock{
		err: errors.New("database unavailable"),
	}
	handler := NewListTransactionsHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestListTransactionsHandler_InvalidQueryParams(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{name: "invalid date_from", query: "?date_from=10-01-2026"},
		{name: "invalid date_to", query: "?date_to=not-a-date"},
		{name: "invalid max_amount", query: "?max_amount=abc"},
		{name: "negative min_amount", query: "?min_amount=-1"},
		{name: "negative max_amount", query: "?max_amount=-10"},
		{name: "invalid limit", query: "?limit=abc"},
		{name: "zero limit", query: "?limit=0"},
		{name: "invalid offset", query: "?offset=abc"},
		{name: "negative offset", query: "?offset=-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &listTransactionsUseCaseMock{}
			handler := NewListTransactionsHandler(mock)

			req := httptest.NewRequest(http.MethodGet, "/transactions"+tt.query, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			if mock.called {
				t.Error("use case should not be called")
			}
		})
	}
}

func TestListTransactionsHandler_ParsesIncomeType(t *testing.T) {
	mock := &listTransactionsUseCaseMock{}
	handler := NewListTransactionsHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/transactions?type=INCOME", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if mock.gotFilter.Type == nil || *mock.gotFilter.Type != transaction.Income {
		t.Errorf("type = %v, want INCOME", mock.gotFilter.Type)
	}
}

func TestListTransactionsHandler_EmptyResult(t *testing.T) {
	mock := &listTransactionsUseCaseMock{
		transactions: []dto.TransactionResponse{},
	}
	handler := NewListTransactionsHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var got []dto.TransactionResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty result, got %d transactions", len(got))
	}
}
