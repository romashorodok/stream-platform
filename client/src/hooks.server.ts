import { type Handle, type HandleFetch, json, redirect } from '@sveltejs/kit';

import { sequence } from '@sveltejs/kit/hooks';
import { mapCookiesFromHeader } from '$lib/utils/cookie';
import { REFRESH_TOKEN_ROUTE, VERIFY_ROUTE, _REFRESH_TOKEN } from '$lib';

export const handleFetch: HandleFetch = (async ({ request, fetch, event: { cookies } }) => {
	if (request.url === VERIFY_ROUTE || request.url === REFRESH_TOKEN_ROUTE) {
		const refreshToken = cookies.get(_REFRESH_TOKEN);

		if (!refreshToken) return json({ message: 'missing refresh token' }, { status: 401 });

		request.headers.set('Authorization', `Bearer ${refreshToken}`);
	}

	const resp = await fetch(request);

	//TODO: Handle this for 401 refesh token

	return resp;
}) satisfies HandleFetch;

const verifyTokenHook: Handle = async ({ resolve, event }): Promise<any> => {
	const { cookies } = event;

	const refreshToken = cookies.get(_REFRESH_TOKEN);

	if (!refreshToken) return resolve(event);

	try {
		const response = await event.fetch(VERIFY_ROUTE, {
			method: 'POST'
		});

		if (response.status === 500) {
			cookies.delete(_REFRESH_TOKEN);
			return await resolve(event);
		}

		event.locals.identityPayload = await response.json();

		console.log(`Render page for user identity: ${JSON.stringify(event.locals.identityPayload)}`);
	} catch (e) {
		console.log(e);
	}

	return resolve(event);
};

const setFreshAccessTokenFuncHook: Handle = async ({ resolve, event }): Promise<any> => {
	const { cookies, fetch } = event;

	const refreshToken = cookies.get(_REFRESH_TOKEN);

	if (!refreshToken) return resolve(event);

	event.locals.getAccessToken = async (): Promise<String | null> => {
		try {
			const response = await fetch(REFRESH_TOKEN_ROUTE, { method: 'PUT' });

			const serverCookies = response.headers.get('set-cookie') as string;

			await mapCookiesFromHeader(cookies, serverCookies);

			const { access_token } = await response.json();

			return access_token;
		} catch (e) {
			return null;
		}
	};

	return resolve(event);
};

const protectedRoutes = ['/dashboard'];

const handleProtectedRoutes: Handle = async ({ event, resolve }) => {
	const { locals } = event;

	const containsAtLeastOne = protectedRoutes.some((route) => event.url.pathname.startsWith(route));

	if (protectedRoutes.includes(event.url.pathname) || containsAtLeastOne) {
		if (!locals.getAccessToken || !locals.identityPayload) {
			console.log('Redirect to page /');
			throw redirect(301, '/');
		}

		console.log('Hit protected route', event.url.pathname);
	}

	return resolve(event);
};

export const handle: Handle = sequence(
	verifyTokenHook,
	setFreshAccessTokenFuncHook,
	handleProtectedRoutes
) satisfies Handle;
