package bm25f

import (
	"encoding/json"
	"sync"
)

type SyncCorpus struct {
	snapshot   *Corpus
	snapshotMu sync.RWMutex
	cloneMu    sync.Mutex
}

// NewSyncCorpus wraps a corpus in thread-safe functions.
// The corpus passed to this function must not be modified directly after this
// function call; instead, the functions of SyncCorpus should be used.
func NewSyncCorpus(corpus *Corpus) *SyncCorpus {
	return &SyncCorpus{
		snapshot: corpus.clone(),
	}
}

// Documents returns a map from document id to Document.
// The returned map should be considered immutable.
func (c *SyncCorpus) Documents() map[string]*Document {
	c.snapshotMu.RLock()
	defer c.snapshotMu.RUnlock()

	return c.Documents()
}

// Len returns the number of documents in the corpus.
func (c *SyncCorpus) Len() int {
	c.snapshotMu.RLock()
	defer c.snapshotMu.RUnlock()

	return len(c.snapshot.Documents())
}

func (c *SyncCorpus) MarshalJSON() ([]byte, error) {
	c.snapshotMu.RLock()
	defer c.snapshotMu.RUnlock()

	return json.Marshal(c.snapshot)
}

func (c *SyncCorpus) UnmarshalJSON(data []byte) error {
	ss := &Corpus{}
	if err := json.Unmarshal(data, ss); err != nil {
		return err
	}

	c.cloneMu.Lock()
	defer c.cloneMu.Unlock()

	c.snapshotMu.Lock()
	c.snapshot = ss
	c.snapshotMu.Unlock()

	return nil
}

// Remove removes all data associated with a document.
func (c *SyncCorpus) Remove(id string) {
	c.modify(func(ss *Corpus) {
		ss.Remove(id)
	})
}

// Upsert processes and adds a document into the corpus.
// The document must not be changed after passing it to this function.
func (c *SyncCorpus) Upsert(id string, document *Document) {
	c.modify(func(ss *Corpus) {
		ss.Upsert(id, document)
	})
}

func (c *SyncCorpus) modify(action func(s *Corpus)) {
	c.cloneMu.Lock()
	defer c.cloneMu.Unlock()

	c.snapshotMu.RLock()
	clone := c.snapshot.clone()
	c.snapshotMu.RUnlock()

	action(clone)

	c.snapshotMu.Lock()
	c.snapshot = clone
	c.snapshotMu.Unlock()
}

// Snapshot returns a readonly snapshot of the corpus at its current state.
// The returned value should be considered immutable.
//
// This snapshot should be passed to BM25F.Score for speed.
func (c *SyncCorpus) Snapshot() *Corpus {
	c.snapshotMu.RLock()
	defer c.snapshotMu.RUnlock()

	return c.snapshot
}
