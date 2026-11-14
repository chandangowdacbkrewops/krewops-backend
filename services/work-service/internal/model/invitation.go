package model

import (
	"fmt"
	"strings"
	"time"
)

const (
	InvitationStatusPending   = "PENDING"
	InvitationStatusAccepted  = "ACCEPTED"
	InvitationStatusDeclined  = "DECLINED"
	InvitationStatusCancelled = "CANCELLED"
	InvitationStatusExpired   = "EXPIRED"

	InvitationOutcomeCreated        = "created"
	InvitationOutcomeAlreadyPending = "already_pending"
	InvitationOutcomeRejected       = "rejected"

	InvitationReasonWorkerNotFound            = "worker_not_found"
	InvitationReasonProfileIncomplete         = "profile_incomplete"
	InvitationReasonNotVerified               = "not_verified"
	InvitationReasonWorkNotOpen               = "work_not_open"
	InvitationReasonNoRemainingCapacity       = "no_remaining_capacity"
	InvitationReasonWorkTypeMismatch          = "work_type_mismatch"
	InvitationReasonAlreadyAssigned           = "already_assigned"
	InvitationReasonInvitationPending         = "invitation_pending"
	InvitationReasonHiringModeMismatch        = "hiring_mode_mismatch"
	InvitationReasonContractorAlreadyAssigned = "contractor_already_assigned"
	InvitationReasonCrewSizeRequired          = "crew_size_required"
	InvitationReasonDuplicateInRequest        = "duplicate_in_request"

	InvitationTTL = 7 * 24 * time.Hour
)

type OfferedPayment struct {
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	RateBasis string  `json:"rate_basis"`
	RateUnit  *string `json:"rate_unit,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

func (p OfferedPayment) Quotation() Quotation {
	return Quotation{
		Amount:    p.Amount,
		Currency:  p.Currency,
		RateBasis: p.RateBasis,
		RateUnit:  p.RateUnit,
		Message:   p.Notes,
	}
}

type WorkInvitation struct {
	ID             string         `json:"id"`
	WorkID         string         `json:"work_id"`
	OwnerID        string         `json:"owner_id"`
	WorkerID       string         `json:"worker_id"`
	Status         string         `json:"status"`
	ApplicantType  string         `json:"applicant_type"`
	SeatCount      int            `json:"seat_count"`
	OfferedPayment OfferedPayment `json:"offered_payment"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	ExpiresAt      time.Time      `json:"expires_at"`
	RespondedAt    *time.Time     `json:"responded_at,omitempty"`
	AssignmentID   string         `json:"assignment_id,omitempty"`
	OwnerName      string         `json:"owner_name,omitempty"`
	WorkerName     string         `json:"worker_name,omitempty"`
	Work           *WorkResponse  `json:"work,omitempty"`
}

type InvitationCreateResult struct {
	WorkerID   string          `json:"worker_id"`
	Outcome    string          `json:"outcome"`
	Reason     string          `json:"reason,omitempty"`
	Invitation *WorkInvitation `json:"invitation,omitempty"`
}

type EligibleRecruitmentWork struct {
	Work      *WorkResponse `json:"work"`
	Eligible  bool          `json:"eligible"`
	Reason    string        `json:"reason,omitempty"`
	SeatCount int           `json:"seat_count"`
}

type RecruitmentWorker struct {
	UserID             string
	ProfileCompleted   bool
	VerificationStatus string
	WorkerType         string
	CrewSize           int
	WorkTypeIDs        []string
}

type RecruitmentEligibilityInput struct {
	Work                       *WorkRecord
	Worker                     *RecruitmentWorker
	HasAssignment              bool
	HasPendingInvite           bool
	HasActiveContractor        bool
	HasPendingContractorInvite bool
}

func NormalizeInvitationStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case InvitationStatusPending:
		return InvitationStatusPending
	case InvitationStatusAccepted:
		return InvitationStatusAccepted
	case InvitationStatusDeclined:
		return InvitationStatusDeclined
	case InvitationStatusCancelled, "CANCELED":
		return InvitationStatusCancelled
	case InvitationStatusExpired:
		return InvitationStatusExpired
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}

func InvitationTransitionAllowed(from string, to string) bool {
	from = NormalizeInvitationStatus(from)
	to = NormalizeInvitationStatus(to)
	if from != InvitationStatusPending {
		return false
	}
	switch to {
	case InvitationStatusAccepted, InvitationStatusDeclined, InvitationStatusCancelled, InvitationStatusExpired:
		return true
	default:
		return false
	}
}

func InvitationExpired(status string, expiresAt time.Time, now time.Time) bool {
	return NormalizeInvitationStatus(status) == InvitationStatusPending && !expiresAt.IsZero() && !now.Before(expiresAt)
}

func ResolveOfferedPayment(work *WorkRecord, override *OfferedPayment) (OfferedPayment, error) {
	if override != nil && (override.Amount > 0 || strings.TrimSpace(override.RateBasis) != "") {
		payment := *override
		if payment.Amount <= 0 {
			return OfferedPayment{}, fmt.Errorf("offered_payment.amount must be greater than 0")
		}
		payment.Currency = strings.ToUpper(strings.TrimSpace(payment.Currency))
		if payment.Currency == "" {
			payment.Currency = "INR"
		}
		payment.RateBasis = strings.ToLower(strings.TrimSpace(payment.RateBasis))
		if payment.RateBasis == "" {
			return OfferedPayment{}, fmt.Errorf("offered_payment.rate_basis is required")
		}
		if payment.RateUnit != nil {
			trimmed := strings.ToLower(strings.TrimSpace(*payment.RateUnit))
			if trimmed == "" {
				payment.RateUnit = nil
			} else {
				payment.RateUnit = &trimmed
			}
		}
		if payment.RateBasis != "negotiable" && payment.RateUnit == nil {
			return OfferedPayment{}, fmt.Errorf("offered_payment.rate_unit is required")
		}
		if payment.RateBasis == "negotiable" && payment.RateUnit != nil {
			return OfferedPayment{}, fmt.Errorf("offered_payment.rate_unit must be empty when rate_basis is negotiable")
		}
		return payment, nil
	}

	if work == nil || work.BudgetRate == nil || *work.BudgetRate <= 0 || work.RateBasis == nil || strings.TrimSpace(*work.RateBasis) == "" {
		return OfferedPayment{}, fmt.Errorf("offered_payment is required")
	}
	payment := OfferedPayment{
		Amount:    *work.BudgetRate,
		Currency:  "INR",
		RateBasis: strings.ToLower(strings.TrimSpace(*work.RateBasis)),
		RateUnit:  work.RateUnit,
		Notes:     work.PaymentNotes,
	}
	if payment.RateBasis != "negotiable" && (payment.RateUnit == nil || strings.TrimSpace(*payment.RateUnit) == "") {
		return OfferedPayment{}, fmt.Errorf("offered_payment.rate_unit is required")
	}
	return payment, nil
}

func RecruitmentEligibility(input RecruitmentEligibilityInput) (bool, string, int) {
	work := input.Work
	worker := input.Worker
	if worker == nil || strings.TrimSpace(worker.UserID) == "" {
		return false, InvitationReasonWorkerNotFound, 0
	}
	if !worker.ProfileCompleted {
		return false, InvitationReasonProfileIncomplete, 0
	}
	if !strings.EqualFold(strings.TrimSpace(worker.VerificationStatus), "verified") {
		return false, InvitationReasonNotVerified, 0
	}
	if work == nil || NormalizeWorkStatus(work.Status) != StatusOpen {
		return false, InvitationReasonWorkNotOpen, 0
	}

	remaining := RemainingWorkers(WorkersNeededCount(work.WorkersNeeded), work.WorkersSelected)
	if remaining < 1 {
		return false, InvitationReasonNoRemainingCapacity, 0
	}

	workTypeID := ""
	if work.WorkTypeID != nil {
		workTypeID = strings.TrimSpace(*work.WorkTypeID)
	}
	if workTypeID == "" || !containsString(worker.WorkTypeIDs, workTypeID) {
		return false, InvitationReasonWorkTypeMismatch, 0
	}

	applicantType := NormalizeApplicantType(worker.WorkerType)
	hiringMode := NormalizeHiringMode(work.HiringMode)
	if applicantType == ApplicantTypeContractor && hiringMode != HiringModeContractor {
		return false, InvitationReasonHiringModeMismatch, 0
	}
	if applicantType != ApplicantTypeContractor && hiringMode == HiringModeContractor {
		return false, InvitationReasonHiringModeMismatch, 0
	}

	seatCount := 1
	if applicantType == ApplicantTypeContractor {
		if worker.CrewSize < 1 {
			return false, InvitationReasonCrewSizeRequired, 0
		}
		seatCount = worker.CrewSize
		if seatCount > remaining {
			return false, InvitationReasonNoRemainingCapacity, seatCount
		}
		if input.HasActiveContractor || input.HasPendingContractorInvite {
			return false, InvitationReasonContractorAlreadyAssigned, seatCount
		}
	}

	if input.HasAssignment {
		return false, InvitationReasonAlreadyAssigned, seatCount
	}
	if input.HasPendingInvite {
		return false, InvitationReasonInvitationPending, seatCount
	}
	return true, "", seatCount
}

func ShouldAcceptAfterSeatClaim(claimed *WorkRecord) bool {
	return claimed != nil
}

func BuildAssignmentFromInvitation(work *WorkRecord, invitation *WorkInvitation) *WorkAssignment {
	assignmentType := AssignmentTypeIndividual
	if NormalizeApplicantType(invitation.ApplicantType) == ApplicantTypeContractor {
		assignmentType = AssignmentTypeContractor
	}
	seatCount := invitation.SeatCount
	if seatCount < 1 {
		seatCount = 1
	}
	quotation := invitation.OfferedPayment.Quotation()
	assignment := &WorkAssignment{
		WorkID:         work.ID,
		InvitationID:   invitation.ID,
		AssignmentType: assignmentType,
		WorkerID:       invitation.WorkerID,
		Status:         AssignmentStatusAssigned,
		Quotation:      quotation,
		AgreedPayment:  AgreedPaymentFromQuotation(quotation, assignmentType),
		JoinedAt:       time.Now().UTC().Format("2006-01-02"),
		PaymentStatus:  PaymentStatusDue,
	}
	if assignmentType == AssignmentTypeContractor {
		assignment.ContractorProfileID = invitation.WorkerID
		assignment.AssignedWorkerCount = seatCount
		assignment.CommittedWorkerCount = seatCount
		if qty := CompletedQuantityFromAttributes(work.Attributes); qty > 0 {
			assignment.CompletedQuantity = qty
		}
	} else {
		assignment.WorkerProfileID = invitation.WorkerID
		assignment.AssignedWorkerCount = 1
	}
	return assignment
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}
