import { redirect, type Handle, type HandleFetch } from '@sveltejs/kit';

import { env as publicEnv } from '$env/dynamic/public';
import { env as privateEnv } from '$env/dynamic/private';
import { sequence } from '@sveltejs/kit/hooks';

export const SESSION = '__session_id' as const;

export const handleFetch: HandleFetch = (async ({ request, fetch, event: { cookies } }) => {
	console.log("Fetch hook set auth token form cookie on server side");
	console.log("cookies ", cookies)

	console.log("Internal credential", privateEnv.INTERNAL);

	console.log("I'm on client side token", publicEnv.PUBLIC_TOKEN)
	console.log("I'm on host ", publicEnv.PUBLIC_STREAM_HOST)

	if (request.url.startsWith(publicEnv.PUBLIC_STREAM_HOST)) {
		request.headers.set('Authorization', `Bearer ${publicEnv.PUBLIC_TOKEN}`);
	}

	const resp = await fetch(request);

	//TODO: Handle this for 401 refesh token

	return resp
}) satisfies HandleFetch;

const publicRoutes = [
	"/",
	"/api/login"
];

const sessionHook: Handle = async ({ resolve, event }): Promise<any> => {
	const sessionId = event.cookies.get(SESSION);

	if (!sessionId && !publicRoutes.includes(event.url.pathname)) {
		console.log("Unauth access");
		// TODO: Remove all from storage 
		throw redirect(303, '/');
	}

	return resolve(event)
}

export const handle: Handle = sequence(sessionHook) satisfies Handle;
