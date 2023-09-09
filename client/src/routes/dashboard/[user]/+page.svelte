<script lang="ts">
	import { env } from '$env/dynamic/public';
	import type { PageData } from './$types';
	import { accessToken } from '$lib/stores/auth';
	import { onMount } from 'svelte';
	import { HandleDashboardWebsocket } from '$lib/websocket/registry';
	import { streamStatus } from '$lib/stores/dashboard';

	export let data: PageData;

	$: ({ fetch } = data);

	const server = {
		ingestTemplate: 'alpine-template'
	};
	let ws: WebSocket;

	onMount(() => {
		try {
			const wsConn = env.PUBLIC_STREAM_SERVICE.replace('http://', 'ws://');
			ws = new WebSocket(`${wsConn}/stream:channel`);

			ws.onmessage = HandleDashboardWebsocket;

			ws.onclose = function (evt) {
				console.log('Close', evt);
			};

			ws.onerror = function (evt) {
				console.log('on error', evt);
			};

			return () => ws.close();
		} catch (err) {
			console.log(err);
		}
	});

	onMount(() =>
		streamStatus.subscribe((data) => {
			console.log(data)
		})
	);

	async function streamStart(): Promise<void> {
		const resp = await fetch(`${env.PUBLIC_STREAM_SERVICE}/stream:start`, {
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
		const resp = await fetch(`${env.PUBLIC_STREAM_SERVICE}/stream:stop`, {
			body: JSON.stringify(server),
			headers: {
				Authorization: `Bearer ${$accessToken}`,
				'Content-Type': 'application/json'
			},
			method: 'POST'
		});

		console.log(resp);
	}

	function sendWsMessage() {
		ws.send(JSON.stringify({ tset: 'hello world' }));
	}
</script>

<button on:click={streamStart}>Start stream</button>
<button on:click={streamStop}>Stop stream</button>
<button on:click={sendWsMessage}>Send to ws</button>
