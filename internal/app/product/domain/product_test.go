package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/incu6us/product-catalog-service/internal/app/product/domain"
)

var now = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

func validPrice(t *testing.T) *domain.Money {
	t.Helper()
	m, err := domain.NewMoney(1999, 100)
	require.NoError(t, err)
	return m
}

func TestNewProduct_Valid(t *testing.T) {
	p, err := domain.NewProduct("id-1", "Widget", "A widget", "electronics", validPrice(t), now)
	require.NoError(t, err)
	assert.Equal(t, "id-1", p.ID())
	assert.Equal(t, "Widget", p.Name())
	assert.Equal(t, "A widget", p.Description())
	assert.Equal(t, "electronics", p.Category())
	assert.Equal(t, domain.ProductStatusDraft, p.Status())
	assert.Equal(t, int64(1), p.Version())
	assert.Len(t, p.DomainEvents(), 1)
}

func TestNewProduct_EmptyName(t *testing.T) {
	_, err := domain.NewProduct("id-1", "", "desc", "cat", validPrice(t), now)
	assert.ErrorIs(t, err, domain.ErrInvalidProductName)
}

func TestNewProduct_EmptyCategory(t *testing.T) {
	_, err := domain.NewProduct("id-1", "Widget", "desc", "", validPrice(t), now)
	assert.ErrorIs(t, err, domain.ErrInvalidCategory)
}

func TestUpdateDetails_EmptyCategory(t *testing.T) {
	p := domain.ReconstructProduct(
		"id-1", "Old", "desc", "cat", validPrice(t), nil,
		domain.ProductStatusDraft, 1, now, now, nil,
	)
	err := p.UpdateDetails("New", "desc", "", now.Add(time.Hour))
	assert.ErrorIs(t, err, domain.ErrInvalidCategory)
}

func TestNewProduct_NilPrice(t *testing.T) {
	_, err := domain.NewProduct("id-1", "Widget", "desc", "cat", nil, now)
	assert.ErrorIs(t, err, domain.ErrInvalidPrice)
}

func TestNewProduct_ZeroPrice(t *testing.T) {
	zero, err := domain.NewMoney(0, 1)
	require.NoError(t, err)
	_, err = domain.NewProduct("id-1", "Widget", "desc", "cat", zero, now)
	assert.ErrorIs(t, err, domain.ErrInvalidPrice)
}

func TestNewProduct_NegativePrice(t *testing.T) {
	neg, err := domain.NewMoney(-100, 1)
	require.NoError(t, err)
	_, err = domain.NewProduct("id-1", "Widget", "desc", "cat", neg, now)
	assert.ErrorIs(t, err, domain.ErrInvalidPrice)
}

func TestActivate_FromDraft(t *testing.T) {
	p, _ := domain.NewProduct("id-1", "Widget", "", "cat", validPrice(t), now)
	err := p.Activate(now)
	require.NoError(t, err)
	assert.Equal(t, domain.ProductStatusActive, p.Status())
	assert.Equal(t, int64(2), p.Version())
	assert.True(t, p.Changes().Dirty(domain.FieldStatus))
	assert.True(t, p.Changes().Dirty(domain.FieldVersion))
}

func TestActivate_AlreadyActive(t *testing.T) {
	p, _ := domain.NewProduct("id-1", "Widget", "", "cat", validPrice(t), now)
	_ = p.Activate(now)
	err := p.Activate(now)
	assert.ErrorIs(t, err, domain.ErrProductAlreadyActive)
}

func TestActivate_FromArchived(t *testing.T) {
	p, _ := domain.NewProduct("id-1", "Widget", "", "cat", validPrice(t), now)
	_ = p.Archive(now)
	err := p.Activate(now)
	assert.ErrorIs(t, err, domain.ErrProductAlreadyArchived)
}

func TestDeactivate_FromActive(t *testing.T) {
	p, _ := domain.NewProduct("id-1", "Widget", "", "cat", validPrice(t), now)
	_ = p.Activate(now)
	err := p.Deactivate(now)
	require.NoError(t, err)
	assert.Equal(t, domain.ProductStatusInactive, p.Status())
	assert.Equal(t, int64(3), p.Version())
}

func TestDeactivate_FromDraft(t *testing.T) {
	p, _ := domain.NewProduct("id-1", "Widget", "", "cat", validPrice(t), now)
	err := p.Deactivate(now)
	assert.ErrorIs(t, err, domain.ErrProductNotActive)
}

func TestArchive_FromDraft(t *testing.T) {
	p, _ := domain.NewProduct("id-1", "Widget", "", "cat", validPrice(t), now)
	err := p.Archive(now)
	require.NoError(t, err)
	assert.Equal(t, domain.ProductStatusArchived, p.Status())
	assert.NotNil(t, p.ArchivedAt())
	assert.Equal(t, int64(2), p.Version())
}

func TestArchive_FromActive(t *testing.T) {
	p, _ := domain.NewProduct("id-1", "Widget", "", "cat", validPrice(t), now)
	_ = p.Activate(now)
	err := p.Archive(now)
	require.NoError(t, err)
	assert.Equal(t, domain.ProductStatusArchived, p.Status())
}

func TestArchive_AlreadyArchived(t *testing.T) {
	p, _ := domain.NewProduct("id-1", "Widget", "", "cat", validPrice(t), now)
	_ = p.Archive(now)
	err := p.Archive(now)
	assert.ErrorIs(t, err, domain.ErrProductAlreadyArchived)
}

func TestUpdateDetails_ChangesTracked(t *testing.T) {
	p := domain.ReconstructProduct(
		"id-1", "Old", "Old desc", "cat", validPrice(t), nil,
		domain.ProductStatusDraft, 1, now, now, nil,
	)
	err := p.UpdateDetails("New", "New desc", "cat", now.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, "New", p.Name())
	assert.Equal(t, "New desc", p.Description())
	assert.True(t, p.Changes().Dirty(domain.FieldName))
	assert.True(t, p.Changes().Dirty(domain.FieldDescription))
	assert.False(t, p.Changes().Dirty(domain.FieldCategory))
	assert.True(t, p.Changes().Dirty(domain.FieldVersion))
	assert.Equal(t, int64(2), p.Version())
	require.Len(t, p.DomainEvents(), 1)

	evt, ok := p.DomainEvents()[0].(*domain.ProductUpdatedEvent)
	require.True(t, ok)
	assert.ElementsMatch(t, []string{domain.FieldName, domain.FieldDescription}, evt.UpdatedFields)
}

func TestUpdateDetails_NoChanges(t *testing.T) {
	p := domain.ReconstructProduct(
		"id-1", "Same", "desc", "cat", validPrice(t), nil,
		domain.ProductStatusDraft, 1, now, now, nil,
	)
	err := p.UpdateDetails("Same", "desc", "cat", now.Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, p.Changes().HasChanges())
	assert.Equal(t, int64(1), p.Version())
	assert.Len(t, p.DomainEvents(), 0)
}

func TestUpdateDetails_Archived(t *testing.T) {
	p := domain.ReconstructProduct(
		"id-1", "Old", "desc", "cat", validPrice(t), nil,
		domain.ProductStatusArchived, 1, now, now, &now,
	)
	err := p.UpdateDetails("New", "desc", "cat", now)
	assert.ErrorIs(t, err, domain.ErrProductAlreadyArchived)
}

func TestApplyDiscount(t *testing.T) {
	p := domain.ReconstructProduct(
		"id-1", "Widget", "", "cat", validPrice(t), nil,
		domain.ProductStatusActive, 1, now, now, nil,
	)
	d, err := domain.NewDiscount(20, now.Add(-time.Hour), now.Add(24*time.Hour))
	require.NoError(t, err)

	err = p.ApplyDiscount(d, now)
	require.NoError(t, err)
	assert.NotNil(t, p.Discount())
	assert.Equal(t, int64(2), p.Version())
	assert.True(t, p.Changes().Dirty(domain.FieldDiscount))
}

func TestApplyDiscount_NotActive(t *testing.T) {
	p := domain.ReconstructProduct(
		"id-1", "Widget", "", "cat", validPrice(t), nil,
		domain.ProductStatusDraft, 1, now, now, nil,
	)
	d, _ := domain.NewDiscount(20, now.Add(-time.Hour), now.Add(24*time.Hour))
	err := p.ApplyDiscount(d, now)
	assert.ErrorIs(t, err, domain.ErrProductNotActive)
}

func TestApplyDiscount_HasActiveDiscount(t *testing.T) {
	existing, _ := domain.NewDiscount(10, now.Add(-time.Hour), now.Add(24*time.Hour))
	p := domain.ReconstructProduct(
		"id-1", "Widget", "", "cat", validPrice(t), existing,
		domain.ProductStatusActive, 1, now, now, nil,
	)
	newDiscount, _ := domain.NewDiscount(20, now.Add(-time.Hour), now.Add(24*time.Hour))
	err := p.ApplyDiscount(newDiscount, now)
	assert.ErrorIs(t, err, domain.ErrProductHasActiveDiscount)
}

func TestRemoveDiscount(t *testing.T) {
	d, _ := domain.NewDiscount(20, now.Add(-time.Hour), now.Add(24*time.Hour))
	p := domain.ReconstructProduct(
		"id-1", "Widget", "", "cat", validPrice(t), d,
		domain.ProductStatusActive, 1, now, now, nil,
	)
	err := p.RemoveDiscount(now)
	require.NoError(t, err)
	assert.Nil(t, p.Discount())
	assert.Equal(t, int64(2), p.Version())
}

func TestRemoveDiscount_NoDiscount(t *testing.T) {
	p := domain.ReconstructProduct(
		"id-1", "Widget", "", "cat", validPrice(t), nil,
		domain.ProductStatusActive, 1, now, now, nil,
	)
	err := p.RemoveDiscount(now)
	assert.ErrorIs(t, err, domain.ErrNoDiscountToRemove)
}

func TestRemoveDiscount_NotActive(t *testing.T) {
	p := domain.ReconstructProduct(
		"id-1", "Widget", "", "cat", validPrice(t), nil,
		domain.ProductStatusDraft, 1, now, now, nil,
	)
	err := p.RemoveDiscount(now)
	assert.ErrorIs(t, err, domain.ErrProductNotActive)
}

func TestVersionIncrements(t *testing.T) {
	p, _ := domain.NewProduct("id-1", "Widget", "", "cat", validPrice(t), now)
	assert.Equal(t, int64(1), p.Version())

	_ = p.Activate(now)
	assert.Equal(t, int64(2), p.Version())

	_ = p.Deactivate(now)
	assert.Equal(t, int64(3), p.Version())
}

func TestClearEvents(t *testing.T) {
	p, _ := domain.NewProduct("id-1", "Widget", "", "cat", validPrice(t), now)
	assert.Len(t, p.DomainEvents(), 1)
	p.ClearEvents()
	assert.Empty(t, p.DomainEvents())
}

func TestDomainEvents_ReturnsCopy(t *testing.T) {
	p, _ := domain.NewProduct("id-1", "Widget", "", "cat", validPrice(t), now)
	events := p.DomainEvents()
	require.Len(t, events, 1)

	// Mutate the returned slice
	events[0] = nil

	// Original aggregate events must be unaffected
	assert.Len(t, p.DomainEvents(), 1)
	assert.NotNil(t, p.DomainEvents()[0])
}
