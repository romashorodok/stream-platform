<script lang="ts">
	import { env } from '$env/dynamic/public';
	import type { PageData } from './$types';
	import { accessToken } from '$lib/stores/auth';

	export let data: PageData;

	$: ({ fetch } = data);

	const server = {
		ingestTemplate: 'alpine-template'
	};

	async function streamStart() {
		const resp = await fetch(`${env.PUBLIC_STREAM_HOST}/stream:start`, {
			body: JSON.stringify(server),
			headers: {
				Authorization: `Bearer ${$accessToken}`,
				'Content-Type': 'application/json'
			},
			method: 'POST'
		});

		console.log(resp);
	}

	async function streamStop() {
		const resp = await fetch(`${env.PUBLIC_STREAM_HOST}/stream:stop`, {
			body: JSON.stringify(server),
			headers: {
				Authorization: `Bearer ${$accessToken}`,
				'Content-Type': 'application/json'
			},
			method: 'POST'
		});

		console.log(resp);
	}
</script>

<button on:click={streamStart}>Start stream</button>
<button on:click={streamStop}>Stop stream</button>
