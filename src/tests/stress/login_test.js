import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 50 }, // ramp up to 50 VUs
    { duration: '20s', target: 50 }, // stay at 50 VUs
    { duration: '10s', target: 0 },  // ramp down to 0 VUs
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    http_req_failed: ['rate<0.01'],
  },
};

// URL del backend y credenciales de un usuario de prueba ya registrado.
// Se pasan por variables de entorno de k6 (no se guardan credenciales en el código):
//   k6 run -e BASE_URL=http://localhost:8080 \
//          -e TEST_EMAIL=demo@example.com -e TEST_PASSWORD=<clave> tests/stress/login_test.js
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TEST_EMAIL = __ENV.TEST_EMAIL || 'demo@example.com';
const TEST_PASSWORD = __ENV.TEST_PASSWORD || 'cambiar-esta-clave';

export default function () {
  const payload = JSON.stringify({ email: TEST_EMAIL, password: TEST_PASSWORD });

  const params = { headers: { 'Content-Type': 'application/json' } };

  const res = http.post(`${BASE_URL}/auth/login`, payload, params);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'has token': (r) => JSON.parse(r.body).token !== undefined,
  });

  sleep(1);
}
