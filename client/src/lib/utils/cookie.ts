import type { Cookies } from "@sveltejs/kit";

export async function mapCookiesFromHeader(cookies: Cookies, cookieSetHeader: string) {
	const cookieParts = [...cookieSetHeader.matchAll(/(.*?); Path=(.*?); Expires=(.*?); HttpOnly,? ?/gm)]

	cookieParts.forEach(part => {
		const [_, keyVal, path, date] = part;
		const [key, val] = keyVal.split("=")

		cookies.set(key, val, {
			expires: new Date(date),
			secure: true,
			path: path,
			httpOnly: true,
		});
	});
} 
