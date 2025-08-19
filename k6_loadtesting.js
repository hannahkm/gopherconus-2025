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
const INSTRUMENTATION = __ENV.INSTRUMENTATION

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
        instrumentation: INSTRUMENTATION,
        success: success,
        status: res.status,
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