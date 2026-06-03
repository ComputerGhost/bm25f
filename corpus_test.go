package bm25f_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/computerghost/bm25f"
)

func TestCorpus(t *testing.T) {
	t.Parallel()

	corpus := bm25f.NewCorpus()
	corpus.Upsert("one", bm25f.NewDocument(
		bm25f.WithField("body", []string{"one"}),
	))
	corpus.Upsert("two", bm25f.NewDocument(
		bm25f.WithField("body", []string{"two", "two"}),
	))
	corpus.Upsert("three", bm25f.NewDocument(
		bm25f.WithField("body", []string{"three", "three", "three"}),
	))

	if corpus.Len() != 3 {
		t.Errorf("Len() = %d, want 3", corpus.Len())
	}
	if _, ok := corpus.Document("one"); !ok {
		t.Fatal(`Document("one") ok = false, want true`)
	}

	corpus.Upsert("one", bm25f.NewDocument(
		bm25f.WithField("body", []string{"uno"}),
	))

	if corpus.Len() != 3 {
		t.Errorf("Len() after replace = %d, want 3", corpus.Len())
	}

	corpus.Remove("one")

	if corpus.Len() != 2 {
		t.Errorf("Len() after remove = %d, want 2", corpus.Len())
	}
	if _, ok := corpus.Document("one"); ok {
		t.Error(`Document("one") ok = true after Remove, want false`)
	}

	corpus.Remove("two")
	corpus.Remove("three")

	if corpus.Len() != 0 {
		t.Errorf("Len() after all removed = %d, want 0", corpus.Len())
	}

	corpus.Remove("missing")

	if corpus.Len() != 0 {
		t.Errorf("Len() after removed missing = %d, want 0", corpus.Len())
	}
}

func TestCorpus_ZeroValue(t *testing.T) {
	t.Parallel()

	tests := func(t *testing.T, corpus *bm25f.Corpus) {
		t.Helper()

		if _, ok := corpus.Document(""); ok {
			t.Errorf(`Document("") ok = true, want false`)
		}
		if count := len(corpus.DocumentIDs()); count != 0 {
			t.Errorf("DocumentIDs length = %d, want 0", count)
		}
		if corpus.Len() != 0 {
			t.Errorf("Len() = %d, want 0", corpus.Len())
		}

		corpus.Remove("missing")
		if corpus.Len() != 0 {
			t.Errorf("Len() after remove missing = %d, want 0", corpus.Len())
		}
	}

	emptyCorpus := bm25f.Corpus{}
	tests(t, &emptyCorpus)

	// Test unaltered NewCorpus too since it should have the same behavior.
	newCorpus := bm25f.NewCorpus()
	tests(t, newCorpus)
}

func TestCorpus_DocumentIDs(t *testing.T) {
	t.Parallel()

	corpus := bm25f.Corpus{}
	corpus.Upsert("charlie", &bm25f.Document{})
	corpus.Upsert("alpha", &bm25f.Document{})
	corpus.Upsert("bravo", &bm25f.Document{})

	got := corpus.DocumentIDs()
	want := []string{"alpha", "bravo", "charlie"}
	if !slices.Equal(got, want) {
		t.Errorf("DocumentIDs() = %v, want %v", got, want)
	}
}

func TestCorpus_JSON(t *testing.T) {
	t.Parallel()

	corpus := bm25f.NewCorpus()
	corpus.Upsert("hello", &bm25f.Document{})
	corpus.Upsert("goodbye", &bm25f.Document{})

	data, err := json.Marshal(corpus)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal() into raw map error: %v", err)
	}

	if _, ok := raw["documents"]; !ok {
		t.Error("marshaled corpus does not documents")
	}
	if _, ok := raw["docs_with_term"]; ok {
		t.Error("marshaled corpus contains docs_with_term, want only source documents")
	}
	if _, ok := raw["total_lengths"]; ok {
		t.Error("marshaled corpus contains total_lengths, want only source documents")
	}

	var rebuilt bm25f.Corpus
	if err := json.Unmarshal(data, &rebuilt); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if _, ok := rebuilt.Document("hello"); !ok {
		t.Errorf(`Document("hello") after JSON ok = false, want true`)
	}

	wantIDs := []string{"goodbye", "hello"}
	if got := rebuilt.DocumentIDs(); !slices.Equal(got, wantIDs) {
		t.Errorf("DocumentIDs() after JSON = %v, want = %v", got, wantIDs)
	}

	if got := rebuilt.Len(); got != 2 {
		t.Errorf("Len() after JSON = %d, want 2", got)
	}
}

func TestDocument_ZeroValue(t *testing.T) {
	t.Parallel()

	test := func(t *testing.T, doc *bm25f.Document) {
		t.Helper()

		if got := doc.Count("", ""); got != 0 {
			t.Errorf(`Count("", "") = %d, want 0`, got)
		}
		if got := doc.FieldLen(""); got != 0 {
			t.Errorf(`FieldLength("") = %v, want 0`, got)
		}
		if got := doc.FieldNames(); len(got) != 0 {
			t.Errorf("FieldNames() = %v, want empty", got)
		}
		if got, ok := doc.Metadata(""); ok || got != "" {
			t.Errorf(`Metadata("") = %q, %v; want "", false`, got, ok)
		}

		doc.SetField("", nil)
		doc.SetMetadata("", "")
	}

	doc := &bm25f.Document{}
	test(t, doc)

	// Test unaltered NewDocument too since it should have the same behavior.
	doc = bm25f.NewDocument()
	test(t, doc)
}

func TestDocument_Fields(t *testing.T) {
	t.Parallel()

	doc := bm25f.Document{}
	doc.SetField("title", []string{"hello"})
	doc.SetField("replaced", []string{"to be replaced"})
	doc.SetField("replaced", []string{"hello", "hello", "world"})
	doc.SetField("empty", nil)

	countTests := []struct {
		field string
		term  string
		want  int
	}{
		{field: "title", term: "hello", want: 1},
		{field: "title", term: "missing", want: 0},
		{field: "replaced", term: "hello", want: 2},
		{field: "replaced", term: "world", want: 1},
		{field: "replaced", term: "missing", want: 0},
		{field: "empty", term: "any", want: 0},
		{field: "missing", term: "any", want: 0},
	}

	for _, tt := range countTests {
		if got := doc.Count(tt.field, tt.term); got != tt.want {
			t.Errorf("Count(%q, %q) = %d, want %d", tt.field, tt.term, got, tt.want)
		}
	}

	fieldLenTests := []struct {
		field string
		want  int
	}{
		{field: "title", want: 1},
		{field: "replaced", want: 3},
		{field: "empty", want: 0},
		{field: "missing", want: 0},
	}

	for _, tt := range fieldLenTests {
		if got := doc.FieldLen(tt.field); got != tt.want {
			t.Errorf("FieldLen(%q) = %d, want %d", tt.field, got, tt.want)
		}
	}

	gotNames := doc.FieldNames()
	wantNames := []string{"empty", "replaced", "title"}
	if !slices.Equal(gotNames, wantNames) {
		t.Errorf("FieldNames() = %v, want %v", gotNames, wantNames)
	}
}

func TestDocument_Metadata(t *testing.T) {
	t.Parallel()

	doc := bm25f.Document{}
	doc.SetMetadata("1", "one")
	doc.SetMetadata("2", "two")
	doc.SetMetadata("1", "uno")

	if got, ok := doc.Metadata("1"); !ok || got != "uno" {
		t.Errorf(`Metadata("1") = %q, %v; want = "uno", false`, got, ok)
	}
	if got, ok := doc.Metadata("2"); !ok || got != "two" {
		t.Errorf(`Metadata("2") = %q, %v; want = "two", false`, got, ok)
	}
	if got, ok := doc.Metadata("missing"); ok {
		t.Errorf(`Metadata("missing") = %q, %v; want -, false`, got, ok)
	}
}

func TestDocument_JSON(t *testing.T) {
	t.Parallel()

	doc := bm25f.Document{}
	doc.SetField("title", []string{"hello"})
	doc.SetField("body", []string{"hello", "blue", "world", "blue"})
	doc.SetMetadata("title", "hello")

	data, err := json.Marshal(&doc)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var rebuilt bm25f.Document
	if err := json.Unmarshal(data, &rebuilt); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	tests := []struct {
		field string
		term  string
		want  int
	}{
		{field: "title", term: "hello", want: 1},
		{field: "body", term: "hello", want: 1},
		{field: "body", term: "blue", want: 2},
		{field: "body", term: "world", want: 1},
	}

	for _, tt := range tests {
		if got := doc.Count(tt.field, tt.term); got != tt.want {
			t.Errorf("Count(%q, %q) after JSON = %d, want %d",
				tt.field, tt.term, got, tt.want)
		}
	}

	if got := doc.FieldLen("body"); got != 4 {
		t.Errorf(`FieldLength("body") after JSON = %v, want 4`, got)
	}

	if title, ok := doc.Metadata("title"); !ok || title != "hello" {
		t.Errorf(`Metadata("title") after JSON = %q, %v; want "hello", true`, title, ok)
	}
}
