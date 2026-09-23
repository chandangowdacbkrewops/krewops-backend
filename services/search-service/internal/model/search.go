package model

import "time"

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// NormalizePagination clamps page/page_size to sane bounds shared by both
// SearchWork and SearchWorkers.
func NormalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = DefaultPage
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

// -----------------------------------------------------------------------
// Work search
// -----------------------------------------------------------------------

// WorkSearchFilters carries the optional query parameters accepted by
// SearchWork. All fields are optional; an unset filter is not applied.
type WorkSearchFilters struct {
	Keyword         *string
	WorkTypeID      *string
	City            *string
	State           *string
	ExperienceLevel *string
	PaymentType     *string
	MinBudgetRate   *float64
	MaxBudgetRate   *float64

	Page     int
	PageSize int
}

// WorkSearchResult is a single work_postings row joined with its work type
// name, projected for the search results list (not the full CreateWork
// response shape).
type WorkSearchResult struct {
	ID     string
	UserID string
	Status string

	Title        *string
	WorkTypeID   *string
	WorkTypeName *string
	Description  *string

	Address   *string
	City      *string
	State     *string
	Latitude  *float64
	Longitude *float64

	WorkersNeeded   *int
	ExperienceLevel *string
	Skills          []string

	StartDate     *time.Time
	DurationValue *int
	DurationUnit  *string
	ShiftTiming   *string

	PaymentType *string
	BudgetRate  *float64

	PublishedAt *time.Time
	CreatedAt   time.Time
}

// WorkSearchResponse is the paginated result set returned by SearchWork.
type WorkSearchResponse struct {
	Results  []WorkSearchResult
	Total    int
	Page     int
	PageSize int
}

// -----------------------------------------------------------------------
// Worker search
// -----------------------------------------------------------------------

// WorkerSearchFilters carries the optional query parameters accepted by
// SearchWorkers.
type WorkerSearchFilters struct {
	Keyword            *string
	WorkTypeID         *string
	City               *string
	State              *string
	WorkerType         *string
	AvailabilityStatus *string
	MinExperienceYears *float64

	Page     int
	PageSize int
}

// WorkerSkillResult is one worker_work_types row joined with its work
// type name.
type WorkerSkillResult struct {
	WorkTypeID      string
	WorkTypeName    string
	ExperienceYears *float64
}

// WorkerSearchResult is a worker_profiles row joined with its owning
// profile (for name/location) and its skills, projected for search
// results.
type WorkerSearchResult struct {
	ID     string
	UserID string

	FirstName *string
	LastName  *string
	City      *string
	State     *string

	WorkerType string
	CrewName   *string
	CrewSize   int

	ExperienceYears *float64
	ExpectedRate    *float64
	RateType        *string

	AvailabilityStatus string
	VerificationStatus string

	Bio *string

	Skills []WorkerSkillResult
}

// WorkerSearchResponse is the paginated result set returned by
// SearchWorkers.
type WorkerSearchResponse struct {
	Results  []WorkerSearchResult
	Total    int
	Page     int
	PageSize int
}
