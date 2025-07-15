import { check, sleep } from 'k6';
import http from 'k6/http';
import { Counter, Trend } from 'k6/metrics';

const numSuccess = new Counter('http_requests_success');
const numFailure = new Counter('http_requests_failed');
const duration = new Trend('http_request_duration');
const cpuUser = new Trend('cpu_user');
const cpuSystem = new Trend('cpu_system');
const cpuIdle = new Trend('cpu_idle');
const cpuTotal = new Trend('cpu_total');
const memoryTotal = new Trend('memory_total');
const memoryUsed = new Trend('memory_used');
const networkBytesReceived = new Counter('network_bytes_received');
const networkBytesSent = new Counter('network_bytes_sent');

// Set up options for avg load testing
export const options = {
    stages: [
        { duration: '10s', target: 30 }, // traffic ramp-up
        { duration: '30s', target: 30 }, // hold steady
        { duration: '10s', target: 0 }, // ramp-down to 0 users
    ]
}

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
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
        networkBytesReceived.add(systemInfo.network.bytes_received, tags);
        networkBytesSent.add(systemInfo.network.bytes_sent, tags);
    }

    if (success) {
        numSuccess.add(1, tags);
    } else {
        numFailure.add(1, tags);
    }
    duration.add(res.timings.duration, tags);

    // brief sleep between iterations
    sleep(2);
}