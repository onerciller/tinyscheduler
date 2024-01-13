package tinyscheduler

import "time"

type RealTimeManager struct{}

func (rtm *RealTimeManager) NewTicker(duration time.Duration) *time.Ticker {
	return time.NewTicker(duration)
}

type MockTimeManager struct {
	C chan time.Time
}

func NewMockTimeManager() *MockTimeManager {
	return &MockTimeManager{C: make(chan time.Time)}
}

func (mtm *MockTimeManager) NewTicker(_ time.Duration) *time.Ticker {
	return &time.Ticker{C: mtm.C}
}
