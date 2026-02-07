import type { Handle } from '@sveltejs/kit';

// Server-side API proxy: forward /api/* requests to the backend
const API_BACKEND = process.env.API_BACKEND || 'http://localhost:7202';

export const handle: Handle = async ({ event, resolve }) => {
	if (event.url.pathname.startsWith('/api/')) {
		const backendUrl = `${API_BACKEND}${event.url.pathname}${event.url.search}`;
		const res = await fetch(backendUrl, {
			method: event.request.method,
			headers: event.request.headers,
			body: event.request.method !== 'GET' && event.request.method !== 'HEAD'
				? await event.request.arrayBuffer()
				: undefined,
			// @ts-expect-error duplex is required for streaming request bodies
			duplex: 'half',
		});

		return new Response(res.body, {
			status: res.status,
			statusText: res.statusText,
			headers: res.headers,
		});
	}

	return resolve(event);
};
