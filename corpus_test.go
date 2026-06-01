package bm25f_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/computerghost/bm25f"
)

func TestCorpus(t *testing.T) {
	t.Parallel()
	corpus := bm25f.Corpus{}

	assertCount := func(term string, want int) {
		t.Helper()
		if got := corpus.DocsWithTerm[term]; got != want {
			t.Errorf("DocsWithTerm[%q]: got %d, want %d", term, got, want)
		}
	}

	assertSize := func(want int) {
		t.Helper()
		if got := len(corpus.Documents); got != want {
			t.Errorf("len(Documents): got %d, want %d", got, want)
		}
	}

	createDocument := func(text string) bm25f.Document {
		doc := bm25f.Document{}
		doc.SetStream("", strings.Split(text, " "))
		return doc
	}

	// Populate corpus
	corpus.Upsert("one", createDocument("one"))
	corpus.Upsert("two", createDocument("two two"))
	corpus.Upsert("three", createDocument("three three three"))
	assertCount("three", 1)
	assertSize(3)
	if t.Failed() {
		t.FailNow()
	}

	// Replace existing document
	corpus.Upsert("one", createDocument("one two three four"))
	assertCount("three", 2)
	assertSize(3)
	if t.Failed() {
		t.FailNow()
	}

	// Remove document
	corpus.Remove("one")
	assertCount("one", 0)
	assertSize(2)
	if t.Failed() {
		t.FailNow()
	}

	// Remove remaining documents
	corpus.Remove("two")
	corpus.Remove("three")
	assertCount("two", 0)
	assertSize(0)
	if t.Failed() {
		t.FailNow()
	}

	// Remove nonexistent
	corpus.Remove("missing")
	assertSize(0)
}

func TestDocument(t *testing.T) {
	t.Parallel()

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()

		doc := bm25f.Document{}
		doc.SetAttachment("", "")

		doc = bm25f.Document{}
		doc.SetStream("", []string{})
	})

	t.Run("attachments", func(t *testing.T) {
		t.Parallel()

		doc := bm25f.Document{}
		doc.SetAttachment("1", "one")
		doc.SetAttachment("2", "two")
		doc.SetAttachment("1", "uno")

		want := map[string]string{
			"1": "uno",
			"2": "two",
		}
		if !maps.Equal(doc.Attachments, want) {
			t.Errorf("Attachments = %v, want %v", doc.Attachments, want)
		}
	})

	t.Run("streams", func(t *testing.T) {
		t.Parallel()

		doc := bm25f.Document{}
		doc.SetStream("1", []string{"one", "two"})
		doc.SetStream("2", []string{"two two"})
		doc.SetStream("1", []string{"uno"})

		gotIds := slices.Collect(maps.Keys(doc.Streams))
		wantIds := []string{"1", "2"}
		if !slices.Equal(gotIds, wantIds) {
			t.Errorf("Streams = %v, want %v", gotIds, wantIds)
		}
	})
}

func TestDocument_Streams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tokens     []string
		wantLength int
		wantCounts map[string]int
	}{
		{
			name:       "empty",
			tokens:     []string{},
			wantLength: 0,
			wantCounts: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := bm25f.Document{}
			doc.SetStream("", tt.tokens)

			if doc.Streams[""] == nil {
				t.Fatalf("Unable to set stream.")
			}
			s := doc.Streams[""]
			if s.Length != tt.wantLength {
				t.Errorf("Length = %v, want %v", s.Length, tt.wantLength)
			}
			if !maps.Equal(s.TermCounts, tt.wantCounts) {
				t.Errorf("TermCounts = %v, want %v", s.TermCounts, tt.wantCounts)
			}
		})
	}
}
