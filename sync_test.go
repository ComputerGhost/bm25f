package bm25f_test

import (
	"testing"

	"github.com/computerghost/bm25f"
)

// See corpus_test.go for helper functions.

func TestSyncCorpus(t *testing.T) {
	t.Parallel()
	testCorpus(t, bm25f.NewSyncCorpus(bm25f.NewCorpus()))
}

func TestSyncCorpus_Documents(t *testing.T) {
	t.Parallel()
	testCorpusDocuments(t, bm25f.NewSyncCorpus(bm25f.NewCorpus()))
}

func TestSyncCorpus_JSON(t *testing.T) {
	t.Parallel()
	testCorpusJSON(t, bm25f.NewSyncCorpus(bm25f.NewCorpus()))
}
