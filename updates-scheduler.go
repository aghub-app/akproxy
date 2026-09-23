package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"akproxy/internal/desktop"
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

// updateScheduler owns the periodic update check. The updater's built-in
// CheckInterval cannot restart after StopPeriodicCheck, so the app runs its
// own timer to honor runtime preference changes.
type updateScheduler struct {
	emit    func(event string, data any)
	updater updateClient

	mu         sync.Mutex
	checkMu    sync.Mutex
	timer      *time.Timer
	generation uint64
	stopped    bool
	prefs      desktop.AppPrefs
	lastAt     time.Time
	lastResult string
	lastFound  string
	pending    *updater.Release
}

type updateClient interface {
	State() updater.State
	Check(context.Context) (*updater.Release, error)
	DownloadAndInstall(context.Context) error
}

func newUpdateScheduler(emit func(event string, data any), client updateClient) *updateScheduler {
	if emit == nil {
		emit = func(string, any) {}
	}
	return &updateScheduler{emit: emit, updater: client, prefs: desktop.ReadAppPrefs(prefsRoot())}
}

func (s *updateScheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped = false
	s.armLocked()
}

func (s *updateScheduler) armLocked() {
	s.generation++
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if s.stopped || version == "dev" || !s.prefs.AutoCheck {
		return
	}
	interval := time.Duration(s.prefs.CheckIntervalHours) * time.Hour
	generation := s.generation
	s.timer = time.AfterFunc(interval, func() { s.tickAt(generation) })
}

// tick runs one automatic check. A check flow already in progress makes the
// updater drop this tick, matching its own periodic behavior.
func (s *updateScheduler) tick() {
	s.tickAt(0)
}

func (s *updateScheduler) tickAt(generation uint64) {
	s.mu.Lock()
	if s.stopped || version == "dev" || !s.prefs.AutoCheck || (generation != 0 && generation != s.generation) {
		s.mu.Unlock()
		return
	}
	current := s.generation
	prefs := s.prefs
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		if current == s.generation {
			s.armLocked()
		}
		s.mu.Unlock()
	}()
	if !s.checkMu.TryLock() {
		return
	}
	defer s.checkMu.Unlock()
	switch s.updater.State() {
	case updater.StateChecking, updater.StateDownloading, updater.StateVerifying, updater.StateInstalling, updater.StateReady:
		return
	}
	_ = s.check(context.Background(), prefs.AutoUpdate, true)
}

// check runs one update round. download=true keeps the historical
// find-and-download behavior; download=false only records the found release
// and notifies the window.
func (s *updateScheduler) check(ctx context.Context, download, background bool) error {
	rel, err := s.updater.Check(ctx)
	s.record(err, rel)
	s.emit("updates:checked", s.Status())
	if err != nil && background {
		s.emit("updates:check-error", err.Error())
	}
	if err != nil || rel == nil {
		return err
	}
	if !download {
		s.emit("updates:new-release", NewReleaseNotice{Version: rel.Version, Notes: rel.Notes})
		return nil
	}
	if err := s.updater.DownloadAndInstall(ctx); err != nil {
		return err
	}
	s.clearPending()
	return nil
}

func (s *updateScheduler) ManualCheck(ctx context.Context) error {
	if !s.checkMu.TryLock() {
		return errors.New("更新正在进行中")
	}
	defer s.checkMu.Unlock()
	switch s.updater.State() {
	case updater.StateChecking, updater.StateDownloading, updater.StateVerifying, updater.StateInstalling:
		return errors.New("更新正在进行中")
	}
	return s.check(ctx, true, false)
}

// DownloadPendingUpdate starts downloading the release found by a
// find-only automatic check.
func (s *updateScheduler) DownloadPendingUpdate() error {
	if !s.checkMu.TryLock() {
		return errors.New("更新正在进行中")
	}
	defer s.checkMu.Unlock()
	s.mu.Lock()
	pending := s.pending
	s.mu.Unlock()
	if pending == nil {
		return errors.New("没有待下载的新版本")
	}
	switch s.updater.State() {
	case updater.StateDownloading, updater.StateVerifying, updater.StateInstalling:
		return errors.New("更新正在进行中")
	}
	if err := s.updater.DownloadAndInstall(context.Background()); err != nil {
		return err
	}
	s.clearPending()
	return nil
}

func (s *updateScheduler) clearPending() {
	s.mu.Lock()
	s.pending = nil
	s.mu.Unlock()
}

func (s *updateScheduler) record(err error, rel *updater.Release) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAt = time.Now()
	s.pending = rel
	switch {
	case err != nil:
		s.lastResult = "检查失败"
		s.lastFound = ""
	case rel == nil:
		s.lastResult = "已是最新版本"
		s.lastFound = ""
	default:
		s.lastFound = rel.Version
		s.lastResult = "发现新版本"
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
	if err := desktop.WriteAppPrefs(prefsRoot(), in); err != nil {
		s.mu.Unlock()
		return s.Status(), err
	}
	s.prefs = in
	s.armLocked()
	s.mu.Unlock()
	if turnedOn && version != "dev" {
		go s.tick()
	}
	return s.Status(), nil
}

// Stop cancels the timer at shutdown.
func (s *updateScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped = true
	s.armLocked()
}
