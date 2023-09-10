// See https://kit.svelte.dev/docs/types#app
// for information about these interfaces

import type { IdentityTokenPayload } from '$lib/stores/auth';

declare global {
	namespace App {
		// interface Error {}
		// interface Platform {}

		interface PageData {
			fetch: (input: RequestInfo | URL, init?: RequestInit | undefined) => Promise<Response>
		}

		interface Locals {
			identityPayload: IdentityTokenPayload | null

			getAccessToken: function(): Promise<String | null>
		}
	}
}

export {};
