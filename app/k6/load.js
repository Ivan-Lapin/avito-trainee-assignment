import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 10,               // 10 виртуальных пользователей
  duration: '1m',        // 1 минута
  thresholds: {
    http_req_duration: ['p(95)<300'], // p95 < 300 мс
    http_req_failed: ['rate<0.001'],  // <0.1% ошибок
  },
};

const BASE = __ENV.BASE || 'http://localhost:8095/api';

export default function () {
  // Health
  let res = http.get(`${BASE}/health`);
  check(res, { '200': (r) => r.status === 200 });

  // Create PR (рандомный id)
  const prId = crypto.randomUUID();
  const body = JSON.stringify({
    id: prId,
    title: "bench",
    author_id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
    team_name: "core"
  });
  res = http.post(`${BASE}/pullRequest/create`, body, { headers: { 'Content-Type': 'application/json' }});
  check(res, { 'create<=201': (r) => r.status === 201 || r.status === 409 });

  // Merge идемпотентно
  const mbody = JSON.stringify({ id: prId });
  res = http.post(`${BASE}/pullRequest/merge`, mbody, { headers: { 'Content-Type': 'application/json', 'Idempotency-Key': prId }});
  check(res, { 'merge 200': (r) => r.status === 200 });

  sleep(0.2);
}
