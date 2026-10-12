import http from 'k6/http';
import { sleep } from 'k6';

const ROUTES = [
    'http://localhost:8000',
]

export const options = {
    vus: 50,
    duration: '200s',
}

export default function () {
  const randomRoute = ROUTES[0];
  http.get(randomRoute)
}
