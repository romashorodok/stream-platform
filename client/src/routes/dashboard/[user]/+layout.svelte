<script lang="ts">
	import AuthorizationGate from '$lib/components/base/AuthorizationGate.svelte';
	import { canAccessProtectedRoutes } from '$lib/stores/auth';
</script>

<AuthorizationGate condition={$canAccessProtectedRoutes}>
	<slot />
</AuthorizationGate>
