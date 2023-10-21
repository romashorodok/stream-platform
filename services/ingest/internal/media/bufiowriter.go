package media

import (
	"bufio"
	"io"
)

type BufioWriterCloser struct {
	writer *bufio.Writer
	closer io.WriteCloser
}

func NewBufioWriterCloser(w io.WriteCloser) *BufioWriterCloser {
	return &BufioWriterCloser{
		writer: bufio.NewWriter(w),
		closer: w,
	}
}

func (c *BufioWriterCloser) Write(p []byte) (n int, err error) {
	return c.writer.Write(p)
}

func (c *BufioWriterCloser) Close() error {
	err := c.writer.Flush()
	if err != nil {
		return err
	}
	return c.closer.Close()
}
