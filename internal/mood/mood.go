package mood

import (
	"time"

	"github.com/jlnesc/gophertchi/internal/metrics"
)

// Mood represents the Gopher's current emotional state.
type Mood int

const (
	Happy Mood = iota
	Hungry
	Tired
	Sad
	Sleeping
)

func (m Mood) String() string {
	switch m {
	case Happy:
		return "Happy"
	case Hungry:
		return "Hungry"
	case Tired:
		return "Tired"
	case Sad:
		return "Sad"
	case Sleeping:
		return "Sleeping"
	default:
		return "Unknown"
	}
}

// FileName returns the PNG filename for this mood.
func (m Mood) FileName() string {
	return m.String() + ".png"
}

// Thresholds define enter/exit bands for each mood trigger.
// Enter thresholds fire when a metric crosses above the value;
// exit thresholds require the metric to drop below before leaving that mood.
type Thresholds struct {
	CPUHungryEnter    float64
	CPUHungryExit     float64
	RAMTiredEnter     float64
	RAMTiredExit      float64
	DiskSadEnter      float64
	DiskSadExit       float64
	CPUSleepingEnter  float64 // CPU must be below this to consider sleeping
	CPUSleepingExit   float64
	RAMSleepingEnter  float64
	RAMSleepingExit   float64
	DiskSleepingEnter float64
	DiskSleepingExit  float64
}

// DefaultThresholds returns production-friendly defaults.
func DefaultThresholds() Thresholds {
	return Thresholds{
		CPUHungryEnter:    75,
		CPUHungryExit:     60,
		RAMTiredEnter:     60,
		RAMTiredExit:      50,
		DiskSadEnter:      90,
		DiskSadExit:       80,
		CPUSleepingEnter:  10,
		CPUSleepingExit:   20,
		RAMSleepingEnter:  50,
		RAMSleepingExit:   60,
		DiskSleepingEnter: 70,
		DiskSleepingExit:  80,
	}
}

// Evaluator maps metrics to moods with hysteresis and dwell-time smoothing.
type Evaluator struct {
	Thresholds     Thresholds
	DwellTime      time.Duration // minimum time a candidate mood must hold before switching
	now            func() time.Time
	current        Mood
	candidate      Mood
	candidateSince time.Time
	lastChange     time.Time
}

// DefaultEvaluator returns an Evaluator ready for use.
func DefaultEvaluator() *Evaluator {
	now := time.Now()
	return &Evaluator{
		Thresholds:     DefaultThresholds(),
		DwellTime:      6 * time.Second,
		now:            time.Now,
		current:        Happy,
		candidate:      Happy,
		candidateSince: now,
		lastChange:     now,
	}
}

// SetClock overrides the time source (useful in tests).
func (e *Evaluator) SetClock(now func() time.Time) {
	if now == nil {
		e.now = time.Now
		return
	}
	e.now = now
}

// Current returns the stable mood after hysteresis.
func (e *Evaluator) Current() Mood {
	return e.current
}

// Update processes a new metrics snapshot and returns the (possibly unchanged) mood.
func (e *Evaluator) Update(s metrics.Snapshot) Mood {
	raw := e.evaluateRaw(s, e.current)
	nowFn := e.now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn()

	if raw == e.current {
		e.candidate = e.current
		e.candidateSince = now
		return e.current
	}

	if raw != e.candidate {
		e.candidate = raw
		e.candidateSince = now
		return e.current
	}

	if now.Sub(e.candidateSince) >= e.DwellTime {
		e.current = raw
		e.lastChange = now
	}

	return e.current
}

// evaluateRaw picks a mood from metrics using priority and hysteresis bands.
// Priority: Sad (disk) > Hungry (CPU) > Tired (RAM) > Sleeping > Happy.
func (e *Evaluator) evaluateRaw(s metrics.Snapshot, current Mood) Mood {
	if e.inDiskSad(s, current) {
		return Sad
	}
	if e.inCPUHungry(s, current) {
		return Hungry
	}
	if e.inRAMTired(s, current) {
		return Tired
	}
	if e.inSleeping(s, current) {
		return Sleeping
	}
	return Happy
}

func (e *Evaluator) inDiskSad(s metrics.Snapshot, current Mood) bool {
	t := e.Thresholds
	if current == Sad {
		return s.Disk >= t.DiskSadExit
	}
	return s.Disk >= t.DiskSadEnter
}

func (e *Evaluator) inCPUHungry(s metrics.Snapshot, current Mood) bool {
	t := e.Thresholds
	if current == Hungry {
		return s.CPU >= t.CPUHungryExit
	}
	return s.CPU >= t.CPUHungryEnter
}

func (e *Evaluator) inRAMTired(s metrics.Snapshot, current Mood) bool {
	t := e.Thresholds
	if current == Tired {
		return s.Memory >= t.RAMTiredExit
	}
	return s.Memory >= t.RAMTiredEnter
}

func (e *Evaluator) inSleeping(s metrics.Snapshot, current Mood) bool {
	t := e.Thresholds
	if current == Sleeping {
		return s.CPU <= t.CPUSleepingExit &&
			s.Memory <= t.RAMSleepingExit &&
			s.Disk <= t.DiskSleepingExit
	}
	return s.CPU <= t.CPUSleepingEnter &&
		s.Memory <= t.RAMSleepingEnter &&
		s.Disk <= t.DiskSleepingEnter
}
