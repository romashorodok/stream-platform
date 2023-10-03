<script context="module" lang="ts">
	type EventHandler<E extends Event = Event, T extends EventTarget = Element> = (
		event: E & { currentTarget: EventTarget & T }
	) => any;
	type MouseEventHandler<T extends EventTarget> = EventHandler<MouseEvent, T>;
</script>

<script lang="ts">
	export let click: MouseEventHandler<HTMLButtonElement> = () => {};
	export let className: string = '';
</script>

<button
	on:click={click}
	class={`${className} theme-bg-base theme-fg-base theme-shadow-base px-4 py-1 rounded-md`}
>
	{#if $$slots.default}
		<div class="flex relative">
			<slot clas />
		</div>
	{:else}
		<span>Hello world</span>
	{/if}
</button>

<style lang="scss" scoped>
	button:hover {
		background: var(--color-background-hover-base);
	}
</style>
