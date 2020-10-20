// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type cronjob struct {
	task      func()
	interval  time.Duration
	nextEvent time.Time
}

// Cron manages different jobs which require interval based execution.
type Cron struct {
	jobs  map[string]*cronjob
	mutex sync.Mutex

	stopSyn chan struct{}
	stopAck chan struct{}
}

// NewCron creates and starts an empty Cron instance.
func NewCron() *Cron {
	cron := &Cron{
		jobs:    make(map[string]*cronjob),
		stopSyn: make(chan struct{}),
		stopAck: make(chan struct{}),
	}

	go cron.loop()

	return cron
}

func (cron *Cron) loop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cron.stopSyn:
			close(cron.stopAck)
			return

		case t := <-ticker.C:
			cron.fire(t)
		}
	}
}

func (cron *Cron) fire(t time.Time) {
	cron.mutex.Lock()
	defer cron.mutex.Unlock()

	for name, job := range cron.jobs {
		if job.nextEvent.After(t) {
			continue
		}

		job.nextEvent = job.nextEvent.Add(job.interval)
		go job.task()

		log.WithFields(log.Fields{
			"job":        name,
			"interval":   job.interval,
			"next_event": job.nextEvent,
		}).Debug("Cron executed job")
	}
}

// Stop this Cron. This method is only allowed to be called once.
func (cron *Cron) Stop() {
	close(cron.stopSyn)
	<-cron.stopAck
}

// Register a new task by its name, function and interval. The interval must be
// at least one second. The function will be executed in a new Goroutine and
// must be thread-safe.
func (cron *Cron) Register(name string, task func(), interval time.Duration) error {
	cron.mutex.Lock()
	defer cron.mutex.Unlock()

	if _, exists := cron.jobs[name]; exists {
		return fmt.Errorf("A job named %s is already registered", name)
	}

	if interval < time.Second {
		return fmt.Errorf("Given interval %v is shorter than a second", interval)
	}

	job := &cronjob{
		task:      task,
		interval:  interval,
		nextEvent: time.Now().Add(interval),
	}
	cron.jobs[name] = job

	return nil
}

// Unregister a task by its name.
func (cron *Cron) Unregister(name string) {
	cron.mutex.Lock()
	defer cron.mutex.Unlock()

	delete(cron.jobs, name)
}
