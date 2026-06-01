package bm25f

type Corpus struct {
	Documents map[string]Document `json:"documents"`

	// DocsWithTerm are the number of documents containing each term.
	DocsWithTerm map[string]int `json:"docs_with_term"`

	// TotalLengths are the total lengths of each field across all documents.
	TotalLengths map[string]int `json:"total_lengths"`
}

// Remove removes all data associated with a document.
func (c *Corpus) Remove(id string) {
	doc, ok := c.Documents[id]
	if !ok {
		return
	}

	delete(c.Documents, id)

	for f, s := range doc.Streams {
		c.TotalLengths[f] -= s.Length
	}

	for _, s := range doc.Streams {
		for term := range s.TermCounts {
			c.DocsWithTerm[term]--
			if c.DocsWithTerm[term] == 0 {
				delete(c.DocsWithTerm, term)
			}
		}
	}
}

// Upsert processes and adds a document into the corpus.
func (c *Corpus) Upsert(id string, document Document) {
	if c.Documents == nil {
		c.Documents = make(map[string]Document)
		c.DocsWithTerm = make(map[string]int)
		c.TotalLengths = make(map[string]int)
	}

	c.Remove(id)

	c.Documents[id] = document

	for f, s := range document.Streams {
		c.TotalLengths[f] += s.Length
	}

	for _, s := range document.Streams {
		for term := range s.TermCounts {
			c.DocsWithTerm[term]++
		}
	}
}

// Document is a searchable entity in the corpus.
// It can have multiple independently-configured streams that contribute to its
// search ranking.
type Document struct {
	Attachments map[string]string  `json:"attachments"`
	Streams     map[string]*stream `json:"streams"`
}

// SetAttachment sets data that is not parsed or used by BM25F.
func (d *Document) SetAttachment(id string, text string) {
	if d.Attachments == nil {
		d.Attachments = make(map[string]string)
	}

	d.Attachments[id] = text
}

// SetStream sets a document stream to represent the given tokens.
func (d *Document) SetStream(field string, tokens []string) {
	if d.Streams == nil {
		d.Streams = make(map[string]*stream)
	}

	termCounts := make(map[string]int)
	for _, token := range tokens {
		termCounts[token]++
	}

	d.Streams[field] = &stream{
		Length:     len(tokens),
		TermCounts: termCounts,
	}
}

// stream represents the tokens in one of a document's streams.
// A document can have multiple streams, each referenced by its field name.
//
// These should be created via Document.SetStream.
type stream struct {
	Length     int            `json:"length"`
	TermCounts map[string]int `json:"term_counts"`
}
