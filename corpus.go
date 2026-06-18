package bm25f

import (
	"encoding/json"
	"maps"
	"slices"
)

type Corpus interface {
	// Clone returns a shallow copy of the corpus.
	// That is, documents inside the corpus are not cloned.
	Clone() Corpus

	// DocsWithTerm returns the number of documents containing a term.
	DocsWithTerm(term string) int

	// Documents returns a map from document id to Document.
	// The returned map should be considered immutable.
	Documents() map[string]*Document

	// Len returns the number of documents in the corpus.
	Len() int

	// TotalLength returns the total length of a field across all documents.
	TotalLength(field string) int

	// Remove removes all data associated with a document.
	Remove(id string)

	// Upsert processes and adds a document into the corpus.
	// The document must not be changed after passing it to this function.
	Upsert(id string, document *Document)
}

// NewCorpus creates an empty SimpleCorpus.
func NewCorpus() Corpus {
	return NewSimpleCorpus()
}

type SimpleCorpus struct {
	documents    map[string]*Document
	docsWithTerm map[string]int
	totalLengths map[string]int
}

// NewSimpleCorpus creates an empty SimpleCorpus.
func NewSimpleCorpus() *SimpleCorpus {
	return &SimpleCorpus{
		documents:    make(map[string]*Document),
		docsWithTerm: make(map[string]int),
		totalLengths: make(map[string]int),
	}
}

func (c *SimpleCorpus) Clone() Corpus {
	return &SimpleCorpus{
		documents:    maps.Clone(c.documents),
		docsWithTerm: maps.Clone(c.docsWithTerm),
		totalLengths: maps.Clone(c.totalLengths),
	}
}

func (c *SimpleCorpus) DocsWithTerm(term string) int {
	return c.docsWithTerm[term]
}

func (c *SimpleCorpus) Documents() map[string]*Document {
	return c.documents
}

func (c *SimpleCorpus) Len() int {
	return len(c.documents)
}

func (c *SimpleCorpus) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Documents map[string]*Document `json:"documents"`
	}{
		Documents: c.documents,
	})
}

func (c *SimpleCorpus) TotalLength(field string) int {
	return c.totalLengths[field]
}

func (c *SimpleCorpus) UnmarshalJSON(data []byte) error {
	state := struct {
		Documents map[string]*Document `json:"documents"`
	}{}
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	c.documents = state.Documents
	c.docsWithTerm = make(map[string]int)
	c.totalLengths = make(map[string]int)

	for _, doc := range c.documents {
		c.addStats(doc)
	}

	return nil
}

func (c *SimpleCorpus) Remove(id string) {
	doc, ok := c.documents[id]
	if !ok {
		return
	}

	delete(c.documents, id)
	c.removeStats(doc)
}

func (c *SimpleCorpus) Upsert(id string, document *Document) {
	if old, ok := c.documents[id]; ok {
		c.removeStats(old)
	}

	c.documents[id] = document
	c.addStats(document)
}

func (c *SimpleCorpus) addStats(doc *Document) {
	for name, field := range doc.fields {
		c.totalLengths[name] += field.length

		for term := range field.termCounts {
			c.docsWithTerm[term]++
		}
	}
}

func (c *SimpleCorpus) removeStats(doc *Document) {
	for name, field := range doc.fields {
		c.totalLengths[name] -= field.length
		if c.totalLengths[name] == 0 {
			delete(c.totalLengths, name)
		}

		for term := range field.termCounts {
			c.docsWithTerm[term]--
			if c.docsWithTerm[term] == 0 {
				delete(c.docsWithTerm, term)
			}
		}
	}
}

type DocumentOption func(d *Document)

func WithField(name string, tokens []string) DocumentOption {
	return func(d *Document) {
		d.SetField(name, tokens)
	}
}

func WithMetadata(name string, text string) DocumentOption {
	return func(d *Document) {
		d.SetMetadata(name, text)
	}
}

// Document is a searchable entity in the corpus.
// It can have multiple independently configured fields that contribute to its
// search ranking.
type Document struct {
	metadata map[string]string
	fields   map[string]*Field
}

func NewDocument(opts ...DocumentOption) *Document {
	d := &Document{}
	d.ensureInitialized()

	for _, opt := range opts {
		opt(d)
	}

	return d
}

func (d *Document) ensureInitialized() {
	if d.metadata == nil {
		d.metadata = make(map[string]string)
	}
	if d.fields == nil {
		d.fields = make(map[string]*Field)
	}
}

// Count returns the number of times a term appears in a field.
func (d *Document) Count(field, term string) int {
	if f := d.fields[field]; f != nil {
		return f.termCounts[term]
	}
	return 0
}

// FieldLen returns the length of a field (in terms).
func (d *Document) FieldLen(field string) int {
	if f := d.fields[field]; f != nil {
		return f.length
	}
	return 0
}

// FieldNames returns the names of all document fields in lexicographic order.
func (d *Document) FieldNames() []string {
	names := slices.Collect(maps.Keys(d.fields))
	slices.Sort(names)
	return names
}

// SetField sets a document field to represent the given tokens.
func (d *Document) SetField(name string, tokens []string) {
	d.ensureInitialized()

	termCounts := make(map[string]int)
	for _, token := range tokens {
		termCounts[token]++
	}

	d.fields[name] = &Field{
		length:     len(tokens),
		termCounts: termCounts,
	}
}

// Metadata gets a metadata entry associated with the document.
// It is the same value previously passed to SetMetadata.
func (d *Document) Metadata(name string) (string, bool) {
	text, ok := d.metadata[name]
	return text, ok
}

// SetMetadata sets data that is not parsed or used by BM25F,
// but it is included in results from BM25F.Score.
func (d *Document) SetMetadata(name string, text string) {
	d.ensureInitialized()
	d.metadata[name] = text
}

func (d *Document) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Metadata map[string]string `json:"metadata"`
		Fields   map[string]*Field `json:"fields"`
	}{
		Metadata: d.metadata,
		Fields:   d.fields,
	})
}

func (d *Document) UnmarshalJSON(data []byte) error {
	state := struct {
		Metadata map[string]string `json:"metadata"`
		Fields   map[string]*Field `json:"fields"`
	}{}
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	d.metadata = state.Metadata
	d.fields = state.Fields
	return nil
}

// Field is a part of a document, such as the title, byline, or body.
//
// Use Document.SetField to set a field value for a document.
type Field struct {
	length     int
	termCounts map[string]int
}

func (f *Field) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Length     int            `json:"length"`
		TermCounts map[string]int `json:"term_counts"`
	}{
		Length:     f.length,
		TermCounts: f.termCounts,
	})
}

func (f *Field) UnmarshalJSON(data []byte) error {
	state := struct {
		Length     int            `json:"length"`
		TermCounts map[string]int `json:"term_counts"`
	}{}
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	f.length = state.Length
	f.termCounts = state.TermCounts
	return nil
}
