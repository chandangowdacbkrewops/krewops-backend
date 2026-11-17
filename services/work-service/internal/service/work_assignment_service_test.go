package service

import (
	"testing"
	"time"

	"github.com/chandangowdacbkrewops/krewops-backend/services/work-service/internal/model"
)

func TestStartWithPartialWorkerCapacity(t *testing.T) {
	needed := 5
	work := &model.WorkRecord{
		HiringMode:      model.HiringModeIndividual,
		WorkersNeeded:   &needed,
		WorkersSelected: 3,
		Status:          model.StatusOpen,
	}
	assignments := []model.WorkAssignment{
		{Status: model.AssignmentStatusAssigned, AssignmentType: model.AssignmentTypeIndividual},
		{Status: model.AssignmentStatusAssigned, AssignmentType: model.AssignmentTypeIndividual},
		{Status: model.AssignmentStatusAssigned, AssignmentType: model.AssignmentTypeIndividual},
	}
	if err := validateStartAssignments(work, assignments); err != nil {
		t.Fatalf("owner should start with a partial crew: %v", err)
	}
}

func TestStartRequiresAtLeastOneAssignment(t *testing.T) {
	work := &model.WorkRecord{HiringMode: model.HiringModeIndividual, Status: model.StatusOpen}
	if err := validateStartAssignments(work, nil); err != ErrNoActiveAssignment {
		t.Fatalf("got %v", err)
	}
}

func TestMixedCrewCanStart(t *testing.T) {
	work := &model.WorkRecord{HiringMode: model.HiringModeContractor, Status: model.StatusOpen}
	if err := validateStartAssignments(work, nil); err != ErrNoActiveAssignment {
		t.Fatalf("got %v", err)
	}
	if err := validateStartAssignments(work, []model.WorkAssignment{
		{Status: model.AssignmentStatusAssigned, AssignmentType: model.AssignmentTypeContractor, AssignedWorkerCount: 6},
		{Status: model.AssignmentStatusAssigned, AssignmentType: model.AssignmentTypeIndividual},
	}); err != nil {
		t.Fatalf("mixed contractor and individual assignments should start: %v", err)
	}
}

func TestMixedCrewRecordsFollowAssignmentType(t *testing.T) {
	assignments := []model.WorkAssignment{
		{
			ID:             "individual-1",
			Status:         model.AssignmentStatusAssigned,
			AssignmentType: model.AssignmentTypeIndividual,
		},
		{
			ID:                  "contractor-1",
			Status:              model.AssignmentStatusAssigned,
			AssignmentType:      model.AssignmentTypeContractor,
			AssignedWorkerCount: 6,
		},
	}
	individuals := model.IndividualRecordAssignments(assignments)
	if len(individuals) != 1 || individuals[0].ID != "individual-1" {
		t.Fatalf("individual assignments belong on the daily worker list, got %#v", individuals)
	}
	contractor := model.PickContractorAssignment(assignments)
	if contractor == nil || contractor.ID != "contractor-1" {
		t.Fatal("contractor assignments belong on contractor day records")
	}
}

func TestBuildAssignmentCopiesOfferedCount(t *testing.T) {
	needed := 10
	work := &model.WorkRecord{ID: "work-1", WorkersNeeded: &needed}
	application := &model.WorkApplication{
		ID:                 "app-1",
		WorkerID:           "contractor-1",
		ApplicantType:      model.ApplicantTypeContractor,
		OfferedWorkerCount: 6,
	}
	assignment := buildAssignmentFromAcceptance(work, application)
	if assignment.AssignmentType != model.AssignmentTypeContractor {
		t.Fatalf("type = %s", assignment.AssignmentType)
	}
	if assignment.AssignedWorkerCount != 6 || assignment.CommittedWorkerCount != 6 {
		t.Fatalf("assigned=%d committed=%d", assignment.AssignedWorkerCount, assignment.CommittedWorkerCount)
	}
}

func TestAllObligationsMustBeResolvedToClose(t *testing.T) {
	if allObligationsResolved([]model.PaymentObligation{
		{PayableAmountMinor: 150000, Status: model.PaymentStatusPaid},
		{PayableAmountMinor: 125000, Status: model.PaymentStatusDue},
	}) {
		t.Fatal("unpaid worker should block close")
	}
	if !allObligationsResolved([]model.PaymentObligation{
		{PayableAmountMinor: 150000, Status: model.PaymentStatusPaid},
		{PayableAmountMinor: 125000, Status: model.PaymentStatusWaived},
	}) {
		t.Fatal("paid and waived obligations should allow close")
	}
}

func TestAllActiveAssignmentsCompleted(t *testing.T) {
	if allActiveAssignmentsCompleted(nil) {
		t.Fatal("empty crew is not complete")
	}
	if allActiveAssignmentsCompleted([]model.WorkAssignment{
		{Status: model.AssignmentStatusSubmitted},
		{Status: model.AssignmentStatusInProgress},
	}) {
		t.Fatal("one incomplete assignment should not count as complete")
	}
	if !allActiveAssignmentsCompleted([]model.WorkAssignment{
		{Status: model.AssignmentStatusSubmitted},
		{Status: model.AssignmentStatusApproved},
	}) {
		t.Fatal("submitted and approved assignments should count as complete")
	}
}

func TestAllAssignmentsHavePaidRecord(t *testing.T) {
	paid := map[string]bool{"a": true}
	if allAssignmentsHavePaidRecord([]model.WorkAssignment{{ID: "a"}, {ID: "b"}}, paid) {
		t.Fatal("missing paid record should block close")
	}
	if !allAssignmentsHavePaidRecord([]model.WorkAssignment{{ID: "a"}, {ID: "b"}}, map[string]bool{"a": true, "b": true}) {
		t.Fatal("all paid should allow owner close")
	}
}

func TestAllPaymentReceiptsConfirmed(t *testing.T) {
	obligations := []model.PaymentObligation{
		{ID: "ob-1", PayableAmountMinor: 80000, PaidAmountMinor: 80000, Status: model.PaymentStatusPaid},
	}
	if allPaymentReceiptsConfirmed(obligations, nil) {
		t.Fatal("paid obligation without a confirmed receipt should not close")
	}
	if allPaymentReceiptsConfirmed(obligations, []model.PaymentTransaction{
		{PaymentObligationID: "ob-1", ReceiptStatus: ""},
	}) {
		t.Fatal("unconfirmed receipt should not close the posting")
	}
	if !allPaymentReceiptsConfirmed(obligations, []model.PaymentTransaction{
		{PaymentObligationID: "ob-1", ReceiptStatus: model.PaymentReceiptConfirmed},
	}) {
		t.Fatal("paid and confirmed receipt should allow the posting to close")
	}
	if !allPaymentReceiptsConfirmed([]model.PaymentObligation{
		{ID: "ob-2", PayableAmountMinor: 80000, PaidAmountMinor: 0, Status: model.PaymentStatusWaived},
	}, nil) {
		t.Fatal("waived obligation does not need a receipt")
	}
}

func TestTransactionToLegacyPaymentCopiesReceipt(t *testing.T) {
	confirmed := time.Now().UTC()
	message := "short cash"
	txn := &model.PaymentTransaction{
		ID:             "txn-1",
		WorkID:         "work-1",
		AssignmentID:   "asg-1",
		AmountMinor:    150000,
		PaymentMethod:  model.PaymentMethodCash,
		ReceiptStatus:  model.PaymentReceiptConfirmed,
		DisputeMessage: &message,
		PaidAt:         confirmed,
		ConfirmedAt:    &confirmed,
	}
	got := transactionToLegacyPayment(txn, "worker-1")
	if got == nil {
		t.Fatal("expected mapped payment")
	}
	if got.ID != "txn-1" || got.AssignmentID != "asg-1" || got.WorkerID != "worker-1" {
		t.Fatalf("identity mismatch: %+v", got)
	}
	if got.ReceiptStatus != model.PaymentReceiptConfirmed {
		t.Fatalf("receipt status = %q", got.ReceiptStatus)
	}
	if got.ConfirmedAt == nil || !got.ConfirmedAt.Equal(confirmed) {
		t.Fatal("confirmed_at should copy from the transaction")
	}
}
