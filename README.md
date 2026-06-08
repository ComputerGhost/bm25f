# BM25F

Go implementation of BM25F algorithm.

## Features

* BM25 algorithm base:
    * `Score` calculates how closely a document matches a search query.
    * Free parameter `b` adjusts document length normalization.
    * Free parameter `k1` adjusts the contribution of high-frequency terms.
* BM25F algorithm:
    * Documents are composed of zero or more fields.
    * Documents have a weighted contribution to the search score.
* Free parameters `k1` and `b` are customizable but have sane defaults.
* Metadata can be attached to documents and is returned with search results.
* `Corpus` and `BM25F` can be serialized to and from JSON.

## Limitations

### Tokenizing

This library does not include a way to tokenize documents and queries.

For tokenizing, I recommend starting with UAX #29.
A good implementation of that is [clipperhouse/uax29](https://github.com/clipperhouse/uax29).

### Sorting and pruning non-matches

This library does not include a way to sort results or prune non-matches.

An example of sorting and pruning results is in [examples/search](examples/search/main.go).
However, some applications may require different sorting rules.

## Quick start

```
go get "github.com/computerghost/bm25f"
```

Create the corpus:

```go
corpus := bm25f.NewCorpus()
corpus.Upsert("hello.md", bm25f.NewDocument(
    bm25f.WithField("title", []string{"Hello"}),
    bm25f.WithField("body", []string{"hello", "world"})
    bm25f.WithMetadata("title", "Hello")
))
corpus.Upsert("nature.md", bm25f.NewDocument(
    bm25f.WithField("title", []string{"Nature"}),
    bm25f.WithField("body", []string{"blue", "world"})
    bm25f.WithMetadata("title", "Nature")
))
```

Create the BM25F algorithm:

```go
bm := bm25f.New()
bm.SetWeight("title", 2.0)
bm.SetWeight("body", 1.0)
```

> [!TIP]
> Both the corpus and ranker can be serialized to and from JSON.
> Use this for easy saving and loading.

Now search:

```go
query := []string{"world"}
scores := index.Ranker.Score(corpus, query)

// Remove non-matches.
scores = slices.DeleteFunc(scores, func(r bm25f.Result) bool {
    return r.Score == 0
})

// Sort the remaining results by score then ID
slices.SortFunc(scores, func(a, b bm25f.Result) int {
    if c := cmp.Compare(b.Score, a.Score); c != 0 {
        return c
    }
    return cmp.Compare(a.ID, b.ID)
})

fmt.Println("Results:")
for i, result := range scores {
    title := result.Document.Metadata("title")
    fmt.Printf("  #%d: %s: %s\n", i, result.ID, title)
}
```
