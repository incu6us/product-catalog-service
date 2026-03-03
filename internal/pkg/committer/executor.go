package committer

import (
	"context"

	"cloud.google.com/go/spanner"
)

// Executor applies a Plan atomically using Spanner.
type Executor struct {
	client *spanner.Client
}

// NewExecutor creates a new Executor.
func NewExecutor(client *spanner.Client) *Executor {
	return &Executor{client: client}
}

// Apply executes the plan atomically.
// If the plan contains DML statements, a read-write transaction is used.
// DML statements that affect 0 rows cause ErrConcurrentModification.
func (e *Executor) Apply(ctx context.Context, plan *Plan) error {
	if plan.IsEmpty() {
		return nil
	}

	if len(plan.dmls) == 0 {
		_, err := e.client.Apply(ctx, plan.mutations)
		return err
	}

	_, err := e.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		for _, dml := range plan.dmls {
			stmt := spanner.Statement{SQL: dml.SQL, Params: dml.Params}
			rowCount, err := txn.Update(ctx, stmt)
			if err != nil {
				return err
			}
			if rowCount == 0 {
				return ErrConcurrentModification
			}
		}
		return txn.BufferWrite(plan.mutations)
	})
	return err
}
