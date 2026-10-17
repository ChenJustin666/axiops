package observabilityaccess

import (
	"context"
	"io"
	"time"
)

// Lease is owned by one stream/writer. At most three seconds of already
// authorized output may be delivered before a permission/session recheck.
type Lease struct {
	ctx  context.Context
	next time.Time
}

func NewLease(ctx context.Context) *Lease { return &Lease{ctx: ctx} }
func (l *Lease) Check() error {
	if err := l.ctx.Err(); err != nil {
		return err
	}
	if time.Now().Before(l.next) {
		return nil
	}
	if err := Recheck(l.ctx); err != nil {
		return err
	}
	l.next = time.Now().Add(3 * time.Second)
	return nil
}

type guardedWriter struct {
	lease *Lease
	dst   io.Writer
}

func Writer(ctx context.Context, dst io.Writer) io.Writer { return &guardedWriter{NewLease(ctx), dst} }
func (w *guardedWriter) Write(p []byte) (int, error) {
	if err := w.lease.Check(); err != nil {
		return 0, err
	}
	return w.dst.Write(p)
}
