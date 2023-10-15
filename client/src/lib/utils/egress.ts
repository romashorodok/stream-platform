import { StreamEgressType } from '$gen/streaming/v1alpha/stream_channels';

type StreamEgressKeys = keyof typeof StreamEgressType;
type StreamEgress = typeof StreamEgressType[StreamEgressKeys];

const streamEgressKeys: StreamEgressKeys[] = Object.keys(StreamEgressType)
	.filter((key) => !isNaN(Number(key)))
	.map(key => key as StreamEgressKeys);

export const STREAM_EGRESS: StreamEgress[] = streamEgressKeys.map((key) => StreamEgressType[key]);

export const STREAM_EGRESS_TYPE = StreamEgressType

export const HLS_EGRESS: string = STREAM_EGRESS[STREAM_EGRESS_TYPE.STREAM_TYPE_HLS] as any;

export const WEBRTC_EGRESS: string = STREAM_EGRESS[STREAM_EGRESS_TYPE.STREAM_TYPE_WEBRTC] as any;
