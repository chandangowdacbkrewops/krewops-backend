package model

import (
	"strings"
	"time"
)

// WorkType mirrors user-service's lookup row for the shared work_types table.
type WorkType struct {
	ID         string `json:"id"`
	CategoryID string `json:"category_id"`
	SegmentID  string `json:"segment_id"`
	Name       string `json:"name"`
}

// WorkCategory mirrors a row in the shared work_categories table.
type WorkCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// WorkSegment mirrors a row in the shared work_segments table.
type WorkSegment struct {
	ID         string `json:"id"`
	CategoryID string `json:"category_id"`
	Name       string `json:"name"`
}

const (
	StatusDraft         = "draft"
	StatusPublished     = "published"
	StatusAssigned      = "assigned"
	StatusInProgress    = "in_progress"
	StatusWorkCompleted = "work_completed"
	StatusClosed        = "closed"
	StatusCancelled     = "cancelled"
)

// ValidCreateStatus reports whether a client may set this status when posting work.
// published is the open state used by the existing create, apply, and search APIs.
func ValidCreateStatus(status string) bool {
	switch status {
	case StatusDraft, StatusPublished:
		return true
	default:
		return false
	}
}

func ValidStatus(status string) bool {
	switch status {
	case StatusDraft,
		StatusPublished,
		StatusAssigned,
		StatusInProgress,
		StatusWorkCompleted,
		StatusClosed,
		StatusCancelled:
		return true
	default:
		return false
	}
}

// NormalizeWorkStatus maps stored values and the lifecycle names used by clients
// onto the status values persisted with a work posting.
func NormalizeWorkStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "DRAFT":
		return StatusDraft
	case "PUBLISHED", "OPEN":
		return StatusPublished
	case "ASSIGNED":
		return StatusAssigned
	case "IN_PROGRESS", "INPROGRESS":
		return StatusInProgress
	case "WORK_COMPLETED", "WORKCOMPLETED":
		return StatusWorkCompleted
	case "CLOSED":
		return StatusClosed
	case "CANCELLED", "CANCELED":
		return StatusCancelled
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

// CanTransition reports whether a work posting may move from one status to another.
// Workers cannot move a work directly to closed; that step is owner confirmation only.
func CanTransition(from string, to string) bool {
	from = NormalizeWorkStatus(from)
	to = NormalizeWorkStatus(to)
	switch to {
	case StatusAssigned:
		return from == StatusPublished
	case StatusPublished:
		return from == StatusAssigned
	case StatusInProgress:
		return from == StatusAssigned || from == StatusPublished
	case StatusWorkCompleted:
		return from == StatusInProgress
	case StatusClosed:
		return from == StatusWorkCompleted
	case StatusCancelled:
		switch from {
		case StatusDraft, StatusPublished, StatusAssigned, StatusInProgress:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

// StatusMatchValues returns stored status spellings that should match the given statuses.
func StatusMatchValues(statuses ...string) []string {
	seen := make(map[string]struct{})
	values := make([]string, 0, len(statuses)*4)
	for _, status := range statuses {
		for _, alias := range statusAliases(NormalizeWorkStatus(status)) {
			if _, exists := seen[alias]; exists {
				continue
			}
			seen[alias] = struct{}{}
			values = append(values, alias)
		}
	}
	return values
}

func statusAliases(status string) []string {
	switch NormalizeWorkStatus(status) {
	case StatusPublished:
		return []string{StatusPublished, "PUBLISHED", "open", "OPEN"}
	case StatusAssigned:
		return []string{StatusAssigned, "ASSIGNED"}
	case StatusInProgress:
		return []string{StatusInProgress, "IN_PROGRESS", "inprogress", "INPROGRESS"}
	case StatusWorkCompleted:
		return []string{StatusWorkCompleted, "WORK_COMPLETED", "workcompleted", "WORKCOMPLETED"}
	case StatusCancelled:
		return []string{StatusCancelled, "CANCELLED", "canceled", "CANCELED"}
	case StatusDraft:
		return []string{StatusDraft, "DRAFT"}
	case StatusClosed:
		return []string{StatusClosed, "CLOSED"}
	default:
		if status == "" {
			return nil
		}
		return []string{status}
	}
}

const (
	DurationUnitDays   = "days"
	DurationUnitWeeks  = "weeks"
	DurationUnitMonths = "months"
)

func ValidDurationUnit(unit string) bool {
	switch unit {
	case DurationUnitDays, DurationUnitWeeks, DurationUnitMonths:
		return true
	default:
		return false
	}
}

// CreateWorkRequest carries the "Post Work" form fields. All fields are
// optional at the binding layer because a "draft" posting may be saved with
// partial data; full presence/format validation is enforced in the service
// layer only when Status resolves to StatusPublished.
type CreateWorkRequest struct {
	Status *string `json:"status"`

	Title          *string `json:"title"`
	WorkTypeID     *string `json:"work_type_id"`
	WorkCategoryID *string `json:"work_category_id"`
	WorkSegmentID  *string `json:"work_segment_id"`
	Description    *string `json:"description"`

	Address   *string  `json:"address"`
	City      *string  `json:"city"`
	State     *string  `json:"state"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`

	WorkersNeeded     *int     `json:"workers_needed"`
	ExperienceLevel   *string  `json:"experience_level"`
	Skills            []string `json:"skills"`
	ToolsProvided     *bool    `json:"tools_provided"`
	MaterialsProvided *bool    `json:"materials_provided"`

	StartDate     *string `json:"start_date"`
	DurationValue *int    `json:"duration_value"`
	DurationUnit  *string `json:"duration_unit"`
	ShiftTiming   *string `json:"shift_timing"`

	RateBasis             *string  `json:"rate_basis"`
	RateUnit              *string  `json:"rate_unit"`
	BudgetRate            *float64 `json:"budget_rate"`
	PaymentNotes          *string  `json:"payment_notes"`
	AccommodationProvided *bool    `json:"accommodation_provided"`
	MealsProvided         *bool    `json:"meals_provided"`

	Attributes map[string]string `json:"attributes"`
}

// WorkRecord is the internal representation of a row in work_postings.
type WorkRecord struct {
	ID     string
	UserID string

	Title            *string
	WorkTypeID       *string
	WorkTypeName     *string
	WorkCategoryID   *string
	WorkCategoryName *string
	WorkSegmentID    *string
	WorkSegmentName  *string
	Description      *string

	Attributes map[string]string

	Address   *string
	City      *string
	State     *string
	Latitude  *float64
	Longitude *float64

	WorkersNeeded     *int
	ExperienceLevel   *string
	Skills            []string
	ToolsProvided     *bool
	MaterialsProvided *bool

	StartDate     *time.Time
	DurationValue *int
	DurationUnit  *string
	ShiftTiming   *string

	RateBasis             *string
	RateUnit              *string
	BudgetRate            *float64
	PaymentNotes          *string
	AccommodationProvided *bool
	MealsProvided         *bool

	Status      string
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WorkResponse is the JSON shape returned to clients.
type WorkResponse struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	Status string `json:"status"`

	Title            *string `json:"title,omitempty"`
	WorkTypeID       *string `json:"work_type_id,omitempty"`
	WorkTypeName     *string `json:"work_type_name,omitempty"`
	WorkCategoryID   *string `json:"work_category_id,omitempty"`
	WorkCategoryName *string `json:"work_category_name,omitempty"`
	WorkSegmentID    *string `json:"work_segment_id,omitempty"`
	WorkSegmentName  *string `json:"work_segment_name,omitempty"`
	Description      *string `json:"description,omitempty"`

	Address   *string  `json:"address,omitempty"`
	City      *string  `json:"city,omitempty"`
	State     *string  `json:"state,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`

	WorkersNeeded     *int     `json:"workers_needed,omitempty"`
	ExperienceLevel   *string  `json:"experience_level,omitempty"`
	Skills            []string `json:"skills"`
	ToolsProvided     *bool    `json:"tools_provided,omitempty"`
	MaterialsProvided *bool    `json:"materials_provided,omitempty"`

	StartDate     *string `json:"start_date,omitempty"`
	DurationValue *int    `json:"duration_value,omitempty"`
	DurationUnit  *string `json:"duration_unit,omitempty"`
	ShiftTiming   *string `json:"shift_timing,omitempty"`

	RateBasis             *string  `json:"rate_basis,omitempty"`
	RateUnit              *string  `json:"rate_unit,omitempty"`
	BudgetRate            *float64 `json:"budget_rate,omitempty"`
	PaymentNotes          *string  `json:"payment_notes,omitempty"`
	AccommodationProvided *bool    `json:"accommodation_provided,omitempty"`
	MealsProvided         *bool    `json:"meals_provided,omitempty"`

	PublishedAt *string   `json:"published_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Attributes map[string]string `json:"attributes"`
}

func BuildWorkResponse(record *WorkRecord) *WorkResponse {
	resp := &WorkResponse{
		ID:     record.ID,
		UserID: record.UserID,
		Status: record.Status,

		Title:            record.Title,
		WorkTypeID:       record.WorkTypeID,
		WorkTypeName:     record.WorkTypeName,
		WorkCategoryID:   record.WorkCategoryID,
		WorkCategoryName: record.WorkCategoryName,
		WorkSegmentID:    record.WorkSegmentID,
		WorkSegmentName:  record.WorkSegmentName,
		Description:      record.Description,

		Address:   record.Address,
		City:      record.City,
		State:     record.State,
		Latitude:  record.Latitude,
		Longitude: record.Longitude,

		WorkersNeeded:     record.WorkersNeeded,
		ExperienceLevel:   record.ExperienceLevel,
		Skills:            record.Skills,
		ToolsProvided:     record.ToolsProvided,
		MaterialsProvided: record.MaterialsProvided,

		DurationValue: record.DurationValue,
		DurationUnit:  record.DurationUnit,
		ShiftTiming:   record.ShiftTiming,

		RateBasis:             record.RateBasis,
		RateUnit:              record.RateUnit,
		BudgetRate:            record.BudgetRate,
		PaymentNotes:          record.PaymentNotes,
		AccommodationProvided: record.AccommodationProvided,
		MealsProvided:         record.MealsProvided,

		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,

		Attributes: record.Attributes,
	}

	if record.Skills == nil {
		resp.Skills = []string{}
	}

	if record.Attributes == nil {
		resp.Attributes = map[string]string{}
	}

	if record.StartDate != nil {
		formatted := record.StartDate.Format("2006-01-02")
		resp.StartDate = &formatted
	}

	if record.PublishedAt != nil {
		formatted := record.PublishedAt.Format(time.RFC3339)
		resp.PublishedAt = &formatted
	}

	return resp
}

// WorkTypeField is the subset of user-service field definitions needed
// to validate dynamic attributes on create.
type WorkTypeField struct {
	FieldKey   string
	IsRequired bool
}
