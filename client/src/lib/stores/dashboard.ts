import { dashboardRegistry, streamStatusMessage } from '$lib/websocket/registry';

export const streamStatus = dashboardRegistry.On(streamStatusMessage);
