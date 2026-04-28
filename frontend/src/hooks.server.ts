import type { Handle } from '@sveltejs/kit';

// Server-side API proxy: forward /api/* requests to the backend.
const API_BACKEND = process.env.API_BACKEND || 'http://localhost:7202';

export const handle: Handle = async ({ event, resolve }) => {
	if (event.url.pathname.startsWith('/api/')) {
		const backendUrl = `${API_BACKEND}${event.url.pathname}${event.url.search}`;
		const method = event.request.method;
		const hasBody = method !== 'GET' && method !== 'HEAD';

		// Copy request headers explicitly so cookies and Authorization flow through.
		const requestHeaders = new Headers();
		event.request.headers.forEach((value, key) => requestHeaders.append(key, value));

		const res = await fetch(backendUrl, {
			method,
			headers: requestHeaders,
			body: hasBody ? await event.request.arrayBuffer() : undefined,
			// @ts-expect-error duplex is required for streaming request bodies
			duplex: 'half',
			// Important: do not auto-follow redirects so the OAuth callback's 302
			// reaches the browser instead of being followed inside the proxy.
			redirect: 'manual',
		});

		// Forward all response headers, preserving multi-value Set-Cookie entries.
		const responseHeaders = new Headers();
		res.headers.forEach((value, key) => responseHeaders.append(key, value));

		return new Response(res.body, {
			status: res.status,
			statusText: res.statusText,
			headers: responseHeaders,
		});
	}

	return resolve(event);
};
