package model

import "testing"

func TestNormalizeWorkStatus(t *testing.T) {
	cases := map[string]string{
		"draft":          StatusDraft,
		"OPEN":           StatusPublished,
		"published":      StatusPublished,
		"ASSIGNED":       StatusAssigned,
		"in_progress":    StatusInProgress,
		"WORK_COMPLETED": StatusWorkCompleted,
		"closed":         StatusClosed,
		"CANCELLED":      StatusCancelled,
	}

	for input, want := range cases {
		if got := NormalizeWorkStatus(input); got != want {
			t.Fatalf("NormalizeWorkStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestWorkStatusTransitions(t *testing.T) {
	allowed := [][2]string{
		{StatusPublished, StatusAssigned},
		{StatusAssigned, StatusPublished},
		{StatusAssigned, StatusInProgress},
		{StatusPublished, StatusInProgress},
		{StatusInProgress, StatusWorkCompleted},
		{StatusWorkCompleted, StatusClosed},
		{StatusDraft, StatusCancelled},
		{StatusPublished, StatusCancelled},
		{StatusAssigned, StatusCancelled},
		{StatusInProgress, StatusCancelled},
	}
	for _, step := range allowed {
		if !CanTransition(step[0], step[1]) {
			t.Fatalf("expected %s → %s", step[0], step[1])
		}
	}

	blocked := [][2]string{
		{StatusPublished, StatusClosed},
		{StatusAssigned, StatusClosed},
		{StatusInProgress, StatusClosed},
		{StatusWorkCompleted, StatusCancelled},
		{StatusClosed, StatusCancelled},
		{StatusCancelled, StatusPublished},
		{StatusDraft, StatusAssigned},
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
			t.Fatalf("accepted worker should be able to start from %q", from)
		}
	}
}

func TestCreateStatusStaysDraftOrPublished(t *testing.T) {
	if !ValidCreateStatus(StatusDraft) || !ValidCreateStatus(StatusPublished) {
		t.Fatal("create must still accept draft and published")
	}
	if ValidCreateStatus(StatusAssigned) || ValidCreateStatus(StatusClosed) {
		t.Fatal("create must not accept lifecycle statuses")
	}
}
