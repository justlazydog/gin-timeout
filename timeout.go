package timeout

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultTimeout = 3 * time.Second
	defaultCode    = 503
	defaultMsg     = "<html><head><title>Timeout</title></head><body><h1>Timeout</h1></body></html>"
)

type Option func(*Timeout)

type Timeout struct {
	timeout time.Duration
	code    int
	msg     string
}

func WithTimeout(timeout time.Duration) Option {
	return func(t *Timeout) {
		t.timeout = timeout
	}
}

func WithResponseCode(code int) Option {
	return func(t *Timeout) {
		t.code = code
	}
}

func WithResponseMsg(msg string) Option {
	return func(t *Timeout) {
		t.msg = msg
	}
}

func New(opts ...Option) gin.HandlerFunc {
	t := &Timeout{
		timeout: defaultTimeout,
		code:    defaultCode,
		msg:     defaultMsg,
	}

	for _, opt := range opts {
		opt(t)
	}

	return func(c *gin.Context) {
		ctx, cancelCtx := context.WithTimeout(c.Request.Context(), t.timeout)
		defer cancelCtx()
		c.Request = c.Request.WithContext(ctx)

		tw := &timeoutWriter{
			ResponseWriter: c.Writer,
			h:              make(http.Header),
		}
		c.Writer = tw

		done := make(chan struct{})
		panicChan := make(chan interface{}, 1)

		go func() {
			defer func() {
				if p := recover(); p != nil {
					panicChan <- p
				}
			}()
			c.Next()
			close(done)
		}()

		select {
		case p := <-panicChan:
			panic(p)
		case <-ctx.Done():
			tw.mu.Lock()
			defer tw.mu.Unlock()
			tw.ResponseWriter.WriteHeader(t.code)
			io.WriteString(tw.ResponseWriter, t.msg)
			tw.timedOut = true
			c.Abort()
		case <-done:
			tw.mu.Lock()
			defer tw.mu.Unlock()
			dst := c.Writer.Header()
			for k, vv := range tw.h {
				dst[k] = vv
			}
			if !tw.wroteHeader {
				tw.code = http.StatusOK
			}
			tw.ResponseWriter.WriteHeader(tw.code)
			tw.ResponseWriter.Write(tw.wbuf.Bytes())
		}
	}
}
