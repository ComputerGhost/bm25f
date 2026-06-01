package bm25f

import (
	"cmp"
	"fmt"
	"math"
	"slices"
)

type fieldConfig struct {
	b      float64
	weight float64
}

type BM25F struct {
	k1            float64
	fieldsConfigs map[string]*fieldConfig
}

// NewBM25F creates a new BM25F with sane defaults.
func NewBM25F() *BM25F {
	return &BM25F{k1: 1.2}
}

// SetB sets the `b` parameter of the algorithm.
// It controls the strength of stream length normalizations.
//
// Values can range from 0 to 1.
// With 0, stream lengths are not considered.
// With 1, stream lengths are fully normalized.
// For most corpora, a value between 0.5 and 0.8 is good.
// The default is 0.72.
func (bm *BM25F) SetB(field string, b float64) error {
	if b < 0 || b > 1 {
		return fmt.Errorf("out of range: %f", b)
	}
	bm.ensureFieldConfig(field).b = b
	return nil
}

// SetK1 sets the `k1` free parameter of the algorithm.
// It controls how much frequent terms affect scores.
//
// Values must be greater than 0.
// With a low value, frequent terms affect scores less.
// With a high value, frequent terms affect scores more.
// For most corpora, a value between 1.2 and 2 is good.
// If NewBM25F was used to create the BM25F, then the default is 1.2.
func (bm *BM25F) SetK1(k1 float64) error {
	if k1 < 0 {
		return fmt.Errorf("out of range: %f", k1)
	}
	bm.k1 = k1
	return nil
}

// SetWeight sets the relative weight of the field.
//
// The default is 0, so this must be called to consider a field.
func (bm *BM25F) SetWeight(field string, weight float64) {
	bm.ensureFieldConfig(field).weight = weight
}

func (bm *BM25F) ensureFieldConfig(name string) *fieldConfig {
	if bm.fieldsConfigs == nil {
		bm.fieldsConfigs = map[string]*fieldConfig{}
	}

	fc, ok := bm.fieldsConfigs[name]
	if !ok {
		fc = &fieldConfig{b: 0.72}
		bm.fieldsConfigs[name] = fc
	}

	return fc
}

type Result struct {
	Id       string
	Document Document

	// Score is how well the document matches the query.
	// A higher value indicates a better match than a lower value.
	// A value of 0 indicates no match.
	Score float64
}

// Rank returns document results sorted by how well they match the query.
// The best match is first. Equal matches are sorted lexigraphically by id.
// Documents that do not match the query are excluded.
func (bm *BM25F) Rank(corpus Corpus, query []string) []Result {
	results := bm.Score(corpus, query)

	// Remove results that are not a match.
	results = slices.DeleteFunc(results, func(result Result) bool {
		return result.Score == 0.0
	})

	// Sort the results descending by score.
	slices.SortFunc(results, func(a, b Result) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		return cmp.Compare(a.Id, b.Id)
	})

	return results
}

// Score calculates how well each document matches the query.
// The results include every document and are unsorted—to remove non-matches
// and sort the results, use Rank or do it yourself.
func (bm *BM25F) Score(corpus Corpus, query []string) []Result {
	// Deduplicate query
	slices.Sort(query)
	query = slices.Compact(query)

	// Init the results with document data and 0 scores.
	results := make([]Result, 0, len(corpus.Documents))
	for id, doc := range corpus.Documents {
		results = append(results, Result{
			Id:       id,
			Document: doc,
			Score:    0.0,
		})
	}

	// A term's score for a document is its overall importance (idf) times its
	// saturation within the document. These scores are summed per document for
	// the final document scores.
	for _, term := range query {
		idf := bm.idf(corpus, term)
		for i := range results {
			result := &results[i]
			termFreq := bm.termFrequency(corpus, result.Document, term)
			saturation := termFreq / (bm.k1 * idf)
			result.Score += saturation * idf
		}
	}

	return results
}

// idf returns the relative importance of a word based on its rarity.
func (bm *BM25F) idf(c Corpus, term string) float64 {
	// For the IDF, we apply a modified Robertson/Sparck Jones formula across
	// all streams. There are rare scenarios where this does not yield good
	// results. We will ignore the problem until it shows itself in practice.
	docCount := float64(len(c.Documents))
	docFreq := float64(c.DocsWithTerm[term])
	return math.Log((docCount-docFreq+0.5)/(docFreq+0.5) + 1)
}

// termFrequency returns the normalized weighted frequency of a term within the
// document across all streams.
func (bm *BM25F) termFrequency(c Corpus, doc Document, term string) (result float64) {
	for field, config := range bm.fieldsConfigs {
		avgStreamLen := bm.avgStreamLength(c, field)
		if avgStreamLen == 0.0 {
			continue
		}

		s := doc.Streams[field]

		// Normalize results when the stream length is far from average.
		streamLen := float64(s.Length)
		lengthNorm := 1 - config.b + config.b*streamLen/avgStreamLen

		// Simple weighted summation with normalization.
		termFreq := float64(s.TermCounts[term])
		result += config.weight * termFreq / lengthNorm
	}
	return
}

func (bm *BM25F) avgStreamLength(c Corpus, field string) float64 {
	if docCount := len(c.Documents); docCount > 0 {
		return float64(c.TotalLengths[field]) / float64(docCount)
	}
	return 0
}
