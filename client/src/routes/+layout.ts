import { accessToken, canAccessProtectedRoutes, identity } from '$lib/stores/auth';
import { fetchIntercepted } from '$lib/utils/fetch';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ data }) => {
	accessToken.set(data.accessToken);
	canAccessProtectedRoutes.set(!!data.accessToken);
	identity.set(data.identity);

	return {
		fetch: fetchIntercepted()
	};
};
