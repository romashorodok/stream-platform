<script lang="ts">
	import Player from '$lib/components/player/player.svelte';
	import Webrtc from '$lib/components/player/webrtc.svelte';
	import { STREAM_EGRESS, STREAM_EGRESS_TYPE, HLS_EGRESS, WEBRTC_EGRESS } from '$lib/utils/egress';
	import type { PageData } from './$types';

	export let data: PageData;
	$: ({ channelResponse } = data);

	let channel: typeof channelResponse.channel | null;
	$: channel = channelResponse?.channel || null;

	let preferEgress: string = STREAM_EGRESS[STREAM_EGRESS_TYPE.STREAM_TYPE_WEBRTC] as any;
</script>

{#if channel}
	{#each channel.egresses as { egress: { type }, route }}
		{#if type === preferEgress && HLS_EGRESS === preferEgress}
			<Player source={route} />
		{/if}
		{#if type === preferEgress && WEBRTC_EGRESS === preferEgress}
			<Webrtc source={route} />
		{/if}
	{/each}

	<select bind:value={preferEgress}>
		{#each channel.egresses as { egress: { type } }}
			<option value={type}>{type}</option>
		{/each}
	</select>

	<h3>{channel.username}</h3>
{/if}
