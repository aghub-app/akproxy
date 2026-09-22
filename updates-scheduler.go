package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"akproxy/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

// UpdatePrefsStatus is what the 关于 tab shows for update preferences.
type UpdatePrefsStatus struct {
	Prefs           desktop.AppPrefs `json:"prefs"`
	LastCheckAt     string           `json:"lastCheckAt"`
	LastCheckResult string           `json:"lastCheckResult"`
	LatestVersion   string           `json:"latestVersion"`
	Platform        string           `json:"platform"`
}

// NewReleaseNotice carries the version found by an automatic check when
// auto-download is off. The window shows a dialog; 立即下载 calls
// DownloadPendingUpdate.
type NewReleaseNotice struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
}

var errUpToDate = errors.New("已是最新版本")

// updateScheduler owns the periodic update check. The updater's built-in
// CheckInterval cannot restart after StopPeriodicCheck, so the app runs its
// own timer to honor runtime preference changes.
type updateScheduler struct {
	emit func(event string, data any)

	mu         sync.Mutex
	timer      *time.Timer
	prefs      desktop.AppPrefs
	lastAt     time.Time
	lastResult string
	lastFound  string
	pending    *updater.Release
}

func newUpdateScheduler(emit func(event string, data any)) *updateScheduler {
	if emit == nil {
		emit = func(string, any) {}
	}
	return &updateScheduler{emit: emit, prefs: desktop.ReadAppPrefs(prefsRoot())}
}

func (s *updateScheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.armLocked()
}

func (s *updateScheduler) armLocked() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if version == "dev" || !s.prefs.AutoCheck {
		return
	}
	interval := time.Duration(s.prefs.CheckIntervalHours) * time.Hour
	s.timer = time.AfterFunc(interval, func() { s.tick() })
}

// tick runs one automatic check. A check flow already in progress makes the
// updater drop this tick, matching its own periodic behavior.
func (s *updateScheduler) tick() {
	s.mu.Lock()
	prefs := s.prefs
	s.mu.Unlock()
	if !prefs.AutoCheck {
		return
	}
	u := application.Get().Updater
	switch u.State() {
	case updater.StateChecking, updater.StateDownloading, updater.StateVerifying, updater.StateInstalling:
		return
	}
	s.check(prefs.AutoUpdate)
}

// check runs one update round. download=true keeps the historical
// find-and-download behavior; download=false only records the found release
// and notifies the window.
func (s *updateScheduler) check(download bool) {
	u := application.Get().Updater
	rel, err := u.Check(context.Background())
	if err == nil && rel != nil && !download {
		s.mu.Lock()
		s.pending = rel
		s.mu.Unlock()
		s.emit("updates:new-release", NewReleaseNotice{Version: rel.Version, Notes: rel.Notes})
	}
	s.record(err, rel)
}

// DownloadPendingUpdate starts downloading the release found by a
// find-only automatic check.
func (s *updateScheduler) DownloadPendingUpdate() error {
	s.mu.Lock()
	pending := s.pending
	s.mu.Unlock()
	if pending == nil {
		return errors.New("没有待下载的新版本")
	}
	u := application.Get().Updater
	switch u.State() {
	case updater.StateDownloading, updater.StateVerifying, updater.StateInstalling:
		return errors.New("更新正在进行中")
	}
	return u.DownloadAndInstall(context.Background())
}

func (s *updateScheduler) record(err error, rel *updater.Release) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAt = time.Now()
	s.pending = nil
	switch {
	case errors.Is(err, errUpToDate):
		s.lastResult = "已是最新版本"
		s.lastFound = ""
	case err != nil:
		s.lastResult = "检查失败"
		s.lastFound = ""
	default:
		if rel != nil {
			s.lastFound = rel.Version
			s.lastResult = "发现新版本"
		}
	}
}

// Status returns the prefs and the last check outcome for the 关于 tab.
func (s *updateScheduler) Status() UpdatePrefsStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	lastAt := ""
	if !s.lastAt.IsZero() {
		lastAt = s.lastAt.Format("2006-01-02 15:04")
	}
	return UpdatePrefsStatus{
		Prefs:           s.prefs,
		LastCheckAt:     lastAt,
		LastCheckResult: s.lastResult,
		LatestVersion:   s.lastFound,
		Platform:        platformLabel(),
	}
}

// Save applies the 关于 page's prefs. Turning auto-check on fires a check
// immediately; other changes rearm the timer from now.
func (s *updateScheduler) Save(in desktop.AppPrefs) (UpdatePrefsStatus, error) {
	if in.CheckIntervalHours < 1 || in.CheckIntervalHours > desktop.MaxCheckIntervalHours {
		return s.Status(), fmt.Errorf("间隔要填 1 到 %d 之间的小时数", desktop.MaxCheckIntervalHours)
	}
	s.mu.Lock()
	turnedOn := in.AutoCheck && !s.prefs.AutoCheck
	s.prefs = in
	if err := desktop.WriteAppPrefs(prefsRoot(), in); err != nil {
		s.mu.Unlock()
		return s.Status(), err
	}
	s.armLocked()
	s.mu.Unlock()
	if turnedOn && version != "dev" {
		s.tick()
	}
	return s.Status(), nil
}

// RecordManualCheck lets the manual 检查更新 entry share the last-check display.
func (s *updateScheduler) RecordManualCheck(err error) {
	u := application.Get().Updater
	var rel *updater.Release
	if err == nil && u.State() == updater.StateAvailable {
		rel = &updater.Release{Version: ""}
	}
	s.record(manualResult(err), rel)
}

// Stop cancels the timer at shutdown.
func (s *updateScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}

// manualResult folds the updater's no-update sentinel into a nil error for
// manual checks, which surface failure through the settings error toast.
func manualResult(err error) error {
	if errors.Is(err, errUpToDate) {
		return nil
	}
	return err
}
