package transaction_classification

import (
	"context"
	"errors"
	"testing"

	"github.com/DovCaio/saldo/internal/adapter/rabbitmq"
)

type fakeClassificationRepository struct {
	transactionID       string
	category            string
	confidence          float64
	descriptionVersion  int64
	err                 error
	called              bool
}

func (f *fakeClassificationRepository) UpdateClassification(
	ctx context.Context,
	transactionID string,
	category string,
	confidence float64,
	descriptionVersion int64,
) error {
	f.called = true
	f.transactionID = transactionID
	f.category = category
	f.confidence = confidence
	f.descriptionVersion = descriptionVersion
	return f.err
}

func TestClassificationUseCaseExecute(t *testing.T) {
	repo := &fakeClassificationRepository{}
	useCase := NewClassificationUseCase(repo)

	result := rabbitmq.ClassificationResult{
		TransactionID:      "tx-1",
		EventID:            "evt-1",
		Category:           "food",
		Confidence:         0.91,
		DescriptionVersion: 2,
	}

	if err := useCase.Execute(context.Background(), result); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !repo.called {
		t.Fatal("expected repository to be called")
	}
	if repo.transactionID != result.TransactionID {
		t.Errorf("transactionID = %q", repo.transactionID)
	}
	if repo.category != result.Category {
		t.Errorf("category = %q", repo.category)
	}
	if repo.confidence != result.Confidence {
		t.Errorf("confidence = %v", repo.confidence)
	}
	if repo.descriptionVersion != result.DescriptionVersion {
		t.Errorf("descriptionVersion = %d", repo.descriptionVersion)
	}
}

func TestClassificationUseCasePropagatesError(t *testing.T) {
	expected := errors.New("not found")
	repo := &fakeClassificationRepository{err: expected}
	useCase := NewClassificationUseCase(repo)

	err := useCase.Execute(context.Background(), rabbitmq.ClassificationResult{
		TransactionID: "tx-1",
		Category:      "food",
	})
	if !errors.Is(err, expected) {
		t.Fatalf("error = %v, want %v", err, expected)
	}
}
