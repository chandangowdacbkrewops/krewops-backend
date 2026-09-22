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

type Profile struct {
	ID                  string     `json:"id"`
	AuthUserID          string     `json:"user_id"`
	FirstName           *string    `json:"first_name,omitempty"`
	LastName            *string    `json:"last_name,omitempty"`
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
	FirstName  string `json:"first_name" binding:"required"`
	LastName   string `json:"last_name" binding:"required"`
	Country    string `json:"country" binding:"required"`
	State      string `json:"state" binding:"required"`
	City       string `json:"city" binding:"required"`
	PostalCode string `json:"postal_code" binding:"required"`
	UserType   string `json:"user_type" binding:"required"`
}

type UpdateProfileRequest struct {
	FirstName  string `json:"first_name" binding:"required"`
	LastName   string `json:"last_name" binding:"required"`
	Country    string `json:"country" binding:"required"`
	State      string `json:"state" binding:"required"`
	City       string `json:"city" binding:"required"`
	PostalCode string `json:"postal_code" binding:"required"`
}

type CreateWorkerProfileRequest struct {
	WorkerType         string   `json:"worker_type" binding:"required"`
	CrewName           *string  `json:"crew_name"`
	CrewSize           int32    `json:"crew_size"`
	ExperienceYears    *float64 `json:"experience_years"`
	ExpectedRate       *float64 `json:"expected_rate"`
	RateType           *string  `json:"rate_type"`
	AvailabilityStatus *string  `json:"availability_status"`
	Bio                *string  `json:"bio"`
	WorkTypeIDs        []string `json:"work_type_ids" binding:"required,min=1"`
}

type WorkerProfileSkill struct {
	ID              string    `json:"id"`
	WorkerProfileID string    `json:"worker_profile_id"`
	WorkTypeID      string    `json:"work_type_id"`
	ExperienceYears *float64  `json:"experience_years,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type WorkerProfile struct {
	ID                 string               `json:"id"`
	UserID             string               `json:"user_id"`
	WorkerType         string               `json:"worker_type"`
	CrewName           *string              `json:"crew_name,omitempty"`
	CrewSize           int32                `json:"crew_size"`
	ExperienceYears    *float64             `json:"experience_years,omitempty"`
	ExpectedRate       *float64             `json:"expected_rate,omitempty"`
	RateType           *string              `json:"rate_type,omitempty"`
	AvailabilityStatus string               `json:"availability_status"`
	VerificationStatus string               `json:"verification_status"`
	Bio                *string              `json:"bio,omitempty"`
	ProfileCompleted   bool                 `json:"profile_completed"`
	Skills             []WorkerProfileSkill `json:"skills"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
}

type ProfileResponse struct {
	ID                  string               `json:"id,omitempty"`
	AuthUserID          string               `json:"user_id,omitempty"`
	ProfileCompleted    bool                 `json:"profile_completed"`
	FirstName           *string              `json:"first_name,omitempty"`
	LastName            *string              `json:"last_name,omitempty"`
	UserType            *string              `json:"user_type,omitempty"`
	DateOfBirth         *string              `json:"date_of_birth,omitempty"`
	Email               *string              `json:"email,omitempty"`
	Country             *string              `json:"country,omitempty"`
	State               *string              `json:"state,omitempty"`
	City                *string              `json:"city,omitempty"`
	PostalCode          *string              `json:"postal_code,omitempty"`
	PreferredLanguage   *string              `json:"preferred_language,omitempty"`
	OwnerProfile        *OwnerProfileDetails `json:"owner_profile,omitempty"`
	OnboardingCompleted bool                 `json:"onboarding_completed"`
}
