package rabbitmq

import (
	"testing"

	"github.com/DovCaio/saldo/internal/transaction"
)

func TestFromTransaction(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, err := FromTransaction(transaction.Transaction{
			ID:          "tx-1",
			Description: "Mercado",
		})
		if err != nil {
			t.Fatalf("FromTransaction() error = %v", err)
		}
		if got.TransactionID != "tx-1" {
			t.Errorf("TransactionID = %q", got.TransactionID)
		}
		if got.Description != "Mercado" {
			t.Errorf("Description = %q", got.Description)
		}
		if got.EventID == "" {
			t.Error("EventID is empty")
		}
		if got.DescriptionVersion != 1 {
			t.Errorf("DescriptionVersion = %d, want 1", got.DescriptionVersion)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		got, err := FromTransaction(transaction.Transaction{Description: "Mercado"})
		if err == nil {
			t.Fatal("expected error")
		}
		if got != nil {
			t.Errorf("result = %#v, want nil", got)
		}
	})

	t.Run("missing description", func(t *testing.T) {
		got, err := FromTransaction(transaction.Transaction{ID: "tx-1"})
		if err == nil {
			t.Fatal("expected error")
		}
		if got != nil {
			t.Errorf("result = %#v, want nil", got)
		}
	})
}
