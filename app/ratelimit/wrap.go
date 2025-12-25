package ratelimit

import (
	"sync"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/transport"
)

type Direction int

const (
	Up   Direction = iota // client -> server (uplink)
	Down                  // server -> client (downlink)
)

type wrapReader struct {
	inner   buf.Reader
	conn    ConnID
	onClose func()
}

func (r *wrapReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	mb, err := r.inner.ReadMultiBuffer()

	if mb != nil && !mb.IsEmpty() {
		nBytes := int(mb.Len())

		ci := Global.Get(r.conn)
		if ci != nil {
			if limit, ok := Limits.GetForConn(ci.UUID, r.conn); ok {
				upBucket, _ := buckets.GetOrCreate(r.conn, limit.Up, limit.Down)
				upBucket.Wait(nBytes)
			}
		}

		Global.AddRx(r.conn, uint64(nBytes))
	}

	if err != nil && r.onClose != nil {
		r.onClose()
	}
	return mb, err
}

func (r *wrapReader) Interrupt() {
	if r.onClose != nil {
		r.onClose()
	}
	common.Interrupt(r.inner)
}

func (r *wrapReader) ReadMultiBufferTimeout(timeout time.Duration) (buf.MultiBuffer, error) {
	if tr, ok := r.inner.(buf.TimeoutReader); ok {
		mb, err := tr.ReadMultiBufferTimeout(timeout)
		if mb != nil && !mb.IsEmpty() {
			nBytes := int(mb.Len())
			ci := Global.Get(r.conn)
			if ci != nil {
				limit, ok := Limits.GetForConn(ci.UUID, r.conn)
				if ok {
					upBucket, _ := buckets.GetOrCreate(r.conn, limit.Up, limit.Down)
					upBucket.Wait(nBytes)
				}
			}
			Global.AddRx(r.conn, uint64(nBytes))
		}
		if err != nil {
			r.onClose()
		}
		return mb, err
	}
	// fallback
	return r.ReadMultiBuffer()
}

// wrapWriter считает bytes и обновляет LastSeen.
// При будущем throttling сюда же добавим limiter.Wait().
type wrapWriter struct {
	inner   buf.Writer
	conn    ConnID
	dir     Direction
	onClose func()
}

func (w *wrapWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	nBytes := int(mb.Len())
	if nBytes > 0 {
		ci := Global.Get(w.conn)
		if ci != nil {
			if limit, ok := Limits.GetForConn(ci.UUID, w.conn); ok {
				_, downBucket := buckets.GetOrCreate(w.conn, limit.Up, limit.Down)
				downBucket.Wait(nBytes)
			}
		}
		Global.AddTx(w.conn, uint64(nBytes))
	}
	err := w.inner.WriteMultiBuffer(mb)
	if err != nil && w.onClose != nil {
		w.onClose()
	}
	return err
}

func (w *wrapWriter) Close() error {
	if w.onClose != nil {
		w.onClose()
	}
	return common.Close(w.inner)
}

// onceFunc — гарантирует, что Remove выполнится один раз,
// даже если Close и Interrupt вызываются в разных местах.
func onceFunc(f func()) func() {
	var once sync.Once
	return func() { once.Do(f) }
}

// NewConn wraps link endpoints and returns conn_id.
func NewConn(uuid string, uplinkWriter buf.Writer, downlinkWriter buf.Writer, uplinkReader buf.Reader, downlinkReader buf.Reader) (ConnID, buf.Writer, buf.Writer, buf.Reader, buf.Reader) {
	ci := Global.Add(uuid)
	connID := ci.ConnID

	// “пульс” соединения
	ci.LastSeen.Store(time.Now().Unix())

	cleanup := onceFunc(func() {
		Global.Remove(connID)
		buckets.Remove(connID)
	})

	uw := &wrapWriter{inner: uplinkWriter, conn: connID, dir: Up, onClose: cleanup}
	dw := &wrapWriter{inner: downlinkWriter, conn: connID, dir: Down, onClose: cleanup}

	ur := &wrapReader{inner: uplinkReader, onClose: cleanup}
	dr := &wrapReader{inner: downlinkReader, onClose: cleanup}

	return connID, uw, dw, ur, dr
}

type wrappedConn interface {
	RateLimitConnID() ConnID
}

func (w *wrapReader) RateLimitConnID() ConnID { return w.conn }
func (w *wrapWriter) RateLimitConnID() ConnID { return w.conn }

func WrapLinkWithConnID(id ConnID, link *transport.Link) *transport.Link {
	// ВАЖНО: тут НЕ должно быть cleanup удаления registry/buckets,
	// потому что link-ов будет много, а ConnID один на inbound.
	// cleanup делаем один раз при закрытии inbound соединения.

	if link != nil {
		if r, ok := link.Reader.(wrappedConn); ok && r.RateLimitConnID() == id {
			return link
		}
		if w, ok := link.Writer.(wrappedConn); ok && w.RateLimitConnID() == id {
			return link
		}
	}

	link.Reader = &wrapReader{
		inner: link.Reader,
		conn:  id,
		// onClose можно оставить nil
	}

	link.Writer = &wrapWriter{
		inner: link.Writer,
		conn:  id,
		dir:   Down,
		// onClose nil
	}

	return link
}
