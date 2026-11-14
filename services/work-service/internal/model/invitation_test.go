package model

import (
	"testing"
	"time"
)

func TestVisibilityDefaultsToPublic(t *testing.T) {
	if NormalizeVisibility("") != VisibilityPublic {
		t.Fatal("missing visibility must stay public")
	}
	if NormalizeVisibility("PUBLIC") != VisibilityPublic {
		t.Fatal("public spelling should normalize")
	}
	if !IsInvitationOnly("invitation_only") {
		t.Fatal("invitation_only should block public applications")
	}
	if IsInvitationOnly("") || IsInvitationOnly("public") {
		t.Fatal("public work must remain applicable")
	}
	if NormalizeVisibility("secret") != "" {
		t.Fatal("unknown visibility must be rejected")
	}
}

func TestRecruitmentEligibility(t *testing.T) {
	workType := "harvesting"
	needed := 3
	work := &WorkRecord{
		Status:          StatusOpen,
		HiringMode:      HiringModeIndividual,
		WorkTypeID:      &workType,
		WorkersNeeded:   &needed,
		WorkersSelected: 1,
	}
	worker := &RecruitmentWorker{
		UserID:             "worker-1",
		ProfileCompleted:   true,
		VerificationStatus: "verified",
		WorkerType:         ApplicantTypeIndividual,
		WorkTypeIDs:        []string{workType},
	}

	eligible, reason, seats := RecruitmentEligibility(RecruitmentEligibilityInput{Work: work, Worker: worker})
	if !eligible || reason != "" || seats != 1 {
		t.Fatalf("expected eligible individual, got eligible=%v reason=%s seats=%d", eligible, reason, seats)
	}

	worker.VerificationStatus = "pending"
	eligible, reason, _ = RecruitmentEligibility(RecruitmentEligibilityInput{Work: work, Worker: worker})
	if eligible || reason != InvitationReasonNotVerified {
		t.Fatalf("unverified worker should be rejected, got %v %s", eligible, reason)
	}

	worker.VerificationStatus = "verified"
	worker.WorkTypeIDs = []string{"painting"}
	eligible, reason, _ = RecruitmentEligibility(RecruitmentEligibilityInput{Work: work, Worker: worker})
	if eligible || reason != InvitationReasonWorkTypeMismatch {
		t.Fatalf("mismatched work type should be rejected, got %v %s", eligible, reason)
	}

	worker.WorkTypeIDs = []string{workType}
	eligible, reason, _ = RecruitmentEligibility(RecruitmentEligibilityInput{
		Work:          work,
		Worker:        worker,
		HasAssignment: true,
	})
	if eligible || reason != InvitationReasonAlreadyAssigned {
		t.Fatalf("assigned worker should be rejected, got %v %s", eligible, reason)
	}

	eligible, reason, _ = RecruitmentEligibility(RecruitmentEligibilityInput{
		Work:             work,
		Worker:           worker,
		HasPendingInvite: true,
	})
	if eligible || reason != InvitationReasonInvitationPending {
		t.Fatalf("pending invite should be rejected, got %v %s", eligible, reason)
	}

	work.WorkersSelected = 3
	eligible, reason, _ = RecruitmentEligibility(RecruitmentEligibilityInput{Work: work, Worker: worker})
	if eligible || reason != InvitationReasonNoRemainingCapacity {
		t.Fatalf("full work should be rejected, got %v %s", eligible, reason)
	}

	work.WorkersSelected = 0
	work.Status = StatusInProgress
	eligible, reason, _ = RecruitmentEligibility(RecruitmentEligibilityInput{Work: work, Worker: worker})
	if eligible || reason != InvitationReasonWorkNotOpen {
		t.Fatalf("in-progress work should not be recruitable, got %v %s", eligible, reason)
	}
}

func TestContractorRecruitmentRules(t *testing.T) {
	workType := "masonry"
	needed := 5
	work := &WorkRecord{
		Status:        StatusOpen,
		HiringMode:    HiringModeContractor,
		WorkTypeID:    &workType,
		WorkersNeeded: &needed,
	}
	worker := &RecruitmentWorker{
		UserID:             "crew-1",
		ProfileCompleted:   true,
		VerificationStatus: "verified",
		WorkerType:         ApplicantTypeContractor,
		CrewSize:           4,
		WorkTypeIDs:        []string{workType},
	}

	eligible, _, seats := RecruitmentEligibility(RecruitmentEligibilityInput{Work: work, Worker: worker})
	if !eligible || seats != 4 {
		t.Fatalf("contractor crew size should be the seat count, eligible=%v seats=%d", eligible, seats)
	}

	eligible, reason, _ := RecruitmentEligibility(RecruitmentEligibilityInput{
		Work:                work,
		Worker:              worker,
		HasActiveContractor: true,
	})
	if eligible || reason != InvitationReasonContractorAlreadyAssigned {
		t.Fatalf("second contractor should be rejected, got %v %s", eligible, reason)
	}

	individual := *worker
	individual.WorkerType = ApplicantTypeIndividual
	eligible, reason, _ = RecruitmentEligibility(RecruitmentEligibilityInput{Work: work, Worker: &individual})
	if eligible || reason != InvitationReasonHiringModeMismatch {
		t.Fatalf("individual on contractor work should be rejected, got %v %s", eligible, reason)
	}
}

func TestInvitationStatusTransitionsAndExpiry(t *testing.T) {
	for _, next := range []string{
		InvitationStatusAccepted,
		InvitationStatusDeclined,
		InvitationStatusCancelled,
		InvitationStatusExpired,
	} {
		if !InvitationTransitionAllowed(InvitationStatusPending, next) {
			t.Fatalf("pending invitation should move to %s", next)
		}
	}
	if InvitationTransitionAllowed(InvitationStatusAccepted, InvitationStatusCancelled) {
		t.Fatal("accepted invitation must not be cancelled through the invitation transition")
	}
	if InvitationTransitionAllowed(InvitationStatusDeclined, InvitationStatusAccepted) {
		t.Fatal("declined invitation must not be accepted")
	}

	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	if !InvitationExpired(InvitationStatusPending, now.Add(-time.Minute), now) {
		t.Fatal("past pending invitation should be expired")
	}
	if InvitationExpired(InvitationStatusPending, now.Add(time.Hour), now) {
		t.Fatal("future invitation should stay pending")
	}
	if InvitationExpired(InvitationStatusAccepted, now.Add(-time.Hour), now) {
		t.Fatal("accepted invitation should not be rewritten as expired")
	}
}

func TestOfferedPaymentSnapshotAndAssignment(t *testing.T) {
	rate := 800.0
	basis := "daily"
	unit := "day"
	work := &WorkRecord{
		ID:         "work-1",
		RateBasis:  &basis,
		RateUnit:   &unit,
		BudgetRate: &rate,
	}
	payment, err := ResolveOfferedPayment(work, nil)
	if err != nil {
		t.Fatal(err)
	}
	changed := 1200.0
	work.BudgetRate = &changed

	invitation := &WorkInvitation{
		ID:             "invite-1",
		WorkerID:       "worker-1",
		ApplicantType:  ApplicantTypeIndividual,
		SeatCount:      1,
		OfferedPayment: payment,
	}
	assignment := BuildAssignmentFromInvitation(work, invitation)
	if assignment.Status != AssignmentStatusAssigned {
		t.Fatalf("accepted invitation should create an assigned row, got %s", assignment.Status)
	}
	if assignment.InvitationID != invitation.ID || assignment.ApplicationID != "" {
		t.Fatal("assignment should point at the invitation and leave the application path empty")
	}
	if assignment.Quotation.Amount != 800 {
		t.Fatalf("agreed rate must stay at the snapshotted offer, got %v", assignment.Quotation.Amount)
	}
	if assignment.AgreedPayment.RateMinor != RupeesToMinor(800) {
		t.Fatalf("agreed payment should be frozen from the invitation, got %d", assignment.AgreedPayment.RateMinor)
	}
	if assignment.AssignedWorkerCount != 1 {
		t.Fatal("individual invitation should take one seat")
	}
	if ShouldAcceptAfterSeatClaim(nil) {
		t.Fatal("a failed seat claim must leave the invitation pending")
	}
	if !ShouldAcceptAfterSeatClaim(work) {
		t.Fatal("a successful seat claim should allow acceptance")
	}
}

func TestPendingInvitationsDoNotReduceRemaining(t *testing.T) {
	if RemainingWorkers(5, 2) != 3 {
		t.Fatal("remaining capacity is required workers minus confirmed assignments")
	}
}
