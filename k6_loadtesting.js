import { check, sleep } from 'k6';
import http from 'k6/http';
import { Counter, Trend } from 'k6/metrics';

const numRequests = new Counter('http_requests');
const numSuccess = new Counter('http_requests_success');
const numFailure = new Counter('http_requests_failed');
const duration = new Trend('http_request_duration');

const gcPause = new Trend('gc_pause');
const gcTotal = new Trend('gc_total');
const cpuTotal = new Trend('cpu_total');
const cpuUser = new Trend('cpu_user');
const cpuIdle = new Trend('cpu_idle');
const gcCycles = new Counter('gc_cycles');
const gcHeapAllocBytes = new Counter('gc_heap_alloc_bytes');
const gcHeapAllocObjects = new Counter('gc_heap_alloc_objects');
const gcHeapObjects = new Trend('gc_heap_objects');
const heapFreeBytes = new Trend('heap_free_bytes');
const heapObjectsBytes = new Trend('heap_objects_bytes');
const heapReleasedBytes = new Trend('heap_released_bytes');
const heapUnusedBytes = new Trend('heap_unused_bytes');
const goroutines = new Trend('goroutines');
const mutexWaitTotal = new Counter('mutex_wait_total');


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
        gcPause.add(systemInfo['/cpu/classes/gc/pause:cpu-seconds'], tags);
        gcTotal.add(systemInfo['/cpu/classes/gc/total:cpu-seconds'], tags);
        cpuTotal.add(systemInfo['/cpu/classes/total:cpu-seconds'], tags);
        cpuUser.add(systemInfo['/cpu/classes/user:cpu-seconds'], tags);
        cpuIdle.add(systemInfo['/cpu/classes/idle:cpu-seconds'], tags);
        gcCycles.add(systemInfo['/gc/cycles/total:gc-cycles'], tags);
        gcHeapAllocBytes.add(systemInfo['/gc/heap/allocs:bytes'], tags);
        gcHeapAllocObjects.add(systemInfo['/gc/heap/allocs:objects'], tags);
        gcHeapObjects.add(systemInfo['/gc/heap/objects:objects'], tags);
        heapFreeBytes.add(systemInfo['/memory/classes/heap/free:bytes'], tags);
        heapObjectsBytes.add(systemInfo['/memory/classes/heap/objects:bytes'], tags);
        heapReleasedBytes.add(systemInfo['/memory/classes/heap/released:bytes'], tags);
        heapUnusedBytes.add(systemInfo['/memory/classes/heap/unused:bytes'], tags);
        goroutines.add(systemInfo['/sched/goroutines:goroutines'], tags);
        mutexWaitTotal.add(systemInfo['/sync/mutex/wait/total:seconds'], tags);
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