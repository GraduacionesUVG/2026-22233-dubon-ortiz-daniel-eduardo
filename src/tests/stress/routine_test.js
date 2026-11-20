import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '5s', target: 3 },  // ramp up to 3 VUs
    { duration: '15s', target: 3 }, // stay at 3 VUs
    { duration: '5s', target: 0 },  // ramp down to 0 VUs
  ],
  thresholds: {
    // 95% of requests must have a Time-To-First-Byte below 10s (backend + LLM latency)
    http_req_waiting: ['p(95)<10000'],
    http_req_failed: ['rate<0.01'],
  },
};

// URL del backend y credenciales de un usuario de prueba ya registrado.
// Se pasan por variables de entorno de k6 (no se guardan credenciales en el código):
//   k6 run -e BASE_URL=http://localhost:8080 \
//          -e TEST_EMAIL=demo@example.com -e TEST_PASSWORD=<clave> tests/stress/routine_test.js
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TEST_EMAIL = __ENV.TEST_EMAIL || 'demo@example.com';
const TEST_PASSWORD = __ENV.TEST_PASSWORD || 'cambiar-esta-clave';

export function setup() {
  const payload = JSON.stringify({ email: TEST_EMAIL, password: TEST_PASSWORD });

  const params = { headers: { 'Content-Type': 'application/json' } };

  const res = http.post(`${BASE_URL}/auth/login`, payload, params);

  check(res, {
    'login status is 200': (r) => r.status === 200,
    'has token': (r) => r.json().token !== undefined,
  });

  return { token: res.json().token };
}

export default function (data) {
  const payload = JSON.stringify({ prompt: 'Genera una rutina rapida de pierna para hipertrofia no tengo lesiones, estoy al 100 de energia y tengo 30m' });

  const params = {
    headers: {
      'Authorization': 'Bearer ' + data.token,
      'Content-Type': 'application/json',
    },
  };

  const res = http.post(`${BASE_URL}/routine`, payload, params);

  check(res, {
    'status is 200': (r) => r.status === 200,
  });

  sleep(2); // space out requests to avoid rate limits
}
