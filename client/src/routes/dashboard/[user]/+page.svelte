<script lang="ts">
	import { env } from '$env/dynamic/public';
	import type { PageData } from './$types';
	import { accessToken } from '$lib/stores/auth';
	import { onMount } from 'svelte';
	import { HandleDashboardWebsocket } from '$lib/websocket/registry';
	import { streamStatus } from '$lib/stores/dashboard';
	// @ts-ignore
	import shaka from 'shaka-player/dist/shaka-player.compiled.debug';
	import { addToast } from '$lib/components/base/toast.svelte';
	import Button from '$lib/components/base/button.svelte';
	import LoadingDots from '$lib/components/base/loading-dots.svelte';
	import type { StreamStatus } from '$gen/streaming/v1alpha/channel';

	export let data: PageData;

	$: ({ fetch } = data);

	const server = {
		ingestTemplate: 'alpine-template'
	};
	let ws: WebSocket;
	let loading: boolean = false;
	let firstLoad: boolean = true;
	let status: StreamStatus | undefined;

	onMount(async () => {
		await shaka.polyfill.installAll();

		let player: shaka.Player = new shaka.Player(video);

		const manifest = `http://localhost:8089/api/egress/hls`;
		player.load(manifest).catch(console.error);

		// const manifest = `http://${$identity?.sub}.localhost:9002/api/live/hls`;

		// player.load(manifest).catch((err) => console.error(err));
	});

	onMount(() => {
		try {
			const wsConn = env.PUBLIC_STREAM_SERVICE.replace('http://', 'ws://');
			ws = new WebSocket(`${wsConn}/stream:channel`);

			ws.onmessage = async function (evt: MessageEvent<any>) {
				await HandleDashboardWebsocket(evt);
				setTimeout(() => {
					firstLoad = false;
				}, 0);
				console.log('first load');
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

	let video: HTMLVideoElement;
	let videoContainer: HTMLDivElement;
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
			{:else}
				<LoadingDots />
			{/if}
		</div>
	</div>
</div>

<!-- <Button click={() => console.log('test')}> -->
<!-- 	<span>test</span> -->
<!-- </Button> -->

<!-- <button on:click={streamStart}>Start stream</button> -->
<!-- <button on:click={streamStop}>Stop stream</button> -->
<!-- <button on:click={sendWsMessage}>Send to ws</button> -->

<!-- <button on:click={create}>Show toast</button> -->

<!-- <div bind:this={videoContainer} data-shaka-player-container style="max-width:40em"> -->
<!-- 	<video controls bind:this={video} data-shaka-player style="width:100%;height:100%"> -->
<!-- 		<track kind="captions" /> -->
<!-- 	</video> -->
<!-- </div> -->

<!-- <button -->
<!-- 	class="inline-flex items-center justify-center rounded-xl bg-white px-4 py-3 -->
<!--   font-medium leading-none text-magnum-700 shadow hover:opacity-75" -->
<!-- > -->
<!-- 	Open Dialog -->
<!-- </button> -->
