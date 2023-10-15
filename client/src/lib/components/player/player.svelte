<script lang="ts">
	import { onMount } from 'svelte';

	// @ts-ignore
	import shaka from 'shaka-player/dist/shaka-player.compiled.debug';

	export let source: string;

	onMount(() => {
		let player: shaka.Player = new shaka.Player(video);

		player.load(source).catch(console.error);

		return () => player.unload();
	});

	let video: HTMLVideoElement;
</script>

<div data-shaka-player-container style="w-full">
	<video autoplay controls bind:this={video} data-shaka-player style="width:100%;height:100%">
		<track kind="captions" />
	</video>
</div>
