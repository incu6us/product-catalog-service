package m_outbox

// Table and column name constants for the outbox_events table.
const (
	TableName = "outbox_events"

	ColEventID     = "event_id"
	ColEventType   = "event_type"
	ColAggregateID = "aggregate_id"
	ColPayload     = "payload"
	ColStatus      = "status"
	ColCreatedAt   = "created_at"
	ColProcessedAt = "processed_at"
)

// AllColumns returns all columns for SELECT queries.
func AllColumns() []string {
	return []string{
		ColEventID,
		ColEventType,
		ColAggregateID,
		ColPayload,
		ColStatus,
		ColCreatedAt,
		ColProcessedAt,
	}
}
