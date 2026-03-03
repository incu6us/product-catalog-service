package m_outbox

import (
	"time"

	"cloud.google.com/go/spanner"
)

// Data represents a row in the outbox_events table.
type Data struct {
	EventID     string
	EventType   string
	AggregateID string
	Payload     string
	Status      string
	CreatedAt   time.Time
	ProcessedAt *time.Time
}

// InsertMut creates a Spanner insert mutation for this outbox event.
func (d *Data) InsertMut() *spanner.Mutation {
	cols := AllColumns()
	vals := []interface{}{
		d.EventID,
		d.EventType,
		d.AggregateID,
		d.Payload,
		d.Status,
		d.CreatedAt,
		d.ProcessedAt,
	}
	return spanner.Insert(TableName, cols, vals)
}
