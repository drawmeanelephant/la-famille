package pack

import (
	"io"
	"net/http"
	"sync"
	"sync/atomic"
)

// Transfer counts received HTTP body bytes, not TLS framing or archive-member
// changes. URLs passed here have already passed credential-free confinement.
type Transfer struct {
	URL    string
	Status int
	Bytes  int64
}

type tracedTransport struct {
	base   http.RoundTripper
	report func(Transfer)
}

func (t tracedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		t.report(Transfer{URL: req.URL.String()})
		return nil, err
	}
	resp.Body = &tracedBody{
		ReadCloser: resp.Body,
		report: func(size int64) {
			t.report(Transfer{URL: req.URL.String(), Status: resp.StatusCode, Bytes: size})
		},
	}
	return resp, nil
}

type tracedBody struct {
	io.ReadCloser
	size   atomic.Int64
	once   sync.Once
	report func(int64)
}

func (b *tracedBody) Read(data []byte) (int, error) {
	n, err := b.ReadCloser.Read(data)
	b.size.Add(int64(n))
	return n, err
}

func (b *tracedBody) Close() error {
	err := b.ReadCloser.Close()
	b.once.Do(func() { b.report(b.size.Load()) })
	return err
}
