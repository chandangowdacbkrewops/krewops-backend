package model

import "time"

// WorkCategory groups related work types (e.g. "Construction",
// "Hospitality") and mirrors the work_categories table.
type WorkCategory struct {
	ID           string    `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	DisplayOrder int32     `json:"display_order"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// WorkType mirrors a row in the work_types table, which now belongs to a
// WorkCategory and carries a stable code alongside its display name.
type WorkType struct {
	ID           string    `json:"id"`
	CategoryID   string    `json:"category_id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	DisplayOrder int32     `json:"display_order"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// WorkTypeFieldOption mirrors a row in the work_type_field_options table,
// representing one selectable option for a SELECT/MULTI_SELECT/RADIO
// dynamic field.
type WorkTypeFieldOption struct {
	ID           string    `json:"id"`
	FieldID      string    `json:"field_id"`
	FieldKey     string    `json:"field_key"`
	Value        string    `json:"value"`
	Label        string    `json:"label"`
	DisplayOrder int32     `json:"display_order"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// WorkTypeField mirrors a row in the work_type_fields table, describing one
// dynamic form field configured for a work type, along with its options
// (populated only for SELECT/MULTI_SELECT/RADIO field types).
type WorkTypeField struct {
	ID           string                `json:"id"`
	WorkTypeID   string                `json:"work_type_id"`
	FieldKey     string                `json:"field_key"`
	Label        string                `json:"label"`
	FieldType    string                `json:"field_type"`
	Placeholder  *string               `json:"placeholder,omitempty"`
	HelpText     *string               `json:"help_text,omitempty"`
	IsRequired   bool                  `json:"is_required"`
	Unit         *string               `json:"unit,omitempty"`
	MinValue     *float64              `json:"min_value,omitempty"`
	MaxValue     *float64              `json:"max_value,omitempty"`
	MinLength    *int32                `json:"min_length,omitempty"`
	MaxLength    *int32                `json:"max_length,omitempty"`
	DisplayOrder int32                 `json:"display_order"`
	IsActive     bool                  `json:"is_active"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
	Options      []WorkTypeFieldOption `json:"options,omitempty"`
}

type PaymentType struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}
