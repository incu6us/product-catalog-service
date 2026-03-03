package repo

import (
	"encoding/json"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/incu6us/commitplan"

	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
	"github.com/incu6us/product-catalog-service/internal/models/m_outbox"
)

var outboxEventNamespace = uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")

// OutboxRepo is the Spanner implementation of contracts.OutboxRepository.
type OutboxRepo struct{}

// NewOutboxRepo creates a new OutboxRepo.
func NewOutboxRepo() *OutboxRepo {
	return &OutboxRepo{}
}

// InsertMuts creates outbox insert mutations for domain events.
func (r *OutboxRepo) InsertMuts(events []domain.DomainEvent, aggregateID string, version int64) ([]*commitplan.Mutation, error) {
	muts := make([]*commitplan.Mutation, 0, len(events))

	for i, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return nil, fmt.Errorf("marshal event %s: %w", event.EventType(), err)
		}
		idempotencyKey := fmt.Sprintf("%s:%d:%d", aggregateID, version, i)
		data := &m_outbox.Data{
			EventID:     uuid.NewSHA1(outboxEventNamespace, []byte(idempotencyKey)).String(),
			EventType:   event.EventType(),
			AggregateID: aggregateID,
			Payload:     string(payload),
			Status:      "pending",
			CreatedAt:   spanner.CommitTimestamp,
		}
		muts = append(muts, &commitplan.Mutation{SpannerMut: data.InsertMut()})
	}

	return muts, nil
}
