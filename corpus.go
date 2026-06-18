package bm25f

import (
	"encoding/json"
	"maps"
	"slices"
)

type Corpus struct {
	documents map[string]*Document

	// docsWithTerm are the number of documents containing each term.
	docsWithTerm map[string]int

	// totalLengths are the total lengths of each field across all documents.
	totalLengths map[string]int
}

// NewCorpus creates an empty Corpus.
func NewCorpus() *Corpus {
	c := &Corpus{}
	c.ensureInitialized()
	return c
}

func (c *Corpus) ensureInitialized() {
	if c.documents == nil {
		c.documents = make(map[string]*Document)
	}
	if c.docsWithTerm == nil {
		c.docsWithTerm = make(map[string]int)
	}
	if c.totalLengths == nil {
		c.totalLengths = make(map[string]int)
	}
}

// Documents returns a map from document id to Document.
// The returned map should be considered immutable.
func (c *Corpus) Documents() map[string]*Document {
	c.ensureInitialized()
	return c.documents
}

// Len returns the number of documents in the corpus.
func (c *Corpus) Len() int {
	return len(c.documents)
}

func (c *Corpus) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Documents map[string]*Document `json:"documents"`
	}{
		Documents: c.documents,
	})
}

func (c *Corpus) UnmarshalJSON(data []byte) error {
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

// Remove removes all data associated with a document.
func (c *Corpus) Remove(id string) {
	c.ensureInitialized()

	doc, ok := c.documents[id]
	if !ok {
		return
	}

	delete(c.documents, id)
	c.removeStats(doc)
}

// Upsert processes and adds a document into the corpus.
// The document must not be changed after passing it to this function.
func (c *Corpus) Upsert(id string, document *Document) {
	c.ensureInitialized()

	if old, ok := c.documents[id]; ok {
		c.removeStats(old)
	}

	c.documents[id] = document
	c.addStats(document)
}

func (c *Corpus) addStats(doc *Document) {
	for name, field := range doc.fields {
		c.totalLengths[name] += field.length

		for term := range field.termCounts {
			c.docsWithTerm[term]++
		}
	}
}

func (c *Corpus) clone() *Corpus {
	return &Corpus{
		documents:    maps.Clone(c.documents),
		docsWithTerm: maps.Clone(c.docsWithTerm),
		totalLengths: maps.Clone(c.totalLengths),
	}
}

func (c *Corpus) removeStats(doc *Document) {
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
