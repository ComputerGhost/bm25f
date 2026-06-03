package bm25f

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
)

type fieldConfig struct {
	B      float64 `json:"b"`
	Weight float64 `json:"weight"`
}

type BM25F struct {
	k1     float64
	fields map[string]*fieldConfig
}

func New() *BM25F {
	return &BM25F{
		k1:     1.2,
		fields: make(map[string]*fieldConfig),
	}
}

// SetK1 sets the `k1` parameter of the BM25F algorithm.
// It controls the impact of frequent terms on the scores.
// With lower values, frequent terms affect the score less.
// With higher values, frequent terms affect the score more.
// For most corpora, a value between 1.2 and 2 is good.
// The default is 1.2.
//
// An error is returned if the value is less than or equal to 0.
func (bm *BM25F) SetK1(k1 float64) error {
	if k1 <= 0 {
		return fmt.Errorf("out of range: %f", k1)
	}

	bm.k1 = k1
	return nil
}

// SetB sets the `b` parameter of the BM25F algorithm.
// It controls the strength of field length normalizations.
// With a value of 0, field lengths are not taken into consideration.
// With a value of 1, field lengths are fully normalized.
// For most corpora, a value between 0.5 and 0.8 is good.
// The default is 0.72.
//
// An error is returned if the value is less than 0 or greater than 1.
func (bm *BM25F) SetB(field string, b float64) error {
	if b < 0 || b > 1 {
		return fmt.Errorf("out of range: %f", b)
	}

	if fc, ok := bm.fields[field]; ok {
		fc.B = b
	} else {
		bm.fields[field] = &fieldConfig{
			B:      b,
			Weight: 1.0,
		}
	}

	return nil
}

// SetWeight sets the relative weight of the field.
// The field with the bulk of the content should be 1.0.
// The default is 1.0
//
// An error is returned if the value is less than 0.
func (bm *BM25F) SetWeight(field string, weight float64) error {
	if weight < 0 {
		return fmt.Errorf("out of range: %f", weight)
	}

	if fc, ok := bm.fields[field]; ok {
		fc.Weight = weight
	} else {
		bm.fields[field] = &fieldConfig{
			B:      0.72,
			Weight: weight,
		}
	}

	return nil
}

func (bm *BM25F) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		K1     float64                 `json:"k1"`
		Fields map[string]*fieldConfig `json:"fields"`
	}{
		K1:     bm.k1,
		Fields: bm.fields,
	})
}

func (bm *BM25F) UnmarshalJSON(data []byte) error {
	state := struct {
		K1     float64                 `json:"k1"`
		Fields map[string]*fieldConfig `json:"fields"`
	}{}
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	bm.k1 = state.K1
	bm.fields = state.Fields
	return nil
}

type Result struct {
	ID string

	// Score indicates how well the document matches the query.
	//
	// A value of 0 indicates no match.
	// Other values are meaningless except in comparison to other results:
	// a higher value indicates a better match.
	Score float64

	document *Document
}

// Metadata returns the metadata associated with the document in the result.
func (r *Result) Metadata(name string) (string, bool) {
	return r.document.Metadata(name)
}

// Score calculates how well each document matches the query.
// The results include every document and are unsorted—to remove non-matches
// and sort the results, use Rank or do it yourself.
func (bm *BM25F) Score(corpus *Corpus, query []string) []Result {
	// Deduplicate query
	query = slices.Clone(query)
	slices.Sort(query)
	query = slices.Compact(query)

	// Init the results with document data and 0 scores.
	results := make([]Result, 0, len(corpus.documents))
	for id, doc := range corpus.documents {
		results = append(results, Result{
			ID:       id,
			document: doc,
		})
	}

	// Cache avg field lengths instead of recalculating when needed.
	avgFieldLengths := make(map[string]float64)
	for field := range bm.fields {
		if docCount := len(corpus.documents); docCount != 0 {
			totalLength := float64(corpus.totalLengths[field])
			avgFieldLengths[field] = totalLength / float64(docCount)
		}
	}

	// A term's score for a document is its overall importance (idf) times its
	// saturation within the document. These scores are summed per document for
	// the final document scores.
	for _, term := range query {
		if corpus.docsWithTerm[term] == 0 {
			continue
		}

		idf := bm.idf(corpus, term)
		for i := range results {
			result := &results[i]
			termFreq := bm.termFrequency(result.document, term, avgFieldLengths)
			if termFreq == 0 {
				continue
			}

			saturation := termFreq * (bm.k1 + 1) / (termFreq + bm.k1)
			result.Score += idf * saturation
		}
	}

	return results
}

// idf returns the relative importance of a word based on its rarity.
func (bm *BM25F) idf(c *Corpus, term string) float64 {
	// For the IDF, we apply a modified Robertson/Sparck Jones formula across
	// all fields. There are rare scenarios where this does not yield good
	// results. We will ignore the problem until it shows itself in practice.
	docCount := float64(len(c.documents))
	docFreq := float64(c.docsWithTerm[term])
	return math.Log((docCount-docFreq+0.5)/(docFreq+0.5) + 1)
}

// termFrequency returns the normalized weighted frequency of a term within the
// document across all fields.
func (bm *BM25F) termFrequency(
	doc *Document, term string, avgFieldLengths map[string]float64,
) (result float64) {
	for field, config := range bm.fields {
		if config.Weight == 0 {
			continue
		}

		avgFieldLen := avgFieldLengths[field]
		if avgFieldLen == 0.0 {
			continue
		}

		// Normalize results when the field length is far from average.
		fieldLen := float64(doc.FieldLen(field))
		lengthNorm := 1 - config.B + config.B*fieldLen/avgFieldLen

		// Simple weighted summation with normalization.
		termFreq := float64(doc.Count(field, term))
		result += config.Weight * termFreq / lengthNorm
	}
	return
}
