# BM25F

Go implementation of BM25F algorithm.

## Features

* BM25 algorithm base:
  * `Score` calculates how closely a document matches a search query.
  * Free parameter `b` adjusts document length normalization.
  * Free parameter `k1` adjusts the contribution of high frequency terms.
* BM25F algorithm:
  * Documents are composed of zero or more fields.
  * Documents have a weighted contribution to the search score.
* Free parameters `k1` and `b` are customizable but have sane defaults.
* Data can be attached to documents and returned alongside them in results.
* `Rank` implements a default sorting and pruning behavior.

## Limitations

BM25F only scores documents based on how closely they match a query.

### Tokenizing

Tokenizing documents and queries is out of scope.

I like [clipperhouse/uax29](https://github.com/clipperhouse/uax29).
It implements UAX #29 rules and is good enough for most scenarios.

I am working on my own tokenizer for my [Musings](https://github.com/subtributary/musings) product,
but even that is using UAX #29 as a base.

### Sorting and pruning non-matches

The `bm25f.Rank` convenience function implements a default sorting behavior.
It is not customizable.
Custom sorting or pruning must be implemented in code that uses `bm25f`.

## Quick start

```
go get "github.com/computerghost/bm25f"
```

Create the corpus:

```go
corpus := bm25f.Corpus{}

helloDoc := bm25f.Document{}
helloDoc.SetStream("title", []string{"hello"})
helloDoc.SetStream("body", []string{"hello", "world"})
helloDoc.SetAttachment("att_title", "hello")

natureDoc := bm25f.Document{}
natureDoc.SetStream("title", []string{"nature"})
natureDoc.SetStream("body", []string{"blue", "tulip", "blue", "sky", "world"})
natureDoc.SetAttachment("att_title", "nature")
```

> [!TIP]
> The corpus can be serialized to and from JSON for easier saving and loading.

Create the BM25F algorithm:

```go
bm := bm25f.NewBM25F()
bm.SetWeight("title", 2.0)
bm.SetWeight("body", 1.0)
```

Now search:

```go
results := bm.Rank([]string{"world"})
for result := range results {
	id := result.Id
	title := result.Attachments["att_title"]
	fmt.Printf("%s -- %s", id, title)
}
```

