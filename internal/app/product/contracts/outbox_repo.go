package contracts

import (
	"github.com/incu6us/commitplan"

	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
)

// OutboxRepository defines the interface for outbox event persistence.
type OutboxRepository interface {
	InsertMuts(events []domain.DomainEvent, aggregateID string, version int64) ([]*commitplan.Mutation, error)
}
