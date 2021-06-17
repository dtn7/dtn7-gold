// SPDX-FileCopyrightText: 2020, 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"encoding/hex"
	"math"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fsnotify/fsnotify"

	"github.com/dtn7/dtn7-go/pkg/agent"
	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// exchange Bundles between an user and a dtnd over the filesystem.
type exchange struct {
	directory     string
	knownFiles    sync.Map
	websocketConn *agent.WebSocketAgentConnector
	watcher       *fsnotify.Watcher

	closeChan      chan os.Signal
	bundleReadChan chan bpv7.Bundle
}

// startExchange to exchange Bundles between client and a dtnd.
func startExchange(args []string) {
	if len(args) != 3 {
		printUsage()
	}

	var (
		websocketAddr = args[0]
		endpointId    = args[1]
		directory     = args[2]

		err error
	)

	ex := &exchange{
		directory:      directory,
		closeChan:      make(chan os.Signal),
		bundleReadChan: make(chan bpv7.Bundle),
	}

	signal.Notify(ex.closeChan, os.Interrupt)

	if ex.websocketConn, err = agent.NewWebSocketAgentConnector(websocketAddr, endpointId); err != nil {
		printFatal(err, "Starting WebSocketAgentConnector errored")
	}

	if ex.watcher, err = fsnotify.NewWatcher(); err != nil {
		printFatal(err, "Starting file watcher errored")
	}
	if err = ex.watcher.Add(directory); err != nil {
		printFatal(err, "Adding directory to file watcher errored")
	}

	go ex.handleBundleRead()
	ex.handler()
}

// cleanFilepath creates a relative path from the initial path to a new file's path.
func (ex *exchange) cleanFilepath(f string) string {
	if rel, err := filepath.Rel(ex.directory, f); err != nil {
		log.WithField("path", f).WithError(err).Fatal("Failed to clean file path")
		return ""
	} else {
		return rel
	}
}

func (ex *exchange) handler() {
	defer func() {
		_ = ex.watcher.Close()
		ex.websocketConn.Close()
	}()

	for {
		select {
		case <-ex.closeChan:
			log.Info("Received interrupt signal")
			return

		case e, ok := <-ex.watcher.Events:
			if !ok {
				log.Error("fsnotify's Event channel was closed")
				return
			}

			if _, ok := ex.knownFiles.Load(ex.cleanFilepath(e.Name)); ok {
				log.WithField("file", e.Name).Debug("Skipping file; already known")
				continue
			}

			if e.Op&fsnotify.Create == 0 {
				log.WithFields(log.Fields{
					"file":      e.Name,
					"operation": e.Op.String(),
				}).Debug("Ignoring fsnotify event")
				continue
			}

			ex.readNewFile(e)

		case err, ok := <-ex.watcher.Errors:
			if !ok {
				log.Error("fsnotify's Errors channel was closed")
				return
			}

			log.WithError(err).Error("fsnotify errored")
			return

		case b, ok := <-ex.bundleReadChan:
			if !ok {
				log.Error("Bundle reader channel was closed")
				return
			}

			filePath := path.Join(ex.directory, hex.EncodeToString([]byte(b.ID().String())))
			logger := log.WithFields(log.Fields{
				"bundle": b.ID(),
				"file":   filePath,
			})

			if f, err := os.Create(filePath); err != nil {
				logger.WithError(err).Error("Creating file errored")
				return
			} else if err := b.MarshalCbor(f); err != nil {
				logger.WithError(err).Error("Marshalling Bundle errored")
			} else if err := f.Close(); err != nil {
				logger.WithError(err).Error("Closing file errored")
			}

			ex.knownFiles.Store(ex.cleanFilepath(filePath), struct{}{})

			logger.Info("Saved received Bundle")
		}
	}
}

func (ex *exchange) readNewFile(e fsnotify.Event) {
	for i := 0; i < 5; i++ {
		var b bpv7.Bundle

		if f, err := os.Open(e.Name); err != nil {
			log.WithError(err).WithField("file", e.Name).Warn("Opening file errored, retrying..")
		} else if err := b.UnmarshalCbor(f); err != nil {
			log.WithError(err).WithField("file", e.Name).Warn("Unmarshalling Bundle errored, retrying..")
		} else if err := f.Close(); err != nil {
			log.WithError(err).WithField("file", e.Name).Warn("Closing file errored, retrying..")
		} else if err := ex.websocketConn.WriteBundle(b); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"file":   e.Name,
				"bundle": b.ID().String(),
			}).Error("Sending Bundle errored")
			return
		} else {
			log.WithError(err).WithFields(log.Fields{
				"file":   e.Name,
				"bundle": b.ID().String(),
			}).Info("Sent Bundle")
			return
		}

		time.Sleep(time.Duration(math.Pow(2, float64(i))) * 100 * time.Millisecond)
	}

	log.WithField("file", e.Name).Error("Failed to process file, giving up.")
}

func (ex *exchange) handleBundleRead() {
	for {
		if b, err := ex.websocketConn.ReadBundle(); err != nil {
			log.WithError(err).Error("Reading Bundle errored")

			close(ex.bundleReadChan)
			return
		} else {
			ex.bundleReadChan <- b
		}
	}
}
