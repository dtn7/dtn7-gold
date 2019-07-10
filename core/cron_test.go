package core

import (
	"sync"
	"testing"
	"time"
)

func TestCronSimple(t *testing.T) {
	var t1counter, t2counter, t3counter, t4counter = 0, 0, 0, 0
	var t1mutex, t2mutex, t3mutex, t4mutex sync.Mutex

	inct1 := func() {
		t1mutex.Lock()
		t1counter += 1
		t1mutex.Unlock()
	}

	inct2 := func() {
		t2mutex.Lock()
		t2counter += 1
		t2mutex.Unlock()
	}

	inct3 := func() {
		t3mutex.Lock()
		t3counter += 1
		t3mutex.Unlock()
	}

	inct4 := func() {
		t4mutex.Lock()
		t4counter += 1
		t4mutex.Unlock()
	}

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

	time.Sleep(5*time.Second + 250*time.Millisecond)
	cron.Unregister("t4")

	time.Sleep(5*time.Second + 250*time.Millisecond)
	cron.Stop()

	t1mutex.Lock()
	t2mutex.Lock()
	t3mutex.Lock()
	t4mutex.Lock()

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

	t1mutex.Unlock()
	t2mutex.Unlock()
	t3mutex.Unlock()
	t4mutex.Unlock()
}
