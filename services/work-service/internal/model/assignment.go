package model

import (
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	AssignmentTypeIndividual = "INDIVIDUAL"
	AssignmentTypeContractor = "CONTRACTOR"

	AssignmentStatusAssigned   = "ASSIGNED"
	AssignmentStatusInProgress = "IN_PROGRESS"
	AssignmentStatusSubmitted  = "SUBMITTED"
	AssignmentStatusApproved   = "APPROVED"
	AssignmentStatusCancelled  = "CANCELLED"

	// Legacy assignment spelling used before owner-driven execution.
	AssignmentStatusStarted   = AssignmentStatusInProgress
	AssignmentStatusCompleted = AssignmentStatusSubmitted

	RateTypeDaily           = "DAILY"
	RateTypePerWorkerPerDay = "PER_WORKER_PER_DAY"
	RateTypeFixed           = "FIXED"
	RateTypeAreaBased       = "AREA_BASED"

	RateUnitDay       = "DAY"
	RateUnitWorkerDay = "WORKER_DAY"
	RateUnitJob       = "JOB"
	RateUnitAcre      = "ACRE"
)

func ValidAssignmentStatus(status string) bool {
	switch NormalizeAssignmentStatus(status) {
	case AssignmentStatusAssigned,
		AssignmentStatusInProgress,
		AssignmentStatusSubmitted,
		AssignmentStatusApproved,
		AssignmentStatusCancelled:
		return true
	default:
		return false
	}
}

func NormalizeAssignmentStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case AssignmentStatusAssigned:
		return AssignmentStatusAssigned
	case AssignmentStatusInProgress, "STARTED":
		return AssignmentStatusInProgress
	case AssignmentStatusSubmitted, "COMPLETED":
		return AssignmentStatusSubmitted
	case AssignmentStatusApproved:
		return AssignmentStatusApproved
	case AssignmentStatusCancelled, "CANCELED":
		return AssignmentStatusCancelled
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}

func NormalizeAssignmentType(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case AssignmentTypeContractor:
		return AssignmentTypeContractor
	default:
		return AssignmentTypeIndividual
	}
}

func IsActiveAssignment(status string) bool {
	return NormalizeAssignmentStatus(status) != AssignmentStatusCancelled
}

type AgreedPayment struct {
	RateType         string `json:"rate_type"`
	RateUnit         string `json:"rate_unit"`
	RateMinor        int64  `json:"rate_minor"`
	Currency         string `json:"currency"`
	FixedAmountMinor *int64 `json:"fixed_amount_minor,omitempty"`
}

type WorkAssignment struct {
	ID             string `json:"id"`
	WorkID         string `json:"work_id"`
	ApplicationID  string `json:"application_id"`
	AssignmentType string `json:"assignment_type"`

	WorkerID            string `json:"worker_id"`
	WorkerProfileID     string `json:"worker_profile_id,omitempty"`
	ContractorProfileID string `json:"contractor_profile_id,omitempty"`

	Status        string        `json:"status"`
	Quotation     Quotation     `json:"agreed_quotation"`
	AgreedPayment AgreedPayment `json:"agreed_payment"`

	JoinedAt             string  `json:"joined_at,omitempty"`
	CommittedWorkerCount int     `json:"committed_worker_count,omitempty"`
	CompletedQuantity    float64 `json:"completed_quantity,omitempty"`

	AssignedAt  time.Time  `json:"assigned_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	PaymentStatus   string    `json:"payment_status"`
	FullDays        int       `json:"full_days"`
	HalfDays        int       `json:"half_days"`
	DaysWorked      float64   `json:"days_worked"`
	PayableAmount   float64   `json:"payable_amount"`
	PayableCurrency string    `json:"payable_currency,omitempty"`
	LogsFrozen      bool      `json:"logs_frozen"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (a WorkAssignment) PayeeID() string {
	if a.AssignmentType == AssignmentTypeContractor {
		if a.ContractorProfileID != "" {
			return a.ContractorProfileID
		}
		return a.WorkerID
	}
	if a.WorkerProfileID != "" {
		return a.WorkerProfileID
	}
	return a.WorkerID
}

func RupeesToMinor(amount float64) int64 {
	if amount <= 0 {
		return 0
	}
	return int64(math.Round(amount * 100))
}

func MinorToRupees(amountMinor int64) float64 {
	return float64(amountMinor) / 100
}

func AgreedPaymentFromQuotation(quotation Quotation, hiringMode string) AgreedPayment {
	currency := strings.ToUpper(strings.TrimSpace(quotation.Currency))
	if currency == "" {
		currency = "INR"
	}
	rateMinor := RupeesToMinor(quotation.Amount)
	basis := strings.ToLower(strings.TrimSpace(quotation.RateBasis))
	unit := ""
	if quotation.RateUnit != nil {
		unit = strings.ToLower(strings.TrimSpace(*quotation.RateUnit))
	}

	payment := AgreedPayment{
		Currency:  currency,
		RateMinor: rateMinor,
	}

	switch {
	case basis == "fixed" || unit == "job":
		payment.RateType = RateTypeFixed
		payment.RateUnit = RateUnitJob
		fixed := rateMinor
		payment.FixedAmountMinor = &fixed
	case strings.Contains(basis, "area") || unit == "acre" || unit == "hectare":
		payment.RateType = RateTypeAreaBased
		payment.RateUnit = RateUnitAcre
	case NormalizeHiringMode(hiringMode) == HiringModeContractor:
		payment.RateType = RateTypePerWorkerPerDay
		payment.RateUnit = RateUnitWorkerDay
	default:
		payment.RateType = RateTypeDaily
		payment.RateUnit = RateUnitDay
	}
	return payment
}

func IndividualPayableMinor(fullDays int, halfDays int, rateMinor int64) int64 {
	if rateMinor <= 0 {
		return 0
	}
	halfUnits := int64(fullDays)*2 + int64(halfDays)
	return halfUnits * rateMinor / 2
}

func ContractorWorkerDayPayableMinor(workerDays int, rateMinor int64) int64 {
	if workerDays <= 0 || rateMinor <= 0 {
		return 0
	}
	return int64(workerDays) * rateMinor
}

func AreaBasedPayableMinor(quantity float64, rateMinor int64) int64 {
	if quantity <= 0 || rateMinor <= 0 {
		return 0
	}
	return int64(math.Round(quantity * float64(rateMinor)))
}

func AssignmentPayableMinor(assignment WorkAssignment, fullDays int, halfDays int, workerDays int, quantity float64) int64 {
	payment := assignment.AgreedPayment
	if payment.RateMinor == 0 && assignment.Quotation.Amount > 0 {
		payment = AgreedPaymentFromQuotation(assignment.Quotation, assignment.AssignmentType)
	}
	switch payment.RateType {
	case RateTypeFixed:
		if payment.FixedAmountMinor != nil {
			return *payment.FixedAmountMinor
		}
		return payment.RateMinor
	case RateTypeAreaBased:
		return AreaBasedPayableMinor(quantity, payment.RateMinor)
	case RateTypePerWorkerPerDay:
		return ContractorWorkerDayPayableMinor(workerDays, payment.RateMinor)
	default:
		return IndividualPayableMinor(fullDays, halfDays, payment.RateMinor)
	}
}

func CompletedQuantityFromAttributes(attributes map[string]string) float64 {
	if attributes == nil {
		return 0
	}
	for _, key := range []string{"completed_area", "area", "acres", "acreage", "quantity"} {
		if raw, ok := attributes[key]; ok {
			trimmed := strings.TrimSpace(raw)
			trimmed = strings.TrimSuffix(strings.ToLower(trimmed), "acres")
			trimmed = strings.TrimSuffix(trimmed, "acre")
			trimmed = strings.TrimSpace(trimmed)
			value, err := strconv.ParseFloat(trimmed, 64)
			if err == nil && value > 0 {
				return value
			}
		}
	}
	return 0
}
