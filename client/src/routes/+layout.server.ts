import type { LayoutServerLoad } from "./$types";

export const load: LayoutServerLoad = async ({ locals }) => {

	return {
		identity: locals.identityPayload || null,

		accessToken: locals.getAccessToken ? await locals.getAccessToken() : null,
	}
}

