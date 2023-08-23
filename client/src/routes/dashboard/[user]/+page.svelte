<script lang="ts">
	import { env } from '$env/dynamic/public';
	import type { PageData } from './$types';

	const server = {
		ingestTemplate: 'alpine-template'
	};

	async function streamStart() {
		// TODO: Remove hardcode token
		const resp = await fetch(`${env.PUBLIC_STREAM_HOST}/stream:start`, {
			body: JSON.stringify(server),
			headers: {
				Authorization: `Bearer ${env.PUBLIC_TOKEN}`,
				'Content-Type': 'application/json'
			},
			method: 'POST'
		});

		console.log(resp);
	}

	export let data: PageData;

	$: console.log(data);
</script>

{data.user.accessToken}
{JSON.stringify(data.user.identity)}

<button on:click={streamStart}>Start stream</button>
