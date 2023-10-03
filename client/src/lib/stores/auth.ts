import { env } from '$env/dynamic/public';
import { writable } from 'svelte/store';

export type IdentityTokenPayload = {
	aud: Array<String>;
	exp: String;
	iss: String;
	sub: String;
	'user:id': String;
	'token:use': String;
};

type Credentials = { username: string; password: string };

export const accessToken = writable<String | null>(null);
export const identity = writable<IdentityTokenPayload | null>(null);

export const canAccessProtectedRoutes = writable<boolean>(false);

const getAccessToken = () =>
	new Promise((resolve) => {
		const unsubscribe = accessToken.subscribe((token) => resolve(token));

		unsubscribe();
	});

export async function login(credentials: Credentials) {
	try {
		const response = await fetch(`${env.PUBLIC_IDENTITY_SERVICE}/sign-in`, {
			method: 'POST',
			body: JSON.stringify(credentials),
			headers: {
				'content-type': 'application/json'
			},
			credentials: 'include'
		}).then((r) => r.json());

		accessToken.set(response.access_token);

		console.log(response);
	} catch (e) {
		console.log(e);
	}
}

export async function logout() {
	try {
		const accessToken = await getAccessToken();

		if (accessToken === null) return;

		await fetch(`${env.PUBLIC_IDENTITY_SERVICE}/sign-out`, {
			method: 'POST',
			headers: {
				Authorization: `Bearer ${accessToken}`
			},
			credentials: 'include'
		});
	} catch (e) {
		console.log(e);
	} finally {
		canAccessProtectedRoutes.set(false);
	}
}
