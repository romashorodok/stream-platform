import * as channelpb from '$gen/streaming/v1alpha/channel';
import { writable, type Readable, type Writable } from 'svelte/store';

interface WebsocketMessage {
	Process(data: string): void
	RegistryName(): string;
}

interface SvelteWritableMessage<F> {
	writable: Writable<F | undefined>
}

type WebsocketStatefulMessage<F> = SvelteWritableMessage<F> & WebsocketMessage

abstract class WebsocketMessageBase<F> implements SvelteWritableMessage<F> {
	readonly writable: Writable<F>;

	constructor() {
		this.writable = writable<F>(undefined)
	}

	abstract RegistryName(): string;
}

type StreamStatusBaseType = channelpb.StreamStatus;

export class StreamStatus extends WebsocketMessageBase<StreamStatusBaseType> implements WebsocketMessage {

	RegistryName(): string {
		return channelpb.StreamStatus.typeName;
	}

	Process(data: string): void {
		const streamStatus = channelpb.StreamStatus.fromJsonString(data)

		this.writable.set(streamStatus)
	}
}

class WebsocketMesssageRegistry {
	private readonly registry = new Map<string, WebsocketStatefulMessage<any>>();

	Register<F>(wsMessage: WebsocketStatefulMessage<F>) {
		this.registry.set(wsMessage.RegistryName(), wsMessage);
	}

	OnMessage(messageType: string, data: string) {
		const messageHandler = this.registry.get(messageType)

		if (!messageHandler) {
			throw new Error(`not found message handler of type: ${messageType}`)
		}

		messageHandler?.Process(data)
	}

	On<F>(message: WebsocketStatefulMessage<F>): Readable<F | undefined> {
		const state = this.registry.get(message.RegistryName());

		if (!state) {
			throw new Error(`not found message handler of type: ${message.RegistryName()}`)
		}

		return state.writable
	}
}

export async function HandleDashboardWebsocket(event: MessageEvent<any>) {
	const buffer = await event.data.arrayBuffer();
	const uint8Array = new Uint8Array(buffer);

	const decoder = new TextDecoder('utf-8');
	const message = decoder.decode(uint8Array);
	const wsProto: { type: string; data: string } = JSON.parse(message);

	dashboardRegistry.OnMessage(wsProto.type, wsProto.data);
}

export const streamStatusMessage = new StreamStatus();

export const dashboardRegistry = new WebsocketMesssageRegistry()
dashboardRegistry.Register(streamStatusMessage)

