package media

import "io"

// Write data to target writer
type TargetMediaWriter struct {
	target io.Writer
}

var _ MediaWriter = (*TargetMediaWriter)(nil)

func (w *TargetMediaWriter) Write(p []byte) (n int, err error) {
	return w.target.Write(p)
}

func NewTargetMediaWriter(target io.Writer) *TargetMediaWriter {
	return &TargetMediaWriter{target: target}
}

