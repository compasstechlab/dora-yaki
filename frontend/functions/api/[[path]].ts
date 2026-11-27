// Cloudflare Pages Function: proxies /api/* to the backend so the session cookie stays first-party.

// PagesContext : the subset of the Pages Functions context this proxy uses.
interface PagesContext {
	request: Request;
	env: { API_BACKEND?: string };
}

// buildBackendRequest : rebuilds the incoming request against backendBase, keeping path, query, headers and body.
export async function buildBackendRequest(request: Request, backendBase: string): Promise<Request> {
	const src = new URL(request.url);
	const base = new URL(backendBase);
	const target = new URL(base);
	target.pathname = `${base.pathname.replace(/\/+$/, '')}${src.pathname}`;
	target.search = src.search;

	const headers = new Headers(request.headers);
	headers.delete('host');
	const hasBody = request.method !== 'GET' && request.method !== 'HEAD';

	return new Request(target, {
		method: request.method,
		headers,
		body: hasBody ? await request.arrayBuffer() : undefined,
		redirect: 'manual', // pass the OAuth 302 and its Set-Cookie through to the browser
	});
}

// onRequest : Pages Functions entry point for /api/*.
export async function onRequest({ request, env }: PagesContext): Promise<Response> {
	if (!env.API_BACKEND) {
		return new Response('API_BACKEND is not configured', { status: 500 });
	}
	return fetch(await buildBackendRequest(request, env.API_BACKEND));
}
