// Helper structs and functions for getting system information
// This information will be included in responses to the /hello endpoint

package handlers

import (
	"fmt"
	"runtime"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/uptime"
)

type SystemStats struct {
	CPU     CPUStats     `json:"cpu"`
	Memory  MemoryStats  `json:"memory"`
	Uptime  UptimeStats  `json:"uptime"`
	Runtime RuntimeStats `json:"runtime"`
}

type CPUStats struct {
	User   uint64 `json:"user"`
	System uint64 `json:"system"`
	Idle   uint64 `json:"idle"`
	Total  uint64 `json:"total"`
}

type UptimeStats struct {
	Milliseconds uint64 `json:"milliseconds"`
}

type MemoryStats struct {
	MemoryAllocated     uint64 `json:"memory_allocated"`
	MemoryHeapSystem    uint64 `json:"memory_system"`
	MemoryHeapAllocated uint64 `json:"memory_heap_allocated"`
	MemoryHeapIdle      uint64 `json:"memory_heap_idle"`
}

type RuntimeStats struct {
	Goroutines    uint64 `json:"goroutines"`
	GCs           uint64 `json:"gc"`
	TotalSTWPause uint64 `json:"total_pause"`
}

func getSystemStats() *SystemStats {
	cpu, err := cpu.Get()
	if err != nil {
		fmt.Printf("error getting CPU stats: %s\n", err)
		return nil
	}

	uptime, err := uptime.Get()
	if err != nil {
		fmt.Printf("error getting network stats: %s\n", err)
		return nil
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return &SystemStats{
		CPU: CPUStats{
			User:   cpu.User,
			System: cpu.System,
			Idle:   cpu.Idle,
			Total:  cpu.Total,
		},
		Memory: MemoryStats{
			MemoryHeapSystem:    ms.HeapSys,
			MemoryHeapAllocated: ms.HeapAlloc,
			MemoryHeapIdle:      ms.HeapIdle,
		},
		Uptime: UptimeStats{
			Milliseconds: uint64(uptime.Milliseconds()),
		},
		Runtime: RuntimeStats{
			Goroutines:    uint64(runtime.NumGoroutine()),
			GCs:           uint64(ms.NumGC),
			TotalSTWPause: ms.PauseTotalNs,
		},
	}
}
