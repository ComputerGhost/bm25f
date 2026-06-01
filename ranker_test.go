package bm25f_test

import (
	"cmp"
	"slices"
	"strings"
	"testing"

	"github.com/computerghost/bm25f"
)

func TestRanker(t *testing.T) {
	t.Parallel()

	t.Run("zero value", func(t *testing.T) {
		_ = (&bm25f.Ranker{}).SetK1(0.0)
		_ = (&bm25f.Ranker{}).SetB("", 0.0)
		(&bm25f.Ranker{}).SetWeight("", 0.0)
	})

	t.Run("config values out of range", func(t *testing.T) {
		bm := bm25f.NewRanker()
		if err := bm.SetK1(-1); err == nil {
			t.Error("expected error for k1 = -1")
		}

		bm = bm25f.NewRanker()
		if err := bm.SetB("", -1); err == nil {
			t.Error("expected error for b = -1")
		}

		bm = bm25f.NewRanker()
		if err := bm.SetB("", 1.1); err == nil {
			t.Error("expected error for b = 1.1")
		}
	})
}

func TestBM25F_Rank(t *testing.T) {
	t.Parallel()

	bm := bm25f.NewRanker()
	bm.SetWeight("title", 2.0)
	bm.SetWeight("body", 1.0)
	_ = bm.SetB("title", 0)
	_ = bm.SetB("body", 0)

	emptyDoc := bm25f.Document{}
	emptyDoc.SetStream("title", []string{})
	emptyDoc.SetStream("body", []string{})

	helloDoc := bm25f.Document{}
	helloDoc.SetStream("title", []string{"hello"})
	helloDoc.SetStream("body", []string{"hello", "blue", "world"})

	natureDoc := bm25f.Document{}
	natureDoc.SetStream("title", []string{"nature"})
	natureDoc.SetStream("body", []string{"blue", "tulip", "blue", "sky", "world"})

	const nonzero = 1.0

	tests := []struct {
		name      string
		documents map[string]bm25f.Document
		query     string
		want      []bm25f.Result
	}{
		{
			name:      "no documents",
			documents: map[string]bm25f.Document{},
			query:     "test",
			want:      []bm25f.Result{},
		},
		{
			name:      "empty query",
			documents: map[string]bm25f.Document{},
			query:     "",
			want:      []bm25f.Result{},
		},
		{
			name: "empty fields",
			documents: map[string]bm25f.Document{
				"empty2": emptyDoc,
				"empty1": emptyDoc,
			},
			query: "test",
			want: []bm25f.Result{
				{Id: "empty1", Score: 0},
				{Id: "empty2", Score: 0},
			},
		},
		{
			name: "single match",
			documents: map[string]bm25f.Document{
				"empty":  emptyDoc,
				"nature": natureDoc,
			},
			query: "tulip",
			want: []bm25f.Result{
				// Only natureDoc has the word "tulip".
				{Id: "nature", Score: nonzero},
				{Id: "empty", Score: 0},
			},
		},
		{
			name: "multiple matches",
			documents: map[string]bm25f.Document{
				"empty":  emptyDoc,
				"nature": natureDoc,
				"hello":  helloDoc,
			},
			query: "world",
			want: []bm25f.Result{
				// helloDoc and natureDoc both have one "word" in the body,
				// so they are sorted alphabetically by title.
				{Id: "hello", Score: nonzero},
				{Id: "nature", Score: nonzero},
				{Id: "empty", Score: 0},
			},
		},
		{
			name: "overused word",
			documents: map[string]bm25f.Document{
				"empty":  emptyDoc,
				"hello":  helloDoc,
				"nature": natureDoc,
			},
			query: "blue",
			want: []bm25f.Result{
				// natureDoc and helloDoc both contain the word "blue",
				// but the word is more frequent in natureDoc.
				{Id: "nature", Score: nonzero},
				{Id: "hello", Score: nonzero},
				{Id: "empty", Score: 0},
			},
		},
		{
			name: "multiword query",
			documents: map[string]bm25f.Document{
				"empty":  emptyDoc,
				"hello":  helloDoc,
				"nature": natureDoc,
			},
			query: "hello blue",
			want: []bm25f.Result{
				// natureDoc and helloDoc both contain the world "blue",
				// but helloDoc also contains "hello" in its title.
				{Id: "hello", Score: nonzero},
				{Id: "nature", Score: nonzero},
				{Id: "empty", Score: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			corpus := bm25f.Corpus{}
			for filename, document := range tt.documents {
				corpus.Upsert(filename, document)
			}

			scores := bm.Score(corpus, strings.Split(tt.query, " "))

			if len(scores) != len(tt.want) {
				t.Errorf("expected %d results, got %d", len(tt.want), len(scores))
			}

			// Sort the results descending by score.
			slices.SortFunc(scores, func(a, b bm25f.Result) int {
				if c := cmp.Compare(b.Score, a.Score); c != 0 {
					return c
				}
				return cmp.Compare(a.Id, b.Id)
			})

			for i, got := range scores {
				want := tt.want[i]
				if got.Id != want.Id {
					t.Errorf("rank #%d: expected %s got %s", i, want.Id, got.Id)
				}
				if want.Score == 0 && got.Score != 0 {
					t.Errorf("%q score: expected 0 got %v", got.Id, got.Score)
				} else if want.Score == nonzero && got.Score == 0 {
					t.Errorf("%q score: expected nonzero, got 0", got.Id)
				}
			}
		})
	}
}
