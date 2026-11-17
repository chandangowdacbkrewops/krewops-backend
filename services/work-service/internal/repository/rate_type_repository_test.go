package repository

import "testing"

func TestRateOptionAllowed(t *testing.T) {
	t.Parallel()

	options := []RateOption{
		{RateBasis: "per_time", RateUnit: "day"},
		{RateBasis: "negotiable", RateUnit: ""},
	}

	if !RateOptionAllowed(options, "per_time", "day") {
		t.Fatal("expected per_time+day to be allowed")
	}
	if !RateOptionAllowed(options, "negotiable", "") {
		t.Fatal("expected negotiable to be allowed")
	}
	if RateOptionAllowed(options, "fixed", "job") {
		t.Fatal("did not expect fixed+job to be allowed")
	}
}
