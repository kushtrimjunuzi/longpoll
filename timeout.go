// Copyright (c) 2015-2017. Oleg Sklyar & teris.io. All rights reserved.
// See the LICENSE file in the project root for licensing information.

package longpoll

import (
	"errors"
	"github.com/ventu-io/slf"
	"sync/atomic"
	"time"
)

// Timeout implements a callback mechanism on timeout (along with
// reporting on a buffered channel), which is extendable in time via
// pinging the object. An alive timeout can be dropped at any time,
// in which case the callback will not be executed, but the exit
// will still be reported on the channel.
//
// This extendable Timeout is used for monitoring long polling
// subscriptions here, which would expire if no client asks for data
// within a defined timeout (or timeout extended otherwise).
type Timeout struct {
	lastping  int64
	alive     int32
	report    chan bool
	onTimeout func()
}

// NewTimeout creates and starts a new timeout timer accepting an optional exit handler.
func NewTimeout(timeout time.Duration, onTimeout func()) (*Timeout, error) {
	if timeout <= 0 {
		return nil, errors.New("positive timeout value expected")
	}
	tor := &Timeout{
		alive:     yes,
		report:    make(chan bool, 1),
		onTimeout: onTimeout,
	}
	tor.Ping()
	go tor.handle(int64(timeout))
	return tor, nil
}

// MustNewTimeout acts just like NewTimeout, however, it does not return errors and panics instead.
func MustNewTimeout(timeout time.Duration, onTimeout func()) *Timeout {
	tor, err := NewTimeout(timeout, onTimeout)
	if err == nil {
		return tor
	}
	panic(err)
}

// Ping pings the timeout handler extending it for another timeout duration.
func (tor *Timeout) Ping() {
	if tor.IsAlive() {
		atomic.StoreInt64(&tor.lastping, tor.now())
	}
}

// ReportChan retrieves the timeout reporting channel, which will get a true
// reported on exit (in case of timeout or drop).
func (tor *Timeout) ReportChan() chan bool {
	return tor.report
}

// Drop drops the timeout handler and reports the exit on the reporting channel.
// The drop will take place at most after 1/100th of the timeout and the
// onTimeout handler will not get called.
func (tor *Timeout) Drop() {
	atomic.StoreInt32(&tor.alive, no)
}

// IsAlive verifies if the timeout handler is up and running.
func (tor *Timeout) IsAlive() bool {
	return atomic.LoadInt32(&tor.alive) == yes
}

func (tor *Timeout) handle(timeout int64) {
	hundredth := timeout / 100
	for tor.elapsed() < timeout && tor.IsAlive() {
		time.Sleep(time.Duration(hundredth))
	}
	if tor.IsAlive() {
		atomic.StoreInt32(&tor.alive, no)
		if tor.onTimeout != nil {
			go tor.onTimeout()
		}
	}
	tor.report <- true
}

func (tor *Timeout) elapsed() int64 {
	return tor.now() - atomic.LoadInt64(&tor.lastping)
}

func (tor *Timeout) now() int64 {
	return time.Now().UnixNano()
}
