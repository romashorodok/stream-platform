<script lang="ts">
	import '../app.css';
	import { goto } from '$app/navigation';

	import { logout } from '$lib/stores/auth';
	import { accessToken, canAccessProtectedRoutes } from '$lib/stores/auth';
	import { onMount } from 'svelte';

	onMount(() => {
		const unsubscribe = accessToken.subscribe((token) => {
			canAccessProtectedRoutes.set(!!token);
		});

		return unsubscribe;
	});

	async function dashboard() {
		goto('/dashboard/user');
	}

	async function login() {
		goto('/login');
	}
</script>

<div class="contents min-h-inherit">
	<div class="flex flex-col min-h-inherit max-h-screen">
		<nav
			class="flex-1 max-h-[50px] min-h-[50px] theme-bg-base theme-fg-base theme-shadow-base z-[100]"
		>
			<a href="/">Home</a>

			<button on:click={dashboard} type="button">My dashboard</button>
			<button on:click={login} type="button">Log In</button>
			<button on:click={logout} type="button">Log Out</button>
		</nav>
		<div class="flex flex-row flex-1 box-border overflow-hidden">
			<!-- <aside class="w-[220px] theme-bg-base theme-fg-base"> -->
			<!-- 	<button on:click={() => scheme.light()}>Light mode</button> -->
			<!-- 	<button on:click={() => scheme.dark()}>Dark mode</button> -->
			<!-- </aside> -->

			<main class="flex-1 overflow-y-scroll">
				<slot />
			</main>
		</div>
		<!-- <footer class="theme-bg-base theme-fg-base">Some content</footer> -->
	</div>
</div>

<style lang="postcass">
	:global(html) {
		min-height: 100%;
	}

	:global(body) {
		@apply theme-fg-body;
		@apply theme-bg-body;

		min-height: 100vh;
		overflow: hidden;
	}

	:global(.theme-bg-body) {
		background: var(--color-background-body);
	}

	:global(.theme-fg-body) {
		color: var(--color-text-body);
	}

	:global(.theme-fg-base) {
		color: var(--color-text-base);
	}

	:global(.theme-bg-base) {
		background: var(--color-background-base);
	}

	:global(.theme-shadow-base) {
		box-shadow: 0px 1px 10px var(--color-slate-dark-2);
	}

	:global(.theme-border-r-base) {
		border: 1px solid var(--color-slate-dark-2);
	}

	:global(.theme-bg-accent) {
		background: var(--color-background-accent);
	}

	:global(.theme-fg-accent) {
		color: var(--color-text-accent);
	}

	:global(.theme-bg-error) {
		background: var(--color-background-error);
	}

	:global(.theme-fg-error) {
		color: var(--color-text-error);
	}

	.min-h-inherit {
		min-height: inherit;
	}
</style>
