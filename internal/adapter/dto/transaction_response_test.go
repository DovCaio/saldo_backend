package dto

import (
	"errors"
	"testing"
	"time"
)

func TestNewTransactionResponse(t *testing.T) {
	date := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		type_    string
		category string
		wantType TransactionTypeResponse
		wantErr  error
	}{
		{name: "income", type_: "INCOME", category: "salary", wantType: Income},
		{name: "expense", type_: "EXPENSE", category: "food", wantType: Expense},
		{name: "invalid type", type_: "TRANSFER", wantErr: InvalidResponse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTransactionResponse(
				"tx-1",
				date,
				"Salario",
				500000,
				tt.type_,
				tt.category,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if got != nil {
					t.Errorf("response = %#v, want nil", got)
				}
				return
			}

			if got.ID != "tx-1" {
				t.Errorf("ID = %q, want tx-1", got.ID)
			}
			if !got.Date.Equal(date) {
				t.Errorf("Date = %v, want %v", got.Date, date)
			}
			if got.Description != "Salario" {
				t.Errorf("Description = %q", got.Description)
			}
			if got.AmountCents != 500000 {
				t.Errorf("AmountCents = %d", got.AmountCents)
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.Category != tt.category {
				t.Errorf("Category = %q, want %q", got.Category, tt.category)
			}
		})
	}
}
