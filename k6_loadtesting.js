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
const memoryTotal = new Trend('memory_total');
const memoryUsed = new Trend('memory_used');
const uptime = new Counter('uptime_milliseconds')

// Load testing scenarios
export const options = {
    scenarios: {
        // warmup
        warmup: {
            executor: 'constant-vus',
            vus: 50,
            duration: '15s',
            startTime: '0s',
            gracefulStop: '5s',
        },

        // avg load-testing
        avgLoad: {
            executor: 'constant-arrival-rate',
            startTime: '15s',
            duration: '45s',
            rate: 30,
            timeUnit: '1s',
            preAllocatedVUs: 5,
            maxVUs: 20,
            gracefulStop: '5s',
        },

        // heavy load-testing
        heavyLoad: {
            executor: 'constant-vus',
            startTime: '15s',
            duration: '45s',
            vus: 5,
            gracefulStop: '5s',
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
        memoryTotal.add(systemInfo.memory.total, tags);
        memoryUsed.add(systemInfo.memory.used, tags);
        uptime.add(systemInfo.uptime.milliseconds, tags);
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