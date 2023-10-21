import { writable, type Writable } from "svelte/store";

const mediaStream: Writable<MediaStream | null> = writable(null)
const connection: Writable<RTCPeerConnection | null> = writable(null)
const onConnection = connection.subscribe

const defaultScreenCaptureOpts: DisplayMediaStreamOptions = {
	video: {
		width: 1280,
		height: 720,
		frameRate: 30,
	},
	audio: false,
}

async function startScreenCapture(options: DisplayMediaStreamOptions = defaultScreenCaptureOpts): Promise<MediaStream> {
	try {
		return await navigator.mediaDevices.getDisplayMedia(options);
	} catch (err) {
		console.error(err)
		return null
	}
}

async function startAudioCapture(): Promise<MediaStream> {
	try {
		return await navigator.mediaDevices.getUserMedia({
			video: false,
			audio: true,
		})
	} catch (err) {
		console.error(err)
		return null
	}
}

const INGEST_API = "http://localhost:8089/api/ingress/whip"

async function startStreamConn() {
	try {
		const screenStream = await startScreenCapture()
		const audioStream = await startAudioCapture()

		const peerConnection = new RTCPeerConnection()

		connection.set(peerConnection)
		mediaStream.set(screenStream)

		peerConnection.oniceconnectionstatechange = _ => console.log(peerConnection.iceConnectionState)

		screenStream.addTrack(audioStream.getAudioTracks()[0])

		screenStream.getTracks().forEach(t => {
			if (t.kind === 'audio') {
				peerConnection.addTransceiver(t, { direction: 'sendonly' })
			} else {
				peerConnection.addTransceiver(t, {
					direction: 'sendonly',
				})
			}
		})

		peerConnection.createOffer().then((offer: RTCLocalSessionDescriptionInit) => {
			peerConnection.setLocalDescription(offer);
			console.log(offer)

			fetch(INGEST_API, {
				method: 'POST',
				body: offer.sdp,
				headers: {
					'Content-Type': 'application/sdp'
				}
			}).then(r => r.text()).then(answer => {
				peerConnection.setRemoteDescription({
					sdp: answer,
					type: 'answer',
				})
			})
		})
	} catch (e) {
		console.log(e)
	}
}

async function stopStreamConn() {
	console.log("Clean up")

	const csub = onConnection((conn) => {
		if (conn == null) {
			return
		}
		conn.getTransceivers().forEach(t => t.stop())
		conn.close()
	})

	const msub = mediaStream.subscribe((stream) => {
		if (stream == null) {
			return
		}
		stream.getTracks().forEach(t => t.stop())
	})

	msub()
	csub()

	mediaStream.set(null)
	connection.set(null)
}

export {
	onConnection,
	startStreamConn,
	stopStreamConn,
}
