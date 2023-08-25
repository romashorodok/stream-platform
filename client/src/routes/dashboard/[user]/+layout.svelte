<script lang="ts">
	import AuthorizationGate from '$lib/components/base/AuthorizationGate.svelte';
	import { canAccessProtectedRoutes } from '$lib/stores/auth';
	import type { LayoutData } from './$types';

	export let data: LayoutData;
</script>

<AuthorizationGate condition={$canAccessProtectedRoutes}>
	<slot />
</AuthorizationGate>
