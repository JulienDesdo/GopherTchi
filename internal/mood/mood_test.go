package mood_test

import (
	"testing"
	"time"

	"github.com/jlnesc/gophertchi/internal/metrics"
	"github.com/jlnesc/gophertchi/internal/mood"
)

func newTestEvaluator(clock *time.Time, dwell time.Duration) *mood.Evaluator {
	e := mood.DefaultEvaluator()
	e.DwellTime = dwell
	e.SetClock(func() time.Time { return *clock })
	return e
}

func settle(e *mood.Evaluator, snap metrics.Snapshot, clock *time.Time, dwell time.Duration) mood.Mood {
	_ = e.Update(snap) // arm candidate
	*clock = clock.Add(dwell)
	return e.Update(snap)
}

func TestMoodTransitions(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dwell := time.Second
	e := newTestEvaluator(&clock, dwell)

	cases := []struct {
		name string
		snap metrics.Snapshot
		want mood.Mood
	}{
		{"happy", metrics.Snapshot{CPU: 20, Memory: 40, Disk: 50}, mood.Happy},
		{"hungry", metrics.Snapshot{CPU: 80, Memory: 40, Disk: 50}, mood.Hungry},
		{"tired", metrics.Snapshot{CPU: 20, Memory: 70, Disk: 50}, mood.Tired},
		{"sad", metrics.Snapshot{CPU: 20, Memory: 40, Disk: 95}, mood.Sad},
		{"sleeping", metrics.Snapshot{CPU: 5, Memory: 40, Disk: 50}, mood.Sleeping},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := settle(e, tc.snap, &clock, dwell)
			if got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestPrioritySadOverHungry(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dwell := time.Second
	e := newTestEvaluator(&clock, dwell)

	got := settle(e, metrics.Snapshot{CPU: 90, Memory: 90, Disk: 95}, &clock, dwell)
	if got != mood.Sad {
		t.Fatalf("got %s, want Sad", got)
	}
}

func TestHysteresisCPUHungry(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dwell := time.Second
	e := newTestEvaluator(&clock, dwell)
	th := e.Thresholds

	// Enter hungry at enter threshold.
	got := settle(e, metrics.Snapshot{CPU: th.CPUHungryEnter, Memory: 40, Disk: 50}, &clock, dwell)
	if got != mood.Hungry {
		t.Fatalf("enter: got %s, want Hungry", got)
	}

	// Stay hungry while still above exit threshold.
	got = settle(e, metrics.Snapshot{CPU: th.CPUHungryExit + 1, Memory: 40, Disk: 50}, &clock, dwell)
	if got != mood.Hungry {
		t.Fatalf("hold: got %s, want Hungry", got)
	}

	// Leave hungry only below exit threshold.
	got = settle(e, metrics.Snapshot{CPU: th.CPUHungryExit - 1, Memory: 40, Disk: 50}, &clock, dwell)
	if got != mood.Happy {
		t.Fatalf("exit: got %s, want Happy", got)
	}
}

func TestHysteresisRAMTired(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dwell := time.Second
	e := newTestEvaluator(&clock, dwell)
	th := e.Thresholds

	got := settle(e, metrics.Snapshot{CPU: 20, Memory: th.RAMTiredEnter, Disk: 50}, &clock, dwell)
	if got != mood.Tired {
		t.Fatalf("enter: got %s, want Tired", got)
	}

	got = settle(e, metrics.Snapshot{CPU: 20, Memory: th.RAMTiredExit + 1, Disk: 50}, &clock, dwell)
	if got != mood.Tired {
		t.Fatalf("hold: got %s, want Tired", got)
	}

	got = settle(e, metrics.Snapshot{CPU: 20, Memory: th.RAMTiredExit - 1, Disk: 50}, &clock, dwell)
	if got != mood.Happy {
		t.Fatalf("exit: got %s, want Happy", got)
	}
}

func TestHysteresisDiskSad(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dwell := time.Second
	e := newTestEvaluator(&clock, dwell)
	th := e.Thresholds

	got := settle(e, metrics.Snapshot{CPU: 20, Memory: 40, Disk: th.DiskSadEnter}, &clock, dwell)
	if got != mood.Sad {
		t.Fatalf("enter: got %s, want Sad", got)
	}

	got = settle(e, metrics.Snapshot{CPU: 20, Memory: 40, Disk: th.DiskSadExit + 1}, &clock, dwell)
	if got != mood.Sad {
		t.Fatalf("hold: got %s, want Sad", got)
	}

	got = settle(e, metrics.Snapshot{CPU: 20, Memory: 40, Disk: th.DiskSadExit - 1}, &clock, dwell)
	if got != mood.Happy {
		t.Fatalf("exit: got %s, want Happy", got)
	}
}

func TestSleepingHysteresis(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dwell := time.Second
	e := newTestEvaluator(&clock, dwell)
	th := e.Thresholds

	got := settle(e, metrics.Snapshot{
		CPU:    th.CPUSleepingEnter,
		Memory: th.RAMSleepingEnter,
		Disk:   th.DiskSleepingEnter,
	}, &clock, dwell)
	if got != mood.Sleeping {
		t.Fatalf("enter: got %s, want Sleeping", got)
	}

	// Still within exit bands → stay sleeping.
	// Keep RAM below Tired enter so Tired does not outrank Sleeping.
	got = settle(e, metrics.Snapshot{
		CPU:    th.CPUSleepingExit,
		Memory: th.RAMTiredEnter - 1,
		Disk:   th.DiskSleepingExit,
	}, &clock, dwell)
	if got != mood.Sleeping {
		t.Fatalf("hold: got %s, want Sleeping", got)
	}

	// CPU rises past exit → leave sleeping.
	got = settle(e, metrics.Snapshot{
		CPU:    th.CPUSleepingExit + 1,
		Memory: th.RAMSleepingEnter,
		Disk:   th.DiskSleepingEnter,
	}, &clock, dwell)
	if got != mood.Happy {
		t.Fatalf("exit: got %s, want Happy", got)
	}
}

func TestDwellTimePreventsFlicker(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dwell := 5 * time.Second
	e := newTestEvaluator(&clock, dwell)

	hungry := metrics.Snapshot{CPU: 90, Memory: 40, Disk: 50}
	if got := e.Update(hungry); got != mood.Happy {
		t.Fatalf("first update should stay Happy, got %s", got)
	}

	clock = clock.Add(2 * time.Second)
	if got := e.Update(hungry); got != mood.Happy {
		t.Fatalf("before dwell should stay Happy, got %s", got)
	}

	clock = clock.Add(4 * time.Second)
	if got := e.Update(hungry); got != mood.Hungry {
		t.Fatalf("after dwell should become Hungry, got %s", got)
	}
}

func TestCandidateResetOnOscillation(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dwell := 5 * time.Second
	e := newTestEvaluator(&clock, dwell)

	hungry := metrics.Snapshot{CPU: 90, Memory: 40, Disk: 50}
	happy := metrics.Snapshot{CPU: 20, Memory: 40, Disk: 50}

	_ = e.Update(hungry)
	clock = clock.Add(3 * time.Second)
	_ = e.Update(happy) // resets candidate
	clock = clock.Add(3 * time.Second)
	_ = e.Update(hungry) // new candidate window starts
	clock = clock.Add(3 * time.Second)
	if got := e.Update(hungry); got != mood.Happy {
		t.Fatalf("interrupted dwell should not switch yet, got %s", got)
	}

	clock = clock.Add(3 * time.Second)
	if got := e.Update(hungry); got != mood.Hungry {
		t.Fatalf("full dwell after reset should switch, got %s", got)
	}
}

func TestBelowEnterThresholdStaysHappy(t *testing.T) {
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dwell := time.Second
	e := newTestEvaluator(&clock, dwell)
	th := e.Thresholds

	got := settle(e, metrics.Snapshot{
		CPU:    th.CPUHungryEnter - 1,
		Memory: th.RAMTiredEnter - 1,
		Disk:   th.DiskSadEnter - 1,
	}, &clock, dwell)
	if got != mood.Happy {
		t.Fatalf("got %s, want Happy", got)
	}
}
