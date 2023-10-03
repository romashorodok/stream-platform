<script lang="ts">
	export let absolute = true;
</script>

<div class={`${absolute ? 'absolute inset-0' : ''} loading`}>
	<div class="flex m-auto">
		<span class="dot" />
		<span class="dot" />
		<span class="dot" />
	</div>
</div>

<style lang="scss" scoped>
	.loading {
		display: flex;
		align-items: center;
	}

	.dot {
		width: 8px;
		height: 8px;
		margin: 0 2px;
		background-color: white;
		border-radius: 50%;
		animation: bounce 1.2s infinite;
	}

	.dot:nth-child(1) {
		animation-delay: 0.4s;
	}

	.dot:nth-child(2) {
		animation-delay: 0.6s;
	}

	.dot:nth-child(3) {
		animation-delay: 0.8s;
	}

	@keyframes bounce {
		0%,
		100% {
			transform: translateY(0px);
		}
		50% {
			transform: translateY(-4px);
		}
	}
</style>
