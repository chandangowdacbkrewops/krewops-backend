package model

import (
	"testing"
	"time"
)

func TestSumDaysWorked(t *testing.T) {
	full, half, days := SumDaysWorked([]DailyLog{
		{DayPortion: DayPortionFull},
		{DayPortion: DayPortionFull},
		{DayPortion: DayPortionHalf},
		{DayType: DayPortionNotWorked},
	})
	if full != 2 || half != 1 || days != 2.5 {
		t.Fatalf("got full=%d half=%d days=%v", full, half, days)
	}
	if PayableAmount(1.5, 750) != 1125 {
		t.Fatalf("payable = %v", PayableAmount(1.5, 750))
	}
}

func TestUnrecordedIsNotNotWorked(t *testing.T) {
	logs := []DailyLog{
		{WorkDate: "2026-10-20", DayType: DayPortionFull},
		{WorkDate: "2026-10-21", DayType: DayPortionNotWorked},
	}
	full, half, notWorked, payable := SumDayTypes(logs)
	if full != 1 || half != 0 || notWorked != 1 || payable != 1 {
		t.Fatalf("got full=%d half=%d notWorked=%d payable=%v", full, half, notWorked, payable)
	}
	expected := []string{"2026-10-20", "2026-10-21", "2026-10-22"}
	if CountUnrecordedDays(expected, logs) != 1 {
		t.Fatal("missing date must count as unrecorded, not not-worked")
	}
}

func TestIndividualHalfDayPayment(t *testing.T) {
	if IndividualPayableMinor(3, 1, 50000) != 175000 {
		t.Fatalf("3 full + 1 half at 50000 should be 175000, got %d", IndividualPayableMinor(3, 1, 50000))
	}
	if IndividualPayableMinor(0, 0, 50000) != 0 {
		t.Fatal("unrecorded days must not create payable amount")
	}
	if IndividualPayableMinor(0, 1, 50000) != 25000 {
		t.Fatal("half day should pay half the rate")
	}
}

func TestWorkDateInWindow(t *testing.T) {
	start := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	duration := 3
	unit := DurationUnitDays
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

	if err := WorkDateInWindow(start, &start, &duration, &unit, now); err != nil {
		t.Fatalf("start date should be allowed: %v", err)
	}
	if err := WorkDateInWindow(start.AddDate(0, 0, 2), &start, &duration, &unit, now.AddDate(0, 0, 2)); err != nil {
		t.Fatalf("last duration day should be allowed: %v", err)
	}
	if err := WorkDateInWindow(start.AddDate(0, 0, 3), &start, &duration, &unit, now.AddDate(0, 0, 3)); err == nil {
		t.Fatal("date after duration should be rejected")
	}
	if err := WorkDateInWindow(start.AddDate(0, 0, 1), &start, &duration, &unit, start); err == nil {
		t.Fatal("future date should be rejected")
	}
}

func TestDateOnOrAfterJoined(t *testing.T) {
	date := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	if err := DateOnOrAfterJoined(date, "2026-10-20"); err == nil {
		t.Fatal("date before joined_at should be rejected")
	}
	if err := DateOnOrAfterJoined(date.AddDate(0, 0, 1), "2026-10-20"); err != nil {
		t.Fatalf("joined_at date should be allowed: %v", err)
	}
}

func TestPaymentMethodNormalization(t *testing.T) {
	if !ValidPaymentMethod("cash") || NormalizePaymentMethod("ONLINE") != PaymentMethodOther {
		t.Fatal("expected cash and online/other mapping")
	}
	if NormalizePaymentMethod("bank_transfer") != PaymentMethodBankTransfer {
		t.Fatal("expected bank transfer mapping")
	}
	if ValidPaymentMethod("bitcoin") {
		t.Fatal("unexpected method")
	}
}

func TestContractorPaymentTypes(t *testing.T) {
	if ContractorWorkerDayPayableMinor(14, 50000) != 700000 {
		t.Fatal("per worker per day should multiply deployed worker-days")
	}
	fixed := int64(1500000)
	assignment := WorkAssignment{AgreedPayment: AgreedPayment{RateType: RateTypeFixed, RateMinor: 1500000, FixedAmountMinor: &fixed}}
	if AssignmentPayableMinor(assignment, 0, 0, 4, 0) != 1500000 {
		t.Fatal("fixed payment must not depend on deployment")
	}
	area := WorkAssignment{AgreedPayment: AgreedPayment{RateType: RateTypeAreaBased, RateMinor: 300000}}
	if AssignmentPayableMinor(area, 0, 0, 14, 4) != 1200000 {
		t.Fatal("area-based payment should use completed quantity")
	}
}

func TestContractorShortfallAllowed(t *testing.T) {
	if ContractorShortfall(5, 4) != 1 {
		t.Fatal("expected shortfall of 1")
	}
	if ContractorShortfall(5, 5) != 0 {
		t.Fatal("full deployment has no shortfall")
	}
}

func TestObligationStatus(t *testing.T) {
	if ObligationStatus(175000, 0) != PaymentStatusDue {
		t.Fatal("zero paid should be DUE")
	}
	if ObligationStatus(175000, 100000) != PaymentStatusPartiallyPaid {
		t.Fatal("partial payment should be PARTIALLY_PAID")
	}
	if ObligationStatus(175000, 175000) != PaymentStatusPaid {
		t.Fatal("full payment should be PAID")
	}
	if ObligationStatus(0, 0) != PaymentStatusDue {
		t.Fatal("zero payable with no payment should stay DUE until records exist")
	}
	if !PaymentObligationResolved(PaymentObligation{PayableAmountMinor: 0, Status: PaymentStatusDue}) {
		t.Fatal("nothing payable should allow close")
	}
	if !ObligationResolved(PaymentStatusWaived) || ObligationResolved(PaymentStatusDue) {
		t.Fatal("waived is resolved, due is not")
	}
}
