package usagedb

import (
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// retentionFirstDelay runs the first prune shortly after start-up.
	retentionFirstDelay = time.Minute
	// retentionInterval reruns the prune daily.
	retentionInterval = 24 * time.Hour
	// retentionPause rests between prune batches.
	retentionPause = 100 * time.Millisecond
)

// startRetentionLocked runs the retention job: once after start-up, then
// daily. Callers hold mu.
func startRetentionLocked() {
	if retentionStop != nil {
		close(retentionStop)
	}
	stop := make(chan struct{})
	retentionStop = stop
	db := collectorDB
	go retentionLoop(stop, db)
}

func retentionLoop(stop <-chan struct{}, db *Store) {
	timer := time.NewTimer(retentionFirstDelay)
	defer timer.Stop()
	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			pruneOnce(db)
			timer.Reset(retentionInterval)
		}
	}
}

// pruneOnce deletes raw events older than the retention window in small
// batches with a pause between them. Rollups stay. Retention 0 or below
// keeps all rows. It records the prune time even when nothing is deleted.
func pruneOnce(db *Store) {
	if db == nil {
		return
	}
	days := collectorDays.Load()
	if days <= 0 {
		return
	}
	cutoff := unixMilliNow() - days*int64(24*time.Hour/time.Millisecond)
	for {
		deleted, err := db.PruneBefore(cutoff, pruneBatchSize)
		if err != nil {
			log.WithError(err).Error("usagedb: retention prune failed")
			return
		}
		if deleted == 0 {
			break
		}
		lastPruneMs.Store(unixMilliNow())
		if deleted < pruneBatchSize {
			break
		}
		time.Sleep(retentionPause)
	}
	lastPruneMs.Store(unixMilliNow())
}
