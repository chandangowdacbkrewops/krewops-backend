package model

import "testing"

func TestRateOptionLabel(t *testing.T) {
	t.Parallel()

	day := "day"
	kg := "kg"
	job := "job"

	cases := []struct {
		basis string
		unit  *string
		want  string
	}{
		{basis: "per_time", unit: &day, want: "Per person / day"},
		{basis: "per_output", unit: &kg, want: "Per kg"},
		{basis: "fixed", unit: &job, want: "Fixed contract"},
		{basis: "negotiable", unit: nil, want: "Negotiable"},
	}

	for _, tc := range cases {
		if got := RateOptionLabel(tc.basis, tc.unit); got != tc.want {
			t.Fatalf("RateOptionLabel(%s, %v) = %q, want %q", tc.basis, tc.unit, got, tc.want)
		}
	}
}

func TestRateOptionMatches(t *testing.T) {
	t.Parallel()

	day := "day"
	options := []RateOption{
		{RateBasis: "per_time", RateUnit: &day},
		{RateBasis: "negotiable"},
	}

	if !RateOptionMatches(options, "per_time", "day") {
		t.Fatal("expected per_time+day to match")
	}
	if !RateOptionMatches(options, "negotiable", "") {
		t.Fatal("expected negotiable with empty unit to match")
	}
	if RateOptionMatches(options, "fixed", "job") {
		t.Fatal("did not expect fixed+job to match")
	}
}
