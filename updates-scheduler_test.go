package main

import (
	"context"
	"errors"
	"testing"

	"akproxy/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

type fakeUpdateClient struct {
	state       updater.State
	release     *updater.Release
	checkErr    error
	downloadErr error
	checks      int
	downloads   int
}

func (f *fakeUpdateClient) State() updater.State { return f.state }

func (f *fakeUpdateClient) Check(context.Context) (*updater.Release, error) {
	f.checks++
	if f.checkErr != nil {
		f.state = updater.StateError
		return nil, f.checkErr
	}
	if f.release == nil {
		f.state = updater.StateUpToDate
	} else {
		f.state = updater.StateAvailable
	}
	return f.release, nil
}

func (f *fakeUpdateClient) DownloadAndInstall(context.Context) error {
	f.downloads++
	if f.downloadErr != nil {
		f.state = updater.StateError
		return f.downloadErr
	}
	f.state = updater.StateReady
	return nil
}

func TestUpdateSchedulerRecurringAndDownloadModes(t *testing.T) {
	oldVersion := version
	version = "1.0.0"
	defer func() { version = oldVersion }()

	client := &fakeUpdateClient{release: &updater.Release{Version: "1.0.1"}}
	s := &updateScheduler{updater: client, prefs: desktop.DefaultAppPrefs(), emit: func(string, any) {}}
	s.Start()
	firstTimer := s.timer
	s.tick()
	if client.checks != 1 || client.downloads != 1 || s.timer == firstTimer {
		t.Fatalf("first tick: checks=%d downloads=%d timer rearmed=%t", client.checks, client.downloads, s.timer != firstTimer)
	}
	client.state = updater.StateIdle
	s.tick()
	if client.checks != 2 || client.downloads != 2 {
		t.Fatalf("second tick: checks=%d downloads=%d", client.checks, client.downloads)
	}
	s.Stop()
	s.tick()
	if client.checks != 2 || s.timer != nil {
		t.Fatal("stopped scheduler checked or retained its timer")
	}

	client = &fakeUpdateClient{release: &updater.Release{Version: "1.0.1"}}
	var notices int
	s = &updateScheduler{
		updater: client,
		prefs:   desktop.AppPrefs{AutoCheck: true, CheckIntervalHours: 1},
		emit: func(event string, _ any) {
			if event == "updates:new-release" {
				notices++
			}
		},
	}
	s.Start()
	s.tick()
	if notices != 1 || client.downloads != 0 || s.pending == nil {
		t.Fatalf("find-only check: notices=%d downloads=%d pending=%v", notices, client.downloads, s.pending)
	}
	client.downloadErr = errors.New("download failed")
	if err := s.DownloadPendingUpdate(); err == nil || s.pending == nil {
		t.Fatal("failed download lost its retryable release")
	}
	client.downloadErr = nil
	if err := s.DownloadPendingUpdate(); err != nil || s.pending != nil {
		t.Fatalf("retry download: %v pending=%v", err, s.pending)
	}
	s.Stop()
}

func TestUpdateSchedulerDevelopmentBuildDoesNotCheck(t *testing.T) {
	oldVersion := version
	version = "dev"
	defer func() { version = oldVersion }()
	client := &fakeUpdateClient{}
	s := &updateScheduler{updater: client, prefs: desktop.DefaultAppPrefs(), emit: func(string, any) {}}
	s.Start()
	s.tick()
	if client.checks != 0 || s.timer != nil {
		t.Fatal("development build scheduled an automatic check")
	}
}

func TestUpdateSchedulerBackgroundCheckErrorIsReported(t *testing.T) {
	oldVersion := version
	version = "1.0.0"
	defer func() { version = oldVersion }()
	client := &fakeUpdateClient{checkErr: errors.New("check failed")}
	var reported string
	s := &updateScheduler{
		updater: client,
		prefs:   desktop.DefaultAppPrefs(),
		emit: func(event string, data any) {
			if event == "updates:check-error" {
				reported, _ = data.(string)
			}
		},
	}
	s.Start()
	s.tick()
	if reported != "check failed" || s.Status().LastCheckResult != "检查失败" {
		t.Fatalf("check failure was not reported: %q, %+v", reported, s.Status())
	}
	s.Stop()
}
