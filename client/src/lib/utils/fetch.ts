import { REFRESH_TOKEN_ROUTE } from '$lib';
import { accessToken } from '$lib/stores/auth';

const getAccessToken = () =>
	new Promise((resolve) => {
		const unsubscribe = accessToken.subscribe((token) => resolve(token));

		unsubscribe();
	});

const setAccessToken = (access_token: string | null) =>
	Promise.resolve(accessToken.set(access_token));

async function refreshToken(): Promise<Response> {
	const token = await getAccessToken();

	return await fetch(REFRESH_TOKEN_ROUTE, {
		method: 'PUT',
		credentials: 'include',
		headers: {
			Authorization: `Bearer ${token}`
		}
	});
}

export const fetchIntercepted =
	() =>
	async (input: RequestInfo | URL, init?: RequestInit | undefined): Promise<Response> => {
		const request = () => Promise.resolve(fetch(input, init));

		const response = await request();

		// @ts-ignore
		if (response.status === 401 && init?.headers && init?.headers['Authorization']) {
			const response = await refreshToken();
			const { access_token } = await response.json();

			// @ts-ignore
			init.headers['Authorization'] = `Bearer ${access_token}`;

			const freshResponse = await request();

			if (!(freshResponse.status >= 400)) {
				setAccessToken(access_token);
			}

			return freshResponse;
		}

		return response;
	};

export type Fetch = typeof fetch
