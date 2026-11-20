import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 50 }, // ramp up to 50 VUs
    { duration: '20s', target: 50 }, // stay at 50 VUs
    { duration: '10s', target: 0 },  // ramp down to 0 VUs
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests must complete below 500ms
    http_req_failed: ['rate<0.01'],   // less than 1% of requests may fail
  },
};

// URL del backend a probar. Por defecto, el servidor local:
//   k6 run -e BASE_URL=http://localhost:8080 tests/stress/load_test.js
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const res = http.get(`${BASE_URL}/health`);

  check(res, {
    'status is 200': (r) => r.status === 200,
  });

  sleep(1);
}
