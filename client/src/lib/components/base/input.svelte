<script lang="ts">
	import type { Writable } from 'svelte/store';

	export let name: string;
	export let label: string;
	export let action: Function;
	export let state: Writable<any>;
	export let errors: Writable<Array<String>>;
</script>

<div>
	<label for={name}>{label}</label>

	<input use:action {name} bind:value={$state} />

	{#each $errors as error}
		<h1>{error}</h1>
	{/each}
</div>
