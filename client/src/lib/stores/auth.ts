import { env } from '$env/dynamic/public';

type Credentials = { username: string, password: string }


export async function login(credentials: Credentials) {
	try {
		const response = await fetch(`${env.PUBLIC_IDENTITY_SERVICE}/sign-in`, {
			method: 'POST',
			body: JSON.stringify(credentials),
			headers: {
				"content-type": "application/json",
			},
			credentials: "include",
		}).then(r => r.json());

		console.log(response)
	} catch (e) {
		console.log(e)
	}
}
