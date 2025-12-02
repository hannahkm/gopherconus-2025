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
const uptime = new Counter('uptime_milliseconds');
const runtimeGoroutines = new Trend('runtime_goroutines');
const runtimeGCs = new Trend('runtime_gc');
const runtimeTotalSTWPause = new Trend('runtime_total_stw_pause');

// Load testing scenarios
export const options = {
    stages: [
        // avg load-testing
        { duration: '15s', target: 100 }, // traffic ramp-up
        { duration: '30s', target: 100 }, // hold steady
        { duration: '15s', target: 0 }, // ramp-down to 0 users

        // spike-testing
        { duration: '2s', target: 1000 }, // sudden jump to 1000 users
        { duration: '2s', target: 0 }, // drop down to 0 users
    ],
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
        uptime.add(systemInfo.uptime.milliseconds, tags);
        runtimeGoroutines.add(systemInfo.runtime.goroutines, tags);
        runtimeGCs.add(systemInfo.runtime.gc, tags);
        runtimeTotalSTWPause.add(systemInfo.runtime.total_pause, tags);
    }

    numRequests.add(1, tags);
    if (success) {
        numSuccess.add(1, tags);
    } else {
        numFailure.add(1, tags);
    }
    duration.add(res.timings.duration, tags);

    // brief sleep between iterations
    sleep(2);
}