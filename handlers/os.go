// Helper structs and functions for getting system information
// This information will be included in responses to the /hello endpoint

package handlers

import (
	"fmt"
	"runtime/metrics"
)

type SystemStats map[string]float64

func getSystemStats() *SystemStats {
	stats := SystemStats{}

	samples := []metrics.Sample{
		{Name: "/cpu/classes/gc/pause:cpu-seconds"},   // estimated CPU time spent performing GC tasks
		{Name: "/cpu/classes/gc/total:cpu-seconds"},   // total CPU time spent performing GC tasks
		{Name: "/cpu/classes/total:cpu-seconds"},      // estimated total available CPU time for Go code and/or Go runtime
		{Name: "/cpu/classes/user:cpu-seconds"},       // estimated CPU time spent running user code
		{Name: "/cpu/classes/idle:cpu-seconds"},       // estimated total available CPU time not spent executing Go code and/or Go runtime code
		{Name: "/gc/cycles/total:gc-cycles"},          // total number of GC cycles performed
		{Name: "/gc/heap/allocs:bytes"},               // total sum of bytes allocated
		{Name: "/gc/heap/allocs:objects"},             // total sum of allocations triggered by application
		{Name: "/gc/heap/objects:objects"},            // number of objects occupying heap memory
		{Name: "/memory/classes/heap/free:bytes"},     // memory available to be returned to the system
		{Name: "/memory/classes/heap/objects:bytes"},  // memory occupied by live and dead, but not freed objects
		{Name: "/memory/classes/heap/released:bytes"}, // memory free and returned to the system
		{Name: "/memory/classes/heap/unused:bytes"},   // memory reserved, but not already used, for heap objects
		{Name: "/sched/goroutines:goroutines"},        // live goroutines
		{Name: "/sync/mutex/wait/total:seconds"},      // estimated total time goroutines have spent waiting for mutexes
	}

	metrics.Read(samples)

	for _, v := range samples {
		name, value := v.Name, v.Value
		switch value.Kind() {
		case metrics.KindBad:
			fmt.Println("bad metric, skipping")
			continue
		case metrics.KindFloat64:
			stats[name] = value.Float64()
		case metrics.KindUint64:
			stats[name] = float64(value.Uint64())
		default:
			fmt.Println("unsupported metric kind, skipping")
			continue
		}
	}
	return &stats

}
