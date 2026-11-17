package model

import "testing"

func TestNormalizeWorkStatus(t *testing.T) {
	cases := map[string]string{
		"draft":           StatusDraft,
		"OPEN":            StatusOpen,
		"published":       StatusOpen,
		"ASSIGNED":        StatusOpen,
		"in_progress":     StatusInProgress,
		"COMPLETED":       StatusCompleted,
		"WORK_COMPLETED":  StatusPaymentPending,
		"PAYMENT_PENDING": StatusPaymentPending,
		"closed":          StatusClosed,
		"CANCELLED":       StatusCancelled,
	}

	for input, want := range cases {
		if got := NormalizeWorkStatus(input); got != want {
			t.Fatalf("NormalizeWorkStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestWorkStatusTransitions(t *testing.T) {
	allowed := [][2]string{
		{StatusOpen, StatusInProgress},
		{StatusInProgress, StatusCompleted},
		{StatusInProgress, StatusPaymentPending},
		{StatusCompleted, StatusPaymentPending},
		{StatusPaymentPending, StatusClosed},
		{StatusCompleted, StatusClosed},
		{StatusDraft, StatusCancelled},
		{StatusOpen, StatusCancelled},
		{StatusInProgress, StatusCancelled},
	}
	for _, step := range allowed {
		if !CanTransition(step[0], step[1]) {
			t.Fatalf("expected %s → %s", step[0], step[1])
		}
	}

	blocked := [][2]string{
		{StatusOpen, StatusClosed},
		{StatusInProgress, StatusClosed},
		{StatusPaymentPending, StatusCancelled},
		{StatusClosed, StatusCancelled},
		{StatusCancelled, StatusOpen},
		{StatusDraft, StatusInProgress},
		{StatusOpen, StatusPaymentPending},
	}
	for _, step := range blocked {
		if CanTransition(step[0], step[1]) {
			t.Fatalf("did not expect %s → %s", step[0], step[1])
		}
	}
}

func TestCanStartFromOpenAliases(t *testing.T) {
	for _, from := range []string{"published", "PUBLISHED", "open", "OPEN", "assigned", "ASSIGNED"} {
		if !CanTransition(from, StatusInProgress) {
			t.Fatalf("owner should be able to start from %q", from)
		}
	}
}

func TestCreateStatusStaysDraftOrOpen(t *testing.T) {
	if !ValidCreateStatus(StatusDraft) || !ValidCreateStatus(StatusOpen) || !ValidCreateStatus("published") {
		t.Fatal("create must still accept draft, open, and published")
	}
	if ValidCreateStatus(StatusClosed) || ValidCreateStatus(StatusInProgress) {
		t.Fatal("create must not accept lifecycle statuses")
	}
}

func TestWorkerCanCompleteAssignmentAfterOwnerCompletesWork(t *testing.T) {
	for _, status := range []string{StatusOpen, StatusInProgress, StatusPaymentPending, StatusCompleted, "IN_PROGRESS", "WORK_COMPLETED"} {
		if WorkRecordsLocked(status) {
			t.Fatalf("worker should still be able to complete an assignment while work is %q", status)
		}
	}
	for _, status := range []string{StatusClosed, StatusCancelled, "CLOSED", "CANCELLED"} {
		if !WorkRecordsLocked(status) {
			t.Fatalf("worker must not complete an assignment after work is %q", status)
		}
	}
}

func TestHiringModeNormalization(t *testing.T) {
	if NormalizeHiringMode("") != HiringModeIndividual {
		t.Fatal("empty hiring mode should default to INDIVIDUAL")
	}
	if NormalizeHiringMode("contractor") != HiringModeContractor {
		t.Fatal("contractor hiring mode should normalize")
	}
}
