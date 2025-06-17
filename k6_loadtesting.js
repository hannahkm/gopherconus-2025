import { check, sleep } from 'k6';
import http from 'k6/http';
import { Counter, Trend } from 'k6/metrics';

const numSuccess = new Counter('http_requests_success');
const numFailure = new Counter('http_requests_failed');
const duration = new Trend('http_request_duration');

// Set up options for avg load testing
export const options = {
    stages: [
        { duration: '15s', target: 30 }, // traffic ramp-up
        { duration: '45s', target: 30 }, // hold steady
        { duration: '15s', target: 0 }, // ramp-down to 0 users
    ]
}

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
    const res = http.get(BASE_URL)
    const success = check(res, {
        'status 200': (r) => r.status === 200,
        'body says hello': (r) => {
            try {
                const b = JSON.parse(r.body);
                return b.message === 'Hello World!'
            } catch (e) {
                console.log("failed:", e)
                return false;
            }
        }
    });

    if (success) {
        numSuccess.add(1);
    } else {
        numFailure.add(1);
    }
    duration.add(res.timings.duration);

    // brief sleep between iterations
    sleep(2);
}