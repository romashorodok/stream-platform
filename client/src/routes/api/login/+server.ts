import { env as publicEnv } from '$env/dynamic/public';
import { json, type RequestHandler } from "@sveltejs/kit";
import { SESSION } from '../../../hooks.server';
import { dev } from '$app/environment';

async function getToken(): Promise<string> {
	return publicEnv.PUBLIC_TOKEN;
}

export const POST: RequestHandler = async ({ cookies }): Promise<any> => {

	const token = await getToken();

	cookies.set(SESSION, token, {
		path: '/',
		httpOnly: true,
		sameSite: 'strict',
		secure: !dev,
		maxAge: 60 * 60 * 24 * 7
	});

	return json({})
}
