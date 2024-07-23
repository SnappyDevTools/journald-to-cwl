package journal

import (
	"context"
	"errors"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
)

// ReaderAPI describes API that reads log entries from journald. The API is a subset of sdjournal.Journal.
type ReaderAPI interface {
	Next() (uint64, error)

	GetEntry() (*sdjournal.JournalEntry, error)

	// Wait blocks until the journal changed, up to timeout.
	Wait(timeout time.Duration) int
}

// Reader reads journal entry into a channel and let consumer consume the channel.
type Reader struct {
	reader ReaderAPI

	entries chan *sdjournal.JournalEntry

	// Time to wait for new entry.
	waitForDataTimeout time.Duration
}

func NewReader(reader ReaderAPI, opts ...Option) *Reader {
	r := Reader{
		reader:             reader,
		waitForDataTimeout: time.Second,
		entries:            make(chan *sdjournal.JournalEntry),
	}
	for _, opt := range opts {
		opt(&r)
	}
	return &r
}

// Entries returns a channel of JournalEntry read from journald. The returned channel is never closed.
func (r *Reader) Entries() <-chan *sdjournal.JournalEntry {
	return r.entries
}

// Read reads from log entries from journald and put them to the channel, until the ctx is canceled.
func (r *Reader) Read(ctx context.Context) {
	var errNoNewData = errors.New("no new data")
	next := func() (*sdjournal.JournalEntry, error) {
		advanced, err := r.reader.Next()
		if err != nil {
			return nil, err
		}
		if advanced == 0 {
			return nil, errNoNewData
		}
		return r.reader.GetEntry()
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			entry, err := next()
			switch err {
			case nil:
				r.entries <- entry
			case errNoNewData:
				r.reader.Wait(r.waitForDataTimeout)
			default:
				// We don't know how to deal with error reading journal.
				panic(err)
			}
		}
	}
}

type Option func(*Reader)

func WithWaitForDataTimeout(d time.Duration) Option {
	return func(r *Reader) {
		r.waitForDataTimeout = d
	}
}
