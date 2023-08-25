<script lang="ts">
	import { goto } from '$app/navigation';
	import { redirect } from '@sveltejs/kit';

	export let condition: boolean;

	if (!condition) {
		redirect(300, '/');
	}

	$: if (!condition) {
		goto('/');
	}
</script>

{#if condition} <slot /> {/if}
