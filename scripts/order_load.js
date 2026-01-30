import http from 'k6/http';
import { check } from 'k6';

// 訂單負載測試 — 請打到 UAT，勿對 dev / production 壓測
//
// UAT（本機 Docker）：
//   k6 run --env URL=http://localhost:8081 scripts/order_load.js
// 未傳 URL 時預設為 http://localhost:8080（dev 本機）

export const options = {
  vus: 20,
  duration: '10s',
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.1'],
  },
};

const BASE = __ENV.URL || 'http://localhost:8080';
const USER_COUNT = parseInt(__ENV.USER_COUNT || '2000', 10);
const QUANTITY = parseInt(__ENV.QUANTITY || '1', 10);

// 透過 API 建立活動 → 票種 → 活動開賣（預熱該活動底下所有票）；user 仍用既有 ID 範圍（不經 API）
export function setup() {
  const api = `${BASE}/api/v1`;

  const eventRes = http.post(
    `${api}/events`,
    JSON.stringify({ name: 'Load Test Event', description: 'k6 setup' }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  if (eventRes.status !== 201) {
    throw new Error(`create event failed: ${eventRes.status} ${eventRes.body}`);
  }
  const event = eventRes.json();
  const eventIdInternal = event.id;
  const eventIdUuid = event.event_id;

  const ticketRes = http.post(
    `${api}/tickets`,
    JSON.stringify({
      event_id: eventIdInternal,
      name: 'Load Test Ticket',
      price: 100,
      total_stock: 1000,
      max_per_user: 1,
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  if (ticketRes.status !== 201) {
    throw new Error(`create ticket failed: ${ticketRes.status} ${ticketRes.body}`);
  }
  const ticket = ticketRes.json();
  const ticketId = ticket.id;

  const openRes = http.post(
    `${api}/events/${eventIdUuid}/open-for-sale`,
    null,
    { headers: { 'Content-Type': 'application/json' } }
  );
  if (openRes.status !== 200) {
    throw new Error(`open-for-sale failed: ${openRes.status} ${openRes.body}`);
  }

  return { ticketId };
}

export default function (data) {
  const ticketId = data.ticketId;
  const userId = Math.floor(Math.random() * USER_COUNT) + 1;
  const body = JSON.stringify({
    user_id: userId,
    ticket_id: ticketId,
    quantity: QUANTITY,
  });
  const res = http.post(`${BASE}/api/v1/orders`, body, {
    headers: { 'Content-Type': 'application/json' },
  });
  check(res, {
    'status 201 or 409 or 400': (r) =>
      r.status === 201 || r.status === 409 || r.status === 400,
  });
}
