package bm25f_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/computerghost/bm25f"
)

func TestBM25F(t *testing.T) {
	t.Parallel()

	t.Run("zero value", func(t *testing.T) {
		_ = (&bm25f.BM25F{}).SetK1(0.0)
		_ = (&bm25f.BM25F{}).SetB("", 0.0)
		(&bm25f.BM25F{}).SetWeight("", 0.0)
	})

	t.Run("config values out of range", func(t *testing.T) {
		bm := bm25f.NewBM25F()
		if err := bm.SetK1(-1); err == nil {
			t.Error("expected error for k1 = -1")
		}

		bm = bm25f.NewBM25F()
		if err := bm.SetB("", -1); err == nil {
			t.Error("expected error for b = -1")
		}

		bm = bm25f.NewBM25F()
		if err := bm.SetB("", 1.1); err == nil {
			t.Error("expected error for b = 1.1")
		}
	})
}

func TestBM25F_Rank(t *testing.T) {
	t.Parallel()

	bm := bm25f.NewBM25F()
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
			want:  []bm25f.Result{},
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
				{Id: "nature", Document: natureDoc},
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
				{Id: "hello", Document: helloDoc},
				{Id: "nature", Document: natureDoc},
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
				{Id: "nature", Document: natureDoc},
				{Id: "hello", Document: helloDoc},
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
				{Id: "hello", Document: helloDoc},
				{Id: "nature", Document: natureDoc},
			},
		},
	}

	extractNames := func(src []bm25f.Result) []string {
		results := make([]string, len(src))
		for i, result := range src {
			results[i] = result.Id
		}
		return results
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			corpus := bm25f.Corpus{}
			for filename, document := range tt.documents {
				corpus.Upsert(filename, document)
			}

			results := bm.Rank(corpus, strings.Split(tt.query, " "))

			gotNames := extractNames(results)
			wantNames := extractNames(tt.want)
			if !slices.Equal(gotNames, wantNames) {
				t.Errorf("Rank: got %#v, want %#v", gotNames, wantNames)
			}
		})
	}
}
