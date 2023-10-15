// place files you want to import through the `$lib` alias in this folder.

import { env } from '$env/dynamic/public';

export const _REFRESH_TOKEN = '_refresh_token' as const;

export const VERIFY_ROUTE = `${env.PUBLIC_IDENTITY_SERVICE}/token-revocation:verify` as const;
export const REFRESH_TOKEN_ROUTE = `${env.PUBLIC_IDENTITY_SERVICE}/access-token` as const;

export const STREAM_CHANNELS_ROUTE = `${env.PUBLIC_STREAM_SERVICE}/stream-channels` as const;
