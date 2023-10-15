<script lang="ts">
	import type { PageData } from './$types';

	export let data: PageData;
	$: ({ channelsResponse } = data);

	let channels: typeof channelsResponse.channels | null;
	$: channels = channelsResponse?.channels || null;

	$: console.log(data);
</script>

<div class="flex flex-wrap flex-row gap-3 cursor-pointer">
	{#if channels}
		{#each channels as channel}
			<a href="/s/{channel.username}">
				<div
					class="animation theme-bg-base theme-fg-base theme-shadow-base stream-card rounded-md"
				/>
				<div class="px-2 pt-1">
					{channel.username}
				</div>
			</a>
		{/each}
	{:else}
		<p>Empty streams</p>
	{/if}
</div>

<style lang="scss">
	$stream-card-w: 21rem;
	$stream-card-h: 11rem;

	.stream-card {
		max-width: $stream-card-w;
		min-width: $stream-card-w;

		max-height: $stream-card-h;
		min-height: $stream-card-h;
	}

	.animation {
		&:hover {
			background-position: 100% 100%, 0 100%;
			background-size: 0 3px, 100% 3px;
		}

		background-image: linear-gradient(transparent, transparent),
			linear-gradient(var(--color-animation-default), var(--color-animation-default));

		background-position: 100% 100%, 0 100%;
		background-repeat: no-repeat;
		background-size: 100% 3px, 0 3px;
		border-bottom-width: 0;

		transition: background-size 0.5s ease-in-out, background-position 0.5s ease-in-out;
	}
</style>
