import type { LayoutLoad } from "./$types";

export const load: LayoutLoad = async ({ data }) => {

	return { user: { accessToken: data.accessToken, identity: data.identity } }
}

