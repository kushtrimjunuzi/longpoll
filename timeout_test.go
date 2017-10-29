// Copyright (c) 2015-2017. Oleg Sklyar & teris.io. All rights reserved.
// See the LICENSE file in the project root for licensing information.

package longpoll_test

import (
	"testing"
	"time"
	"github.com/teris-io/longpoll"
)

func TestTimeout_onNewTimeout_success(t *testing.T) {
	tor, err := longpoll.NewTimeout(time.Minute, nil)
	if err != nil {
		t.Error("no error expected")
	}
	if !tor.IsAlive() {
		t.Error("tor not alive on start")
	}
}

func TestTimeout_onNewTimeout_whenZeroDuration_error(t *testing.T) {
	var tm time.Duration
	_, err := longpoll.NewTimeout(tm, nil)
	if err == nil {
		t.Error("error expected")
	}
}

func TestTimeout_onMustNewTimeout_success(t *testing.T) {
	tor := longpoll.MustNewTimeout(time.Minute, nil)
	if !tor.IsAlive() {
		t.Error("tor not alive on start")
	}
}

func TestTimeout_onMustNewTimeout_whenZeroDuration_panics(t *testing.T) {
	var tm time.Duration
	defer func() {
		if r := recover(); r == nil {
			t.Error("panic expected")
		}
	}()
	longpoll.MustNewTimeout(tm, nil)
}

func TestTimeout_onNoPing_expires(t *testing.T) {
	timeout := 200 * time.Millisecond
	tolerance := 50 * time.Millisecond

	start := time.Now()
	var end time.Time
	tor := longpoll.MustNewTimeout(timeout, func() {
		end = time.Now()
	})
	if !tor.IsAlive() {
		t.Errorf("tor not alive on start")
	}
	time.Sleep(timeout + tolerance)
	if tor.IsAlive() {
		t.Errorf("tor alive after timeout")
	}
	if end.Sub(start) < timeout {
		t.Errorf("timeout too early")
	}
	if end.Sub(start) > timeout+tolerance {
		t.Errorf("timeout too late")
	}
}

func TestTimeout_onPing_extends(t *testing.T) {
	timeout := 200 * time.Millisecond
	tolerance := 50 * time.Millisecond

	start := time.Now()
	var end time.Time
	tor := longpoll.MustNewTimeout(timeout, func() {
		end = time.Now()
	})

	time.Sleep(timeout - tolerance)
	tor.Ping()

	time.Sleep(tolerance + tolerance)
	if !tor.IsAlive() {
		t.Errorf("tor not after ping")
	}

	time.Sleep(timeout)
	if tor.IsAlive() {
		t.Errorf("tor alive after timeout")
	}

	if end.Sub(start) < timeout+timeout-tolerance {
		t.Errorf("timeout too early")
	}
	if end.Sub(start) > timeout+timeout {
		t.Errorf("timeout too late")
	}
}

func TestTimeout_onExpiry_callsHandler_andReportsOnChannel(t *testing.T) {
	timeout := 200 * time.Millisecond
	tolerance := 50 * time.Millisecond

	failed := true
	tor := longpoll.MustNewTimeout(timeout, func() {
		failed = false
	})
	time.Sleep(timeout + tolerance)
	if failed {
		t.Errorf("onTimeout handler not called")
	}
	select {
	case <-tor.ReportChan(): // all good, ignore
	default:
		t.Errorf("timeout not reported on channel")
	}
}

func TestTimeout_onNoHandler_reportsOnChannelOnExpiry(t *testing.T) {
	timeout := 200 * time.Millisecond
	tolerance := 50 * time.Millisecond

	tor := longpoll.MustNewTimeout(timeout, nil)
	if !tor.IsAlive() {
		t.Errorf("tor not alive on start")
	}
	time.Sleep(timeout + tolerance)
	if tor.IsAlive() {
		t.Errorf("tor alive after timeout")
	}
	select {
	case <-tor.ReportChan(): // all good, ignore
	default:
		t.Errorf("timeout not reported on channel")
	}
}

func TestTimeout_onDrop_skipsHandler_butReportsOnChannel(t *testing.T) {
	timeout := 200 * time.Millisecond
	tolerance := 50 * time.Millisecond

	failed := false
	tor := longpoll.MustNewTimeout(timeout, func() {
		failed = true
	})
	if !tor.IsAlive() {
		t.Errorf("tor not alive on start")
	}
	time.Sleep(tolerance)
	if !tor.IsAlive() {
		t.Errorf("tor not alive on start")
	}
	tor.Drop()
	time.Sleep(tolerance)
	if tor.IsAlive() {
		t.Errorf("tor alive after drop and wait")
	}
	if failed {
		t.Errorf("handler called on drop")
	}
	select {
	case <-tor.ReportChan(): // all good, ignore
	default:
		t.Errorf("timeout or drop not reported on channel")
	}
}
