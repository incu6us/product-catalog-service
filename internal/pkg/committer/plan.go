package committer

import (
	"errors"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/incu6us/commitplan"
)

// ErrConcurrentModification is returned when a DML statement affects 0 rows,
// indicating that the row was modified by another transaction.
var ErrConcurrentModification = errors.New("concurrent modification detected")

// DMLStatement represents a parameterized DML statement.
type DMLStatement struct {
	SQL    string
	Params map[string]interface{}
}

// Plan collects mutations and DML statements for atomic application.
type Plan struct {
	mutations []*spanner.Mutation
	dmls      []DMLStatement
}

// NewPlan creates a new empty Plan.
func NewPlan() *Plan {
	return &Plan{}
}

// AddMutation adds a commitplan mutation to the plan.
func (p *Plan) AddMutation(mut *commitplan.Mutation) error {
	if mut == nil || mut.SpannerMut == nil {
		return nil
	}
	sm, ok := mut.SpannerMut.(*spanner.Mutation)
	if !ok {
		return fmt.Errorf("unsupported mutation type %T, expected *spanner.Mutation", mut.SpannerMut)
	}
	p.mutations = append(p.mutations, sm)
	return nil
}

// AddMutations adds multiple commitplan mutations to the plan.
func (p *Plan) AddMutations(muts []*commitplan.Mutation) error {
	for _, m := range muts {
		if err := p.AddMutation(m); err != nil {
			return err
		}
	}
	return nil
}

// AddDML adds a DML statement to the plan.
func (p *Plan) AddDML(stmt DMLStatement) {
	p.dmls = append(p.dmls, stmt)
}

// IsEmpty returns true if the plan has no mutations or DML statements.
func (p *Plan) IsEmpty() bool {
	return len(p.mutations) == 0 && len(p.dmls) == 0
}

// Mutations returns all spanner mutations in the plan.
func (p *Plan) Mutations() []*spanner.Mutation {
	return p.mutations
}

// DMLs returns all DML statements in the plan.
func (p *Plan) DMLs() []DMLStatement {
	return p.dmls
}
