package model

import "time"

const (
	UserTypeWorker = "worker"
	UserTypeAdmin  = "admin"
)

// ValidUserType reports whether userType (expected lowercase) is one of the
// known roles. UserTypeUser is defined in owner_profile.go (same package).
func ValidUserType(userType string) bool {
	switch userType {
	case UserTypeWorker, UserTypeUser, UserTypeAdmin:
		return true
	default:
		return false
	}
}

const (
	GenderMale   = "male"
	GenderFemale = "female"
	GenderOther  = "other"
)

func ValidGender(gender string) bool {
	switch gender {
	case GenderMale, GenderFemale, GenderOther:
		return true
	default:
		return false
	}
}

type Profile struct {
	ID                  string     `json:"id"`
	AuthUserID          string     `json:"user_id"`
	FirstName           *string    `json:"first_name,omitempty"`
	LastName            *string    `json:"last_name,omitempty"`
	Gender              *string    `json:"gender,omitempty"`
	UserType            *string    `json:"user_type,omitempty"`
	DateOfBirth         *time.Time `json:"date_of_birth,omitempty"`
	Email               *string    `json:"email,omitempty"`
	Country             *string    `json:"country,omitempty"`
	State               *string    `json:"state,omitempty"`
	City                *string    `json:"city,omitempty"`
	PostalCode          *string    `json:"postal_code,omitempty"`
	PreferredLanguage   *string    `json:"preferred_language,omitempty"`
	ProfileCompleted    bool       `json:"profile_completed"`
	OnboardingCompleted bool       `json:"onboarding_completed"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type CreateProfileRequest struct {
	FirstName  string  `json:"first_name" binding:"required"`
	LastName   string  `json:"last_name" binding:"required"`
	Gender     *string `json:"gender"`
	Country    string  `json:"country" binding:"required"`
	State      string  `json:"state" binding:"required"`
	City       string  `json:"city" binding:"required"`
	PostalCode string  `json:"postal_code" binding:"required"`
	UserType   string  `json:"user_type" binding:"required"`
}

type UpdateProfileRequest struct {
	FirstName  string  `json:"first_name" binding:"required"`
	LastName   string  `json:"last_name" binding:"required"`
	Gender     *string `json:"gender"`
	Country    string  `json:"country" binding:"required"`
	State      string  `json:"state" binding:"required"`
	City       string  `json:"city" binding:"required"`
	PostalCode string  `json:"postal_code" binding:"required"`
}

type WorkerTypeSelection struct {
	WorkTypeID string   `json:"work_type_id"`
	SkillIDs   []string `json:"skill_ids"`
}

type WorkerSegmentSelection struct {
	WorkSegmentID string                `json:"work_segment_id"`
	WorkTypes     []WorkerTypeSelection `json:"work_types"`
}

type WorkerCategorySelection struct {
	WorkCategoryID string                   `json:"work_category_id"`
	Segments       []WorkerSegmentSelection `json:"segments"`
}

type CreateWorkerProfileRequest struct {
	WorkerType         string                    `json:"worker_type" binding:"required"`
	CrewName           *string                   `json:"crew_name"`
	CrewSize           int32                     `json:"crew_size"`
	ExperienceYears    *float64                  `json:"experience_years"`
	ExpectedRate       *float64                  `json:"expected_rate"`
	ExpectedRateMin    *float64                  `json:"expected_rate_min"`
	ExpectedRateMax    *float64                  `json:"expected_rate_max"`
	RateBasis          *string                   `json:"rate_basis"`
	RateUnit           *string                   `json:"rate_unit"`
	AvailabilityStatus *string                   `json:"availability_status"`
	Bio                *string                   `json:"bio"`
	Selections         []WorkerCategorySelection `json:"selections"`
}

type UpdateWorkerProfileRequest struct {
	Selections []WorkerCategorySelection `json:"selections"`
	RateBasis  *string                   `json:"rate_basis"`
	RateUnit   *string                   `json:"rate_unit"`
}

type WorkerSkillSummary struct {
	ID         string `json:"id"`
	WorkTypeID string `json:"work_type_id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
}

type WorkerWorkTypeSummary struct {
	WorkTypeID string               `json:"work_type_id"`
	Name       string               `json:"name"`
	CategoryID string               `json:"category_id"`
	SegmentID  string               `json:"segment_id"`
	IsPrimary  bool                 `json:"is_primary"`
	Skills     []WorkerSkillSummary `json:"skills"`
}

type WorkerWorkSegmentSummary struct {
	WorkSegmentID string                  `json:"work_segment_id"`
	Code          string                  `json:"code"`
	Name          string                  `json:"name"`
	CategoryID    string                  `json:"category_id"`
	WorkTypes     []WorkerWorkTypeSummary `json:"work_types"`
}

type WorkerWorkCategorySummary struct {
	WorkCategoryID string                     `json:"work_category_id"`
	Code           string                     `json:"code"`
	Name           string                     `json:"name"`
	Segments       []WorkerWorkSegmentSummary `json:"segments"`
	WorkTypes      []WorkerWorkTypeSummary    `json:"work_types"`
}

type WorkerProfile struct {
	ID                 string                      `json:"id"`
	UserID             string                      `json:"user_id"`
	WorkerType         string                      `json:"worker_type"`
	CrewName           *string                     `json:"crew_name,omitempty"`
	CrewSize           int32                       `json:"crew_size"`
	ExperienceYears    *float64                    `json:"experience_years,omitempty"`
	ExpectedRate       *float64                    `json:"expected_rate,omitempty"`
	ExpectedRateMin    *float64                    `json:"expected_rate_min,omitempty"`
	ExpectedRateMax    *float64                    `json:"expected_rate_max,omitempty"`
	RateBasis          *string                     `json:"rate_basis,omitempty"`
	RateUnit           *string                     `json:"rate_unit,omitempty"`
	AvailabilityStatus string                      `json:"availability_status"`
	VerificationStatus string                      `json:"verification_status"`
	Bio                *string                     `json:"bio,omitempty"`
	ProfileCompleted   bool                        `json:"profile_completed"`
	WorkCategories     []WorkerWorkCategorySummary `json:"work_categories"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
}

type ProfileResponse struct {
	ID                  string                      `json:"id,omitempty"`
	AuthUserID          string                      `json:"user_id,omitempty"`
	ProfileCompleted    bool                        `json:"profile_completed"`
	FirstName           *string                     `json:"first_name,omitempty"`
	LastName            *string                     `json:"last_name,omitempty"`
	Gender              *string                     `json:"gender,omitempty"`
	UserType            *string                     `json:"user_type,omitempty"`
	DateOfBirth         *string                     `json:"date_of_birth,omitempty"`
	Email               *string                     `json:"email,omitempty"`
	Country             *string                     `json:"country,omitempty"`
	State               *string                     `json:"state,omitempty"`
	City                *string                     `json:"city,omitempty"`
	PostalCode          *string                     `json:"postal_code,omitempty"`
	PreferredLanguage   *string                     `json:"preferred_language,omitempty"`
	OwnerProfile        *OwnerProfileDetails        `json:"owner_profile,omitempty"`
	OnboardingCompleted bool                        `json:"onboarding_completed"`
	WorkCategories      []WorkerWorkCategorySummary `json:"work_categories,omitempty"`
	ExpectedRateMin     *float64                    `json:"expected_rate_min,omitempty"`
	ExpectedRateMax     *float64                    `json:"expected_rate_max,omitempty"`
	RateBasis           *string                     `json:"rate_basis,omitempty"`
	RateUnit            *string                     `json:"rate_unit,omitempty"`
}
