package domain

import "time"

// ProductStatus represents the lifecycle state of a product.
type ProductStatus string

const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
	ProductStatusArchived ProductStatus = "archived"
)

// Field constants for change tracking.
const (
	FieldName        = "name"
	FieldDescription = "description"
	FieldCategory    = "category"
	FieldDiscount    = "discount"
	FieldStatus      = "status"
	FieldArchivedAt  = "archived_at"
	FieldVersion     = "version"
)

// ChangeTracker tracks which fields have been modified.
type ChangeTracker struct {
	dirtyFields map[string]bool
}

// NewChangeTracker creates a new ChangeTracker.
func NewChangeTracker() *ChangeTracker {
	return &ChangeTracker{dirtyFields: make(map[string]bool)}
}

// MarkDirty marks a field as modified.
func (ct *ChangeTracker) MarkDirty(field string) {
	ct.dirtyFields[field] = true
}

// Dirty returns true if the field has been modified.
func (ct *ChangeTracker) Dirty(field string) bool {
	return ct.dirtyFields[field]
}

// HasChanges returns true if any field has been modified.
func (ct *ChangeTracker) HasChanges() bool {
	return len(ct.dirtyFields) > 0
}

// DirtyCount returns the number of dirty fields.
func (ct *ChangeTracker) DirtyCount() int {
	return len(ct.dirtyFields)
}

// Product is the aggregate root for product management.
type Product struct {
	id          string
	name        string
	description string
	category    string
	basePrice   *Money
	discount    *Discount
	status      ProductStatus
	version     int64
	createdAt   time.Time
	updatedAt   time.Time
	archivedAt  *time.Time

	changes *ChangeTracker
	events  []DomainEvent
}

// NewProduct creates a new Product in draft status.
func NewProduct(id, name, description, category string, basePrice *Money, now time.Time) (*Product, error) {
	if name == "" {
		return nil, ErrInvalidProductName
	}
	if category == "" {
		return nil, ErrInvalidCategory
	}
	if basePrice == nil || basePrice.IsNegative() || basePrice.IsZero() {
		return nil, ErrInvalidPrice
	}

	p := &Product{
		id:          id,
		name:        name,
		description: description,
		category:    category,
		basePrice:   basePrice,
		status:      ProductStatusDraft,
		version:     1,
		createdAt:   now,
		updatedAt:   now,
		changes:     NewChangeTracker(),
		events:      nil,
	}

	p.events = append(p.events, &ProductCreatedEvent{
		ProductID:  id,
		Name:       name,
		Category:   category,
		BasePrice:  basePrice,
		OccurredAt: now,
	})

	return p, nil
}

// ReconstructProduct creates a Product from persisted data (no validation, no events).
func ReconstructProduct(
	id, name, description, category string,
	basePrice *Money,
	discount *Discount,
	status ProductStatus,
	version int64,
	createdAt, updatedAt time.Time,
	archivedAt *time.Time,
) *Product {
	return &Product{
		id:          id,
		name:        name,
		description: description,
		category:    category,
		basePrice:   basePrice,
		discount:    discount,
		status:      status,
		version:     version,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		archivedAt:  archivedAt,
		changes:     NewChangeTracker(),
	}
}

// UpdateDetails updates the product name, description, and category.
func (p *Product) UpdateDetails(name, description, category string, now time.Time) error {
	if p.status == ProductStatusArchived {
		return ErrProductAlreadyArchived
	}
	if name == "" {
		return ErrInvalidProductName
	}
	if category == "" {
		return ErrInvalidCategory
	}

	var updatedFields []string

	if p.name != name {
		p.name = name
		p.changes.MarkDirty(FieldName)
		updatedFields = append(updatedFields, FieldName)
	}
	if p.description != description {
		p.description = description
		p.changes.MarkDirty(FieldDescription)
		updatedFields = append(updatedFields, FieldDescription)
	}
	if p.category != category {
		p.category = category
		p.changes.MarkDirty(FieldCategory)
		updatedFields = append(updatedFields, FieldCategory)
	}

	if len(updatedFields) > 0 {
		p.updatedAt = now
		p.version++
		p.changes.MarkDirty(FieldVersion)
		p.events = append(p.events, &ProductUpdatedEvent{
			ProductID:     p.id,
			UpdatedFields: updatedFields,
			OccurredAt:    now,
		})
	}

	return nil
}

// Activate transitions the product to active status.
func (p *Product) Activate(now time.Time) error {
	if p.status == ProductStatusActive {
		return ErrProductAlreadyActive
	}
	if p.status == ProductStatusArchived {
		return ErrProductAlreadyArchived
	}

	p.status = ProductStatusActive
	p.updatedAt = now
	p.version++
	p.changes.MarkDirty(FieldStatus)
	p.changes.MarkDirty(FieldVersion)

	p.events = append(p.events, &ProductActivatedEvent{
		ProductID:  p.id,
		OccurredAt: now,
	})

	return nil
}

// Deactivate transitions the product to inactive status.
func (p *Product) Deactivate(now time.Time) error {
	if p.status != ProductStatusActive {
		return ErrProductNotActive
	}

	p.status = ProductStatusInactive
	p.updatedAt = now
	p.version++
	p.changes.MarkDirty(FieldStatus)
	p.changes.MarkDirty(FieldVersion)

	p.events = append(p.events, &ProductDeactivatedEvent{
		ProductID:  p.id,
		OccurredAt: now,
	})

	return nil
}

// ApplyDiscount applies a percentage discount to the product.
func (p *Product) ApplyDiscount(discount *Discount, now time.Time) error {
	if p.status != ProductStatusActive {
		return ErrProductNotActive
	}
	if !discount.IsValidAt(now) {
		return ErrInvalidDiscountPeriod
	}
	if p.discount != nil && p.discount.IsActive(now) {
		return ErrProductHasActiveDiscount
	}

	pct, err := discount.SafePercentageInt64()
	if err != nil {
		return err
	}

	p.discount = discount
	p.updatedAt = now
	p.version++
	p.changes.MarkDirty(FieldDiscount)
	p.changes.MarkDirty(FieldVersion)

	p.events = append(p.events, &DiscountAppliedEvent{
		ProductID:  p.id,
		Percentage: pct,
		StartDate:  discount.StartDate(),
		EndDate:    discount.EndDate(),
		OccurredAt: now,
	})

	return nil
}

// RemoveDiscount removes the active discount from the product.
func (p *Product) RemoveDiscount(now time.Time) error {
	if p.status != ProductStatusActive {
		return ErrProductNotActive
	}
	if p.discount == nil {
		return ErrNoDiscountToRemove
	}

	p.discount = nil
	p.updatedAt = now
	p.version++
	p.changes.MarkDirty(FieldDiscount)
	p.changes.MarkDirty(FieldVersion)

	p.events = append(p.events, &DiscountRemovedEvent{
		ProductID:  p.id,
		OccurredAt: now,
	})

	return nil
}

// Archive soft-deletes the product.
func (p *Product) Archive(now time.Time) error {
	if p.status == ProductStatusArchived {
		return ErrProductAlreadyArchived
	}

	p.status = ProductStatusArchived
	p.archivedAt = &now
	p.updatedAt = now
	p.version++
	p.changes.MarkDirty(FieldStatus)
	p.changes.MarkDirty(FieldArchivedAt)
	p.changes.MarkDirty(FieldVersion)

	p.events = append(p.events, &ProductArchivedEvent{
		ProductID:  p.id,
		OccurredAt: now,
	})

	return nil
}

// Getters

func (p *Product) ID() string              { return p.id }
func (p *Product) Name() string            { return p.name }
func (p *Product) Description() string     { return p.description }
func (p *Product) Category() string        { return p.category }
func (p *Product) BasePrice() *Money       { return p.basePrice }
func (p *Product) Discount() *Discount     { return p.discount }
func (p *Product) Status() ProductStatus   { return p.status }
func (p *Product) Version() int64          { return p.version }
func (p *Product) CreatedAt() time.Time    { return p.createdAt }
func (p *Product) UpdatedAt() time.Time    { return p.updatedAt }
func (p *Product) ArchivedAt() *time.Time  { return p.archivedAt }
func (p *Product) Changes() *ChangeTracker { return p.changes }

// DomainEvents returns all domain events captured by this aggregate.
// Returns a copy to prevent callers from mutating the aggregate's internal slice.
func (p *Product) DomainEvents() []DomainEvent {
	return append([]DomainEvent{}, p.events...)
}

// ClearEvents clears all domain events.
func (p *Product) ClearEvents() {
	p.events = nil
}
