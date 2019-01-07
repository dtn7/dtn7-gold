package bundle

import (
	"strings"
	"testing"
	"time"
)

func TestDtnTime(t *testing.T) {
	var epoch DtnTime = 0
	var ttime time.Time = epoch.Time()

	if !strings.HasPrefix(ttime.String(), "2000-01-01 00:00:00") {
		t.Errorf("Time does not represent 2000-01-01, instead: %v", ttime.String())
	}

	if _, offset := ttime.Zone(); offset != 0 {
		t.Errorf("Time is not located in UTC, instead: %d", offset)
	}

	var epoch2 DtnTime = DtnTimeFromTime(ttime)
	if epoch != epoch2 {
		t.Errorf("Converting time.Time back to DtnTime diverges: %d", epoch2)
	}

	durr, _ := time.ParseDuration("48h30m")
	ttime = ttime.Add(durr)
	if epoch+((48*60+30)*60) != DtnTimeFromTime(ttime) {
		t.Errorf("Converting time.Time back to DTNTime diverges: %d", epoch2)
	}
}
