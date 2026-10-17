package model

import (
	"strings"
	"time"
)

const (
	PaymentStatusPending       = "PENDING"
	PaymentStatusDue           = "DUE"
	PaymentStatusPartiallyPaid = "PARTIALLY_PAID"
	PaymentStatusPaid          = "PAID"
	PaymentStatusWaived        = "WAIVED"
	PaymentStatusDisputed      = "DISPUTED"

	PayeeTypeWorker     = "WORKER"
	PayeeTypeContractor = "CONTRACTOR"

	PaymentMethodCash         = "CASH"
	PaymentMethodUPI          = "UPI"
	PaymentMethodBankTransfer = "BANK_TRANSFER"
	PaymentMethodOther        = "OTHER"

	PaymentReceiptNone      = ""
	PaymentReceiptConfirmed = "CONFIRMED"
	PaymentReceiptDisputed  = "DISPUTED"
)

func ValidPaymentMethod(method string) bool {
	switch NormalizePaymentMethod(method) {
	case PaymentMethodCash, PaymentMethodUPI, PaymentMethodBankTransfer, PaymentMethodOther:
		return true
	default:
		return false
	}
}

func NormalizePaymentMethod(method string) string {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case PaymentMethodCash, "CASH_PAYMENT":
		return PaymentMethodCash
	case PaymentMethodUPI:
		return PaymentMethodUPI
	case PaymentMethodBankTransfer, "BANK", "NEFT", "IMPS", "RTGS":
		return PaymentMethodBankTransfer
	case PaymentMethodOther, "ONLINE", "EXTERNAL":
		return PaymentMethodOther
	default:
		return strings.ToUpper(strings.TrimSpace(method))
	}
}

func ObligationStatus(payableMinor int64, paidMinor int64) string {
	if payableMinor <= 0 {
		if paidMinor > 0 {
			return PaymentStatusPaid
		}
		return PaymentStatusDue
	}
	if paidMinor <= 0 {
		return PaymentStatusDue
	}
	if paidMinor < payableMinor {
		return PaymentStatusPartiallyPaid
	}
	return PaymentStatusPaid
}

func ObligationResolved(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case PaymentStatusPaid, PaymentStatusWaived:
		return true
	default:
		return false
	}
}

func PaymentObligationResolved(obligation PaymentObligation) bool {
	if obligation.PayableAmountMinor <= 0 {
		return true
	}
	return ObligationResolved(obligation.Status)
}

type PaymentObligation struct {
	ID                 string    `json:"id"`
	WorkID             string    `json:"work_id"`
	AssignmentID       string    `json:"assignment_id"`
	PayeeType          string    `json:"payee_type"`
	PayeeID            string    `json:"payee_id"`
	PayeeName          string    `json:"payee_name,omitempty"`
	Currency           string    `json:"currency"`
	PayableAmountMinor int64     `json:"payable_amount_minor"`
	PaidAmountMinor    int64     `json:"paid_amount_minor"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type PaymentTransaction struct {
	ID                  string     `json:"id"`
	PaymentObligationID string     `json:"payment_obligation_id"`
	WorkID              string     `json:"work_id"`
	AssignmentID        string     `json:"assignment_id"`
	AmountMinor         int64      `json:"amount_minor"`
	PaymentMethod       string     `json:"payment_method"`
	Reference           *string    `json:"reference,omitempty"`
	RecordedBy          string     `json:"recorded_by"`
	PaidAt              time.Time  `json:"paid_at"`
	ReceiptStatus       string     `json:"receipt_status,omitempty"`
	DisputeMessage      *string    `json:"dispute_message,omitempty"`
	ConfirmedAt         *time.Time `json:"confirmed_at,omitempty"`
	DisputedAt          *time.Time `json:"disputed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

type WorkPayment struct {
	ID             string     `json:"id"`
	WorkID         string     `json:"work_id"`
	AssignmentID   string     `json:"assignment_id"`
	WorkerID       string     `json:"worker_id"`
	Method         string     `json:"method"`
	Amount         float64    `json:"amount"`
	AmountMinor    int64      `json:"amount_minor,omitempty"`
	Currency       string     `json:"currency"`
	Reference      *string    `json:"reference,omitempty"`
	Status         string     `json:"status"`
	ReceiptStatus  string     `json:"receipt_status,omitempty"`
	DisputeMessage *string    `json:"dispute_message,omitempty"`
	RecordedAt     time.Time  `json:"recorded_at"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
	DisputedAt     *time.Time `json:"disputed_at,omitempty"`
}

type RecordPaymentRequest struct {
	Method    string  `json:"method"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Reference *string `json:"reference"`
}

type RecordObligationPaymentRequest struct {
	AmountMinor   int64   `json:"amount_minor"`
	PaymentMethod string  `json:"payment_method"`
	Reference     *string `json:"reference"`
	PaidAt        *string `json:"paid_at"`
}

type DisputePaymentRequest struct {
	Message *string `json:"message"`
}

type UpsertDailyLogRequest struct {
	DayPortion string  `json:"day_portion"`
	Note       *string `json:"note"`
}

type WorkPaymentItem struct {
	Obligation           PaymentObligation    `json:"obligation"`
	AssignmentID         string               `json:"assignment_id"`
	PayeeType            string               `json:"payee_type"`
	PayeeID              string               `json:"payee_id"`
	PayeeName            string               `json:"payee_name"`
	RateType             string               `json:"rate_type,omitempty"`
	PayableAmountMinor   int64                `json:"payable_amount_minor"`
	PaidAmountMinor      int64                `json:"paid_amount_minor"`
	RemainingAmountMinor int64                `json:"remaining_amount_minor"`
	Status               string               `json:"status"`
	Transactions         []PaymentTransaction `json:"transactions"`
}

type WorkPaymentsResponse struct {
	WorkID           string            `json:"work_id"`
	HiringMode       string            `json:"hiring_mode"`
	OutstandingMinor int64             `json:"outstanding_minor"`
	Payments         []WorkPaymentItem `json:"payments"`
}

type WorkerSettlement struct {
	Assignment    WorkAssignment `json:"assignment"`
	Logs          []DailyLog     `json:"logs"`
	Payments      []WorkPayment  `json:"payments"`
	FullDays      int            `json:"full_days"`
	HalfDays      int            `json:"half_days"`
	DaysWorked    float64        `json:"days_worked"`
	PayableAmount float64        `json:"payable_amount"`
	Currency      string         `json:"currency"`
	PaymentStatus string         `json:"payment_status"`
}

type WorkSettlement struct {
	Work    *WorkResponse      `json:"work"`
	Workers []WorkerSettlement `json:"workers"`
}
