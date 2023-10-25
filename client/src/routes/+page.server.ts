import type { Fetch } from "$lib/utils/fetch"
import type { ServerLoad } from "@sveltejs/kit"
import { env } from '$env/dynamic/private';

const PRIVATE_STREAM_CHANNELS_ROUTE = `${env.STREAM_SERVICE}/stream-channels` as const;

type Egress = {
	egress: {
		id: string;
		type: string;
	};
}

type Stream = {
	active_stream_id: string;
	username: string;
	egresses: Array<Egress>;
}

type ChannelsResponse = {
	channels: Array<Stream>;
}

async function loadChannels(fetch: Fetch): Promise<ChannelsResponse | null> {
	try {
		return await fetch(PRIVATE_STREAM_CHANNELS_ROUTE).then(r => r.json());
	} catch (_) {
		return null;
	}
}

export const load: ServerLoad = async ({ fetch }) => {
	return {
		channelsResponse: loadChannels(fetch),
	};
}
