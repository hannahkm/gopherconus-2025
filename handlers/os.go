// Helper structs and functions for getting system information
// This information will be included in responses to the /hello endpoint

package handlers

import (
	"fmt"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/network"
)

type SystemStats struct {
	CPU     CPUStats     `json:"cpu"`
	Memory  MemoryStats  `json:"memory"`
	Network NetworkStats `json:"network"`
}

type CPUStats struct {
	User   uint64 `json:"user"`
	System uint64 `json:"system"`
	Idle   uint64 `json:"idle"`
	Total  uint64 `json:"total"`
}

type MemoryStats struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
}

type NetworkStats struct {
	BytesReceived uint64 `json:"bytes_received"`
	BytesSent     uint64 `json:"bytes_sent"`
}

func getSystemStats() *SystemStats {
	cpu, err := cpu.Get()
	if err != nil {
		fmt.Printf("error getting CPU stats: %s\n", err)
		return nil
	}

	memory, err := memory.Get()
	if err != nil {
		fmt.Printf("error getting memory stats: %s\n", err)
		return nil
	}

	networks, err := network.Get()
	if err != nil {
		fmt.Printf("error getting network stats: %s\n", err)
		return nil
	}

	var totalRx, totalTx uint64
	for _, net := range networks {
		totalRx += net.RxBytes
		totalTx += net.TxBytes
	}

	return &SystemStats{
		CPU: CPUStats{
			User:   cpu.User,
			System: cpu.System,
			Idle:   cpu.Idle,
			Total:  cpu.Total,
		},
		Memory: MemoryStats{
			Total: memory.Total,
			Used:  memory.Used,
		},
		Network: NetworkStats{
			BytesReceived: totalRx,
			BytesSent:     totalTx,
		},
	}
}
