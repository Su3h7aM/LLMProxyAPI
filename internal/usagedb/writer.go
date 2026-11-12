package usagedb

import (
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// batchMaxRows flushes the writer queue after this many rows.
	batchMaxRows = 200
	// batchMaxWait flushes the writer queue after this delay.
	batchMaxWait = 250 * time.Millisecond

	// writeMaxTries bounds insert retries for one batch.
	writeMaxTries = 5
	// writeBaseBackoff is the first retry delay; it doubles to writeMaxBackoff.
	writeBaseBackoff = time.Second
	// writeMaxBackoff caps the retry delay.
	writeMaxBackoff = 30 * time.Second
)

// Writer buffers rows in a channel and inserts them in batches from one
// goroutine. Submit never blocks the caller: when the buffer is full the
// oldest row is dropped and the drop counter grows.
type Writer struct {
	store *Store
	ch    chan UsageRow

	maxTries   int
	baseBackoff time.Duration
	maxBackoff  time.Duration

	stopCh chan struct{}
	doneCh chan struct{}

	dropMu sync.Mutex

	dropped     atomic.Int64
	writeErrors atomic.Int64
	lastError   atomic.Value // string
}

// NewWriter creates a writer with the given channel capacity.
func NewWriter(store *Store, capacity int) *Writer {
	if capacity <= 0 {
		capacity = 1
	}
	return &Writer{
		store:       store,
		ch:          make(chan UsageRow, capacity),
		maxTries:    writeMaxTries,
		baseBackoff: writeBaseBackoff,
		maxBackoff:  writeMaxBackoff,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}
}

// Start runs the batch loop. Calling Start twice is safe.
func (w *Writer) Start() {
	if w == nil {
		return
	}
	go w.run()
}

// Submit queues one row. It never blocks: a full buffer drops the oldest row.
func (w *Writer) Submit(row UsageRow) {
	if w == nil {
		return
	}
	select {
	case w.ch <- row:
		return
	default:
	}
	w.dropMu.Lock()
	defer w.dropMu.Unlock()
	select {
	case w.ch <- row:
	default:
		select {
		case <-w.ch:
			w.dropped.Add(1)
		default:
		}
		w.ch <- row
	}
}

// Depth returns the current buffered row count.
func (w *Writer) Depth() int {
	if w == nil {
		return 0
	}
	return len(w.ch)
}

// Capacity returns the channel capacity.
func (w *Writer) Capacity() int {
	if w == nil {
		return 0
	}
	return cap(w.ch)
}

// Dropped counts rows dropped from a full buffer plus batches dropped after
// failed retries.
func (w *Writer) Dropped() int64 {
	if w == nil {
		return 0
	}
	return w.dropped.Load()
}

// WriteErrors counts failed insert attempts.
func (w *Writer) WriteErrors() int64 {
	if w == nil {
		return 0
	}
	return w.writeErrors.Load()
}

// LastError returns the last insert error text, or empty.
func (w *Writer) LastError() string {
	if w == nil {
		return ""
	}
	if value := w.lastError.Load(); value != nil {
		if text, ok := value.(string); ok {
			return text
		}
	}
	return ""
}

// Stop ends the batch loop and flushes queued rows within timeout.
// It waits for the flush or the timeout, whichever comes first.
func (w *Writer) Stop(timeout time.Duration) {
	if w == nil {
		return
	}
	select {
	case <-w.stopCh:
		return
	default:
		close(w.stopCh)
	}
	if timeout <= 0 {
		<-w.doneCh
		return
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-w.doneCh:
	case <-timer.C:
	}
}

func (w *Writer) run() {
	defer close(w.doneCh)
	batch := make([]UsageRow, 0, batchMaxRows)
	timer := time.NewTimer(batchMaxWait)
	defer timer.Stop()
	flush := func() {
		if len(batch) == 0 {
			return
		}
		w.writeWithRetry(batch)
		batch = batch[:0]
	}
	for {
		select {
		case <-w.stopCh:
			for {
				select {
				case row := <-w.ch:
					batch = append(batch, row)
					if len(batch) >= batchMaxRows {
						w.writeOnce(batch)
						batch = batch[:0]
					}
				default:
					if len(batch) > 0 {
						w.writeOnce(batch)
					}
					return
				}
			}
		case row := <-w.ch:
			batch = append(batch, row)
			if len(batch) >= batchMaxRows {
				flush()
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(batchMaxWait)
			}
		case <-timer.C:
			flush()
			timer.Reset(batchMaxWait)
		}
	}
}

// writeOnce inserts one batch with a single try and counts errors.
func (w *Writer) writeOnce(batch []UsageRow) {
	if _, err := w.store.InsertBatch(batch); err != nil {
		w.writeErrors.Add(1)
		w.lastError.Store(err.Error())
		log.WithError(err).Error("usagedb: shutdown flush failed")
		w.dropped.Add(int64(len(batch)))
	}
}

// writeWithRetry inserts one batch with backoff. After maxTries failed
// tries it drops the batch and counts it.
func (w *Writer) writeWithRetry(batch []UsageRow) {
	backoff := w.baseBackoff
	for try := 1; try <= w.maxTries; try++ {
		if _, err := w.store.InsertBatch(batch); err == nil {
			return
		} else if try == w.maxTries {
			w.writeErrors.Add(1)
			w.lastError.Store(err.Error())
			log.WithError(err).WithField("rows", len(batch)).Error("usagedb: batch failed, dropping it")
			w.dropped.Add(int64(len(batch)))
			return
		} else {
			w.writeErrors.Add(1)
			w.lastError.Store(err.Error())
			log.WithError(err).WithField("try", try).Warn("usagedb: batch failed, retrying")
		}
		select {
		case <-w.stopCh:
			w.dropped.Add(int64(len(batch)))
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > w.maxBackoff {
			backoff = w.maxBackoff
		}
	}
}
