package metrics

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

// Snapshot holds a point-in-time reading of system resource usage (0–100%).
type Snapshot struct {
	CPU    float64
	Memory float64
	Disk   float64
}

// Reader polls system metrics on a configurable interval.
type Reader struct {
	Interval time.Duration
	Path     string // mount point for disk usage, e.g. "/"
}

// DefaultReader returns a Reader with sensible defaults for macOS.
func DefaultReader() *Reader {
	return &Reader{
		Interval: 3 * time.Second,
		Path:     "/",
	}
}

// Poll reads CPU, memory, and disk usage once.
func (r *Reader) Poll(ctx context.Context) (Snapshot, error) {
	cpuPct, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return Snapshot{}, err
	}

	memInfo, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return Snapshot{}, err
	}

	diskInfo, err := disk.UsageWithContext(ctx, r.Path)
	if err != nil {
		return Snapshot{}, err
	}

	var cpuVal float64
	if len(cpuPct) > 0 {
		cpuVal = cpuPct[0]
	}

	return Snapshot{
		CPU:    cpuVal,
		Memory: memInfo.UsedPercent,
		Disk:   diskInfo.UsedPercent,
	}, nil
}
