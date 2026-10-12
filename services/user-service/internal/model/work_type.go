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

// WorkSegment sits between a category and its work types (e.g. Coffee
// under Agriculture).
type WorkSegment struct {
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

// WorkType mirrors a row in the work_types table, which now belongs to a
// WorkSegment (and denormalizes the parent category_id).
type WorkType struct {
	ID           string    `json:"id"`
	CategoryID   string    `json:"category_id"`
	SegmentID    string    `json:"segment_id"`
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
// Field context values: WORKER_PROFILE fields are captured once on the
// worker's work-type selection, WORK_POST fields are captured per job
// posting (legacy default), BOTH applies to either context.
const (
	FieldContextWorkerProfile = "WORKER_PROFILE"
	FieldContextWorkPost      = "WORK_POST"
	FieldContextBoth          = "BOTH"
)

type WorkTypeField struct {
	ID           string                `json:"id"`
	WorkTypeID   string                `json:"work_type_id"`
	FieldKey     string                `json:"field_key"`
	Label        string                `json:"label"`
	FieldType    string                `json:"field_type"`
	Context      string                `json:"context"`
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

// Skill is global, reusable master data (no longer tied 1:1 to a single
// work type); its applicability to a work type is expressed via the
// work_type_skills join table.
type Skill struct {
	ID         string    `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	IsRequired bool      `json:"is_required,omitempty"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CommonRole is global reusable master data describing a job title/role
// commonly associated with one or more work types (e.g. "Mason", "Picker").
type CommonRole struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	IsPrimary bool      `json:"is_primary,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
