package model

import "time"

const (
	UserTypeUser           = "user"
	RegistrationBusiness   = "business"
	RegistrationIndividual = "individual"
)

type BusinessType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CreateOwnerProfileRequest struct {
	RegistrationType      string  `json:"registration_type" binding:"required"`
	BusinessName          *string `json:"business_name"`
	BusinessDescription   *string `json:"business_description"`
	BusinessTypeID        string  `json:"business_type_id" binding:"required"`
	BusinessSize          *string `json:"business_size"`
	GSTRegistrationNumber *string `json:"gst_registration_number"`
	FirstName             string  `json:"first_name" binding:"required"`
	LastName              *string `json:"last_name"`
	DateOfBirth           string  `json:"date_of_birth" binding:"required"`
	Email                 *string `json:"email"`
	Country               string  `json:"country" binding:"required"`
	State                 string  `json:"state" binding:"required"`
	City                  string  `json:"city" binding:"required"`
	PostalCode            string  `json:"postal_code" binding:"required"`
	PreferredLanguage     string  `json:"preferred_language" binding:"required"`
}

type OwnerProfileDetails struct {
	RegistrationType      string       `json:"registration_type"`
	BusinessName          *string      `json:"business_name,omitempty"`
	BusinessDescription   *string      `json:"business_description,omitempty"`
	BusinessType          BusinessType `json:"business_type"`
	BusinessSize          *string      `json:"business_size,omitempty"`
	GSTRegistrationNumber *string      `json:"gst_registration_number,omitempty"`
}

type OwnerProfileResponse struct {
	ProfileCompleted  bool                 `json:"profile_completed"`
	UserType          string               `json:"user_type"`
	FirstName         string               `json:"first_name"`
	LastName          *string              `json:"last_name,omitempty"`
	DateOfBirth       string               `json:"date_of_birth"`
	Email             *string              `json:"email,omitempty"`
	Country           string               `json:"country"`
	State             string               `json:"state"`
	City              string               `json:"city"`
	PostalCode        string               `json:"postal_code"`
	PreferredLanguage string               `json:"preferred_language"`
	OwnerProfile      *OwnerProfileDetails `json:"owner_profile,omitempty"`
}

type ProfileRecord struct {
	ID                  string
	AuthUserID          string
	FirstName           *string
	LastName            *string
	UserType            *string
	DateOfBirth         *time.Time
	Email               *string
	Country             *string
	State               *string
	City                *string
	PostalCode          *string
	PreferredLanguage   *string
	ProfileCompleted    bool
	OnboardingCompleted bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type OwnerProfileRecord struct {
	ID                    string
	ProfileID             string
	RegistrationType      *string
	BusinessName          *string
	BusinessDescription   *string
	BusinessSize          *string
	GSTRegistrationNumber *string
	BusinessType          *BusinessType
}
