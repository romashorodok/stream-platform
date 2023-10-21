<script lang="ts">
	import { env } from '$env/dynamic/public';
	import type { PageData } from './$types';
	import { accessToken } from '$lib/stores/auth';
	import { onMount } from 'svelte';
	import { HandleDashboardWebsocket } from '$lib/websocket/registry';
	import { streamStatus } from '$lib/stores/dashboard';
	import { addToast } from '$lib/components/base/toast.svelte';
	import Button from '$lib/components/base/button.svelte';
	import LoadingDots from '$lib/components/base/loading-dots.svelte';
	import type { StreamStatus } from '$gen/streaming/v1alpha/channel';
	import { startStreamConn, stopStreamConn } from '$lib/stores/studio';

	export let data: PageData;

	$: ({ fetch } = data);

	const server = {
		ingestTemplate: 'alpine-template'
	};
	let ws: WebSocket;
	let loading: boolean = false;
	let firstLoad: boolean = true;
	let status: StreamStatus | undefined;

	onMount(() => stopStreamConn);

	onMount(() => {
		try {
			const wsConn = env.PUBLIC_STREAM_SERVICE.replace('http://', 'ws://');
			ws = new WebSocket(`${wsConn}/stream:channel`);

			ws.onmessage = async function (evt: MessageEvent<any>) {
				await HandleDashboardWebsocket(evt);
				setTimeout(() => {
					firstLoad = false;
				}, 0);
			};

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

	onMount(() => streamStatus.subscribe((data) => (status = data)));

	$: console.log(status);

	async function streamStart(): Promise<void> {
		loading = true;

		const resp = await fetch(`${env.PUBLIC_STREAM_SERVICE}/stream:start`, {
			body: JSON.stringify(server),
			headers: {
				Authorization: `Bearer ${$accessToken}`,
				'Content-Type': 'application/json'
			},
			method: 'POST'
		});

		console.log(resp);
		loading = false;
	}

	async function streamStop() {
		loading = true;

		const resp = await fetch(`${env.PUBLIC_STREAM_SERVICE}/stream:stop`, {
			body: JSON.stringify(server),
			headers: {
				Authorization: `Bearer ${$accessToken}`,
				'Content-Type': 'application/json'
			},
			method: 'POST'
		});

		console.log(resp);
		loading = false;
	}

	function sendWsMessage() {
		ws.send(JSON.stringify({ tset: 'hello world' }));
	}

	function create() {
		addToast({
			data: {
				title: 'Success',
				description: 'The resource was created!',
				color: 'bg-green-500'
			}
		});
	}
</script>

<div class="h-full">
	<div class="flex flex-col justify-between">
		<div class="flex-[0]">
			{#if !firstLoad}
				{#if status?.deployed}
					<Button
						click={streamStop}
						className="theme-bg-accent-action theme-bg-hover-accent-action theme-fg-accent-action"
					>
						{#if loading}
							<LoadingDots />
						{/if}
						<span class={`${loading ? 'invisible' : ''}`}>End live</span>
					</Button>
				{:else}
					<Button
						click={streamStart}
						className="theme-bg-accent theme-bg-hover-accent theme-fg-accent"
					>
						{#if loading}
							<LoadingDots />
						{/if}
						<span class={`${loading ? 'invisible' : ''}`}>Go live</span>
					</Button>
				{/if}

				<div>
					<button on:click={startStreamConn}>Start</button>
				</div>
			{:else}
				<LoadingDots />
			{/if}
		</div>
	</div>
</div>
