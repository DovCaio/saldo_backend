package transaction_classification

import (
	"context"

	"github.com/DovCaio/saldo/internal/adapter/rabbitmq"
	"github.com/DovCaio/saldo/internal/application"
)



type ClassificationUseCase struct {
    repository application.ClassificationRepository
}

func NewClassificationUseCase(
    repository application.ClassificationRepository,
) *ClassificationUseCase {
    return &ClassificationUseCase{
        repository: repository,
    }
}

func (uc *ClassificationUseCase) Execute(
    ctx context.Context,
    result rabbitmq.ClassificationResult,
) error {
    return uc.repository.UpdateClassification(
        ctx,
        result.TransactionID,
        result.Category,
        result.Confidence,
        result.DescriptionVersion,
    )
}