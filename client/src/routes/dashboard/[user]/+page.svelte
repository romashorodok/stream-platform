<script lang="ts">
	import { env } from '$env/dynamic/public';

	const data = {
		ingestTemplate: 'alpine-template',
	};

	async function streamStart() {
		// TODO: Remove hardcode token
		const resp = await fetch(`${env.PUBLIC_STREAM_HOST}/stream:start`, {
			body: JSON.stringify(data),
			headers: {
				Authorization: `Bearer ${env.PUBLIC_TOKEN}`,
				'Content-Type': 'application/json'
			},
			method: 'POST'
		});

		console.log(resp)
	}
</script>

<button on:click={streamStart}>Start stream</button>
