package model

import "time"

const (
	ApplicationStatusApplied     = "APPLIED"
	ApplicationStatusShortlisted = "SHORTLISTED"
	ApplicationStatusAccepted    = "ACCEPTED"
	ApplicationStatusRejected    = "REJECTED"
	ApplicationStatusWithdrawn   = "WITHDRAWN"
	ApplicationStatusCancelled   = "CANCELLED"
)

func ValidApplicationStatus(status string) bool {
	switch status {
	case ApplicationStatusApplied,
		ApplicationStatusShortlisted,
		ApplicationStatusAccepted,
		ApplicationStatusRejected,
		ApplicationStatusWithdrawn,
		ApplicationStatusCancelled:
		return true
	default:
		return false
	}
}

type Quotation struct {
	Amount                 float64  `json:"amount"`
	Currency               string   `json:"currency"`
	PriceType              string   `json:"price_type"`
	EstimatedDurationHours *float64 `json:"estimated_duration_hours,omitempty"`
	Message                *string  `json:"message,omitempty"`
}

type ApplyToWorkRequest struct {
	Quotation Quotation `json:"quotation"`
}

type WorkApplication struct {
	ID        string    `json:"id"`
	WorkID    string    `json:"work_id"`
	WorkerID  string    `json:"worker_id"`
	Status    string    `json:"status"`
	Quotation Quotation `json:"quotation"`
	AppliedAt time.Time `json:"applied_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MyWorkApplication struct {
	Application WorkApplication `json:"application"`
	Work        *WorkResponse   `json:"work,omitempty"`
}
