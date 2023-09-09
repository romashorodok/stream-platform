package subject

import "strings"

const StreamDestroyedNotification = "private.stream.destroyed.notification.*.empty"

func NewStreamDestroyedNotification(broadcasterID string) string {
	return strings.Replace(StreamDestroyedNotification, "*", broadcasterID, 1)
}
