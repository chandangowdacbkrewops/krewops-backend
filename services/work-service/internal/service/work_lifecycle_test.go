package service

import (
	"testing"
	"time"
)

func TestStartAllowedOnOrAfterSchedule(t *testing.T) {
	now := time.Date(2026, 10, 11, 15, 0, 0, 0, time.UTC)
	today := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)
	yesterday := today.AddDate(0, 0, -1)

	if !startAllowed(nil, now) {
		t.Fatal("missing start date should allow start")
	}
	if !startAllowed(&today, now) || !startAllowed(&yesterday, now) {
		t.Fatal("start should be allowed on or after the scheduled date")
	}
	if startAllowed(&tomorrow, now) {
		t.Fatal("start should wait until the scheduled date")
	}
}
