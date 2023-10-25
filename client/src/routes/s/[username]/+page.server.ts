import type { Fetch } from "$lib/utils/fetch";
import type { ServerLoad } from "@sveltejs/kit";
import { env } from '$env/dynamic/private';

const PRIVATE_STREAM_CHANNELS_ROUTE = `${env.STREAM_SERVICE}/stream-channels` as const;

type EgressWithRoute = {
	egress: {
		id: string;
		type: string;
	};
	route: string;
}

type Stream = {
	active_stream_id: string;
	username: string;
	egresses: Array<EgressWithRoute>;
}

type ChannelResponse = {
	channel: Stream;
}

async function loadChannel(fetch: Fetch, username: string): Promise<ChannelResponse | null> {
	try {
		return await fetch(`${PRIVATE_STREAM_CHANNELS_ROUTE}/${username}`).then(r => r.json());
	} catch (_) {
		return null;
	}
}

export const load: ServerLoad = async ({ fetch, params }) => {
	const { username } = params;

	return {
		channelResponse: loadChannel(fetch, username),
	};
}
