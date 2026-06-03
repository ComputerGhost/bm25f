package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/computerghost/bm25f"
)

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = printUsage
	indexFile := flags.String("i", "index.json", "")
	_ = flags.Parse(os.Args[3:])

	switch os.Args[1] {
	case "index":
		if err := indexMain(os.Args[2], *indexFile); err != nil {
			log.Printf("Error indexing dir: %v", err)
		}
	case "query":
		if err := queryMain(os.Args[2], *indexFile); err != nil {
			log.Printf("Error querying index: %v", err)
		}
	}
}

func printUsage() {
	program := filepath.Base(os.Args[0])
	fmt.Println("Usage:")
	fmt.Printf("  %s index <directory> [OPTIONS]\n", program)
	fmt.Printf("  %s query <search-query> [OPTIONS]\n", program)
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -i    index file location [default: index.json]")
}

type Index struct {
	Corpus *bm25f.Corpus `json:"corpus"`
	Ranker *bm25f.BM25F  `json:"ranker"`
}

func indexMain(dir, indexFile string) error {
	index := Index{
		Corpus: bm25f.NewCorpus(),
		Ranker: bm25f.New(),
	}
	_ = index.Ranker.SetWeight("content", 1.0)
	_ = index.Ranker.SetWeight("path", 2.0)

	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() || filepath.Ext(d.Name()) != ".go" {
			return nil
		}

		content, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			return fmt.Errorf("read file: %v", err)
		}
		contentTokens := strings.Fields(string(content))

		pathTokens := strings.FieldsFunc(path, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})

		index.Corpus.Upsert(path, bm25f.NewDocument(
			bm25f.WithField("content", contentTokens),
			bm25f.WithField("path", pathTokens),
		))

		return nil
	}); err != nil {
		return fmt.Errorf("walk dir: %v", err)
	}

	data, err := json.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshal index: %v", err)
	}

	if err := os.WriteFile(indexFile, data, 0644); err != nil {
		return fmt.Errorf("write index file: %v", err)
	}

	return nil
}

func queryMain(query, indexFile string) error {
	data, err := os.ReadFile(indexFile)
	if err != nil {
		return fmt.Errorf("read index file: %v", err)
	}

	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("unmarshal index: %v", err)
	}

	queryTokens := strings.Fields(query)
	scores := index.Ranker.Score(index.Corpus, queryTokens)

	scores = slices.DeleteFunc(scores, func(r bm25f.Result) bool {
		return r.Score == 0
	})

	slices.SortFunc(scores, func(a, b bm25f.Result) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})

	fmt.Println("Results:")
	for i, result := range scores {
		fmt.Printf("  #%d: %s\n", i, result.ID)
	}

	return nil
}
