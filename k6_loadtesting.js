import { check, sleep } from 'k6';
import http from 'k6/http';
import { Counter, Trend } from 'k6/metrics';

const numRequests = new Counter('http_requests');
const numSuccess = new Counter('http_requests_success');
const numFailure = new Counter('http_requests_failed');
const duration = new Trend('http_request_duration');
const cpuUser = new Trend('cpu_user');
const cpuSystem = new Trend('cpu_system');
const cpuIdle = new Trend('cpu_idle');
const cpuTotal = new Trend('cpu_total');
const memoryHeapSystem = new Trend('memory_heap_system');
const memoryHeapAllocated = new Trend('memory_heap_allocated');
const memoryHeapIdle = new Trend('memory_heap_idle');
const memoryHeapInuse = new Trend('memory_heap_inuse');
const memoryHeapReleased = new Trend('memory_heap_released');
const memoryHeapObjects = new Trend('memory_heap_objects');
const uptime = new Counter('uptime_milliseconds');
const runtimeGoroutines = new Trend('runtime_goroutines');
const runtimeGCs = new Trend('runtime_gc');
const runtimeTotalSTWPause = new Trend('runtime_total_stw_pause');
const runtimeStackInUse = new Trend('runtime_stack_in_use');
const runtimeStackSys = new Trend('runtime_stack_sys');

// Load testing scenarios
export const options = {
    scenarios: {
        // warmup
        warmup: {
            executor: 'constant-vus',
            vus: 50,
            duration: '30s',
            startTime: '0s',
            gracefulStop: '5s',
        },

        // avg load-testing
        avgLoad: {
            executor: 'constant-arrival-rate',
            startTime: '30s',
            duration: '45s',
            rate: 30,
            timeUnit: '1s',
            preAllocatedVUs: 20,
            maxVUs: 20,
            gracefulStop: '5s',
        },

        // heavy load-testing
        heavyLoad: {
            executor: 'constant-arrival-rate',
            startTime: '30s',
            duration: '45s',
            rate: 300,
            timeUnit: '1s',
            preAllocatedVUs: 20,
            gracefulStop: '0s',
        },
    },
    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)', 'p(99.9)', 'count'],
}

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/hello';
const INSTRUMENTATION = __ENV.INSTRUMENTATION || 'default';

export default function () {
    const res = http.get(BASE_URL)

    let parsedBody, systemInfo;
    try {
        parsedBody = JSON.parse(res.body);
        systemInfo = parsedBody.system_info;
    } catch (e) {
        console.log("Failed to parse response body:", e);
    }

    const success = check(res, {
        'status 200': (r) => r.status === 200,
        'body says hello': (r) => parsedBody && parsedBody.message === `Hello, ${INSTRUMENTATION} instrumentation!`
    });

    var tags = {
        instrumentation: String(INSTRUMENTATION),
        success: Boolean(success),
        status: Number(res.status),
    }

    if (systemInfo) {
        cpuUser.add(systemInfo.cpu.user, tags);
        cpuSystem.add(systemInfo.cpu.system, tags);
        cpuIdle.add(systemInfo.cpu.idle, tags);
        cpuTotal.add(systemInfo.cpu.total, tags);
        memoryHeapSystem.add(systemInfo.memory.memory_system, tags);
        memoryHeapAllocated.add(systemInfo.memory.memory_heap_allocated, tags);
        memoryHeapIdle.add(systemInfo.memory.memory_heap_idle, tags);
        memoryHeapInuse.add(systemInfo.memory.memory_heap_inuse, tags);
        memoryHeapReleased.add(systemInfo.memory.memory_heap_released, tags);
        memoryHeapObjects.add(systemInfo.memory.memory_heap_objects, tags);
        uptime.add(systemInfo.uptime.milliseconds, tags);
        runtimeGoroutines.add(systemInfo.runtime.goroutines, tags);
        runtimeGCs.add(systemInfo.runtime.gc, tags);
        runtimeTotalSTWPause.add(systemInfo.runtime.total_pause, tags);
        runtimeStackInUse.add(systemInfo.runtime.stack_in_use, tags);
        runtimeStackSys.add(systemInfo.runtime.stack_sys, tags);
    }

    numRequests.add(1, tags);
    if (success) {
        numSuccess.add(1, tags);
    } else {
        numFailure.add(1, tags);
    }
    duration.add(res.timings.duration, tags);

    // brief sleep between iterations, unless we are intentionally running a heavy load test
    if (__ENV.SCENARIO !== 'heavyLoad') {
        sleep(2);
    }
}