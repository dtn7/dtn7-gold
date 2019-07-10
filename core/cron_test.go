package core

import (
	"testing"
	"time"
)

func TestCronSimple(t *testing.T) {
	var t1counter, t2counter, t3counter, t4counter = 0, 0, 0, 0

	inct1 := func() { t1counter += 1 }
	inct2 := func() { t2counter += 1 }
	inct3 := func() { t3counter += 1 }
	inct4 := func() { t4counter += 1 }

	cron := NewCron()

	if err := cron.Register("t1", inct1, time.Second*2); err != nil {
		t.Fatal(err)
	}
	if err := cron.Register("t2", inct2, time.Second*3); err != nil {
		t.Fatal(err)
	}
	if err := cron.Register("t3", inct3, time.Second*4); err != nil {
		t.Fatal(err)
	}
	if err := cron.Register("t4", inct4, time.Second*2); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 5)
	cron.Unregister("t4")

	time.Sleep(time.Second * 5)
	cron.Stop()

	if t1counter != 5 {
		t.Fatalf("t1 should be 5 instead of %d", t1counter)
	}
	if t2counter != 3 {
		t.Fatalf("t2 should be 3 instead of %d", t2counter)
	}
	if t3counter != 2 {
		t.Fatalf("t3 should be 2 instead of %d", t3counter)
	}
	if t4counter != 2 {
		t.Fatalf("t4 should be 2 instead of %d", t4counter)
	}
}
