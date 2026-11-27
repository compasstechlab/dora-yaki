import assert from 'node:assert/strict';
import { afterEach, describe, it } from 'node:test';
import { buildBackendRequest, onRequest } from '../../functions/api/[[path]].ts';

const BACKEND = 'https://asia-northeast1-example.cloudfunctions.net/dora-yaki-backend';

describe('buildBackendRequest', () => {
	it('keeps the /api path and query under the backend base URL', async () => {
		const req = new Request('https://dora.example.com/api/metrics/cycle-time?start=2026-01-01');
		const out = await buildBackendRequest(req, BACKEND);
		assert.equal(out.url, `${BACKEND}/api/metrics/cycle-time?start=2026-01-01`);
	});

	it('ignores a trailing slash on the backend base URL', async () => {
		const req = new Request('https://dora.example.com/api/auth/me');
		const out = await buildBackendRequest(req, `${BACKEND}/`);
		assert.equal(out.url, `${BACKEND}/api/auth/me`);
	});

	it('never leaves the backend host even for a protocol-relative looking path', async () => {
		const req = new Request('https://dora.example.com//evil.example.org/api/auth/me');
		const out = await buildBackendRequest(req, BACKEND);
		assert.equal(new URL(out.url).host, new URL(BACKEND).host);
	});

	it('forwards the session cookie and does not follow redirects', async () => {
		const req = new Request('https://dora.example.com/api/auth/me', {
			headers: { cookie: 'dora_yaki_session=abc' },
		});
		const out = await buildBackendRequest(req, BACKEND);
		assert.equal(out.headers.get('cookie'), 'dora_yaki_session=abc');
		assert.equal(out.redirect, 'manual');
	});

	it('forwards the method and body of write requests', async () => {
		const req = new Request('https://dora.example.com/api/bots', {
			method: 'POST',
			body: '{"name":"renovate"}',
			headers: { 'content-type': 'application/json' },
		});
		const out = await buildBackendRequest(req, BACKEND);
		assert.equal(out.method, 'POST');
		assert.equal(await out.text(), '{"name":"renovate"}');
	});
});

describe('onRequest', () => {
	const originalFetch = globalThis.fetch;
	afterEach(() => {
		globalThis.fetch = originalFetch;
	});

	it('returns 500 when API_BACKEND is not configured', async () => {
		const res = await onRequest({
			request: new Request('https://dora.example.com/api/auth/me'),
			env: {},
		});
		assert.equal(res.status, 500);
	});

	it('returns the backend response as-is, including redirects and Set-Cookie', async () => {
		let calledUrl = '';
		globalThis.fetch = async (input) => {
			calledUrl = input instanceof Request ? input.url : String(input);
			return new Response(null, {
				status: 302,
				headers: { location: 'https://dora.example.com/', 'set-cookie': 'dora_yaki_session=xyz' },
			});
		};
		const res = await onRequest({
			request: new Request('https://dora.example.com/api/auth/github/callback?code=c&state=s'),
			env: { API_BACKEND: BACKEND },
		});
		assert.equal(calledUrl, `${BACKEND}/api/auth/github/callback?code=c&state=s`);
		assert.equal(res.status, 302);
		assert.equal(res.headers.get('location'), 'https://dora.example.com/');
		assert.equal(res.headers.get('set-cookie'), 'dora_yaki_session=xyz');
	});
});
