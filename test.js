import http from 'k6/http';
import { sleep } from 'k6';

const ROUTES = [
    'http://localhost:8000',
    'http://localhost:8000/path',
]

export const options = {
    vus: 250,
    duration: '10s',
}

export default function () {
  const randomRoute = ROUTES[0];
  http.get(randomRoute)
}
