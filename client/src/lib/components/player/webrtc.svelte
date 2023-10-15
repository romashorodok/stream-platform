<script lang="ts">
	import { onMount } from 'svelte';

	export let source: string;

	onMount(() => {
		const peerConnection = new RTCPeerConnection({
			iceServers: []
		});

		peerConnection.ontrack = (evt: RTCTrackEvent) => {
			mediaSource = evt.streams[0];
		};

		peerConnection.addTransceiver('audio', { direction: 'recvonly' });
		peerConnection.addTransceiver('video', { direction: 'recvonly' });

		peerConnection.createOffer().then((offer: RTCLocalSessionDescriptionInit) => {
			peerConnection.setLocalDescription(offer);

			fetch(source, {
				method: 'POST',
				body: offer.sdp,
				headers: {
					'Content-Type': 'application/sdp'
				}
			})
				.then((r) => {
					return r.text();
				})
				.then((answer) => {
					peerConnection.setRemoteDescription({
						sdp: answer,
						type: 'answer'
					});
				});
		});

		return () => peerConnection.close();
	});

	let mediaSource: MediaProvider = null;
	$: if (video && mediaSource !== null) {
		video.srcObject = mediaSource;
	}

	let video: HTMLVideoElement;
</script>

<video bind:this={video} controls class="w-full" autoplay playsinline>
	<track kind="captions" />
</video>
