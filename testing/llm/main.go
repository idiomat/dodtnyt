package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var bookPath string
var db *gorm.DB

func init() {
	flag.StringVar(&bookPath, "book", "", "path to the book ")
	flag.Parse()

	var err error
	var dsn = "postgres://postgres:password@localhost:5432/test?sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalln("failed to connect database", err)
	}

	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		log.Fatalln("failed to create extension", err)
	}

	if err := db.AutoMigrate(&book{}, &bookEmbedding{}); err != nil {
		log.Fatalln("failed to migrate", err)
	}

	if err := db.Exec("CREATE INDEX ON book_embeddings USING hnsw (embedding vector_l2_ops)").Error; err != nil {
		log.Fatalln("failed to create index", err)
	}
}

type book struct {
	gorm.Model
	Title      string
	Author     string
	Embeddings []bookEmbedding
}

type bookEmbedding struct {
	gorm.Model
	BookID    uint
	Text      string
	Embedding pgvector.Vector `gorm:"type:vector(384)"`
}

func main() {
	ctx := context.Background()

	log.Println("Start")

	if bookPath == "" {
		log.Fatalln("book path is required")
	}

	f, err := os.Open(bookPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var text []string
	for scanner.Scan() {
		text = append(text, scanner.Text())
	}

	const chunkSize = 512
	const chunkOverlap = 128
	var chunks []string                     // store the final chunks of text
	var currentChunkBuilder strings.Builder // helps efficiently build the current chunk of text
	var currentChunkWords int               // keeps track of the number of words in the current chunk

	for _, line := range text {
		words := strings.Fields(line) // split the line into words
		for _, word := range words {
			if currentChunkWords > 0 {
				currentChunkBuilder.WriteString(" ") // add a space before adding the next word
			}
			currentChunkBuilder.WriteString(word) // add the word to the current chunk
			currentChunkWords++                   // increment the number of words in the current chunk

			// build the full chunk
			if currentChunkWords >= chunkSize {
				chunks = append(chunks, currentChunkBuilder.String())
				overlapWords := strings.Fields(currentChunkBuilder.String())
				currentChunkBuilder.Reset()
				currentChunkWords = 0
				for i := len(overlapWords) - chunkOverlap; i < len(overlapWords); i++ {
					if currentChunkWords > 0 {
						currentChunkBuilder.WriteString(" ")
					}
					currentChunkBuilder.WriteString(overlapWords[i])
					currentChunkWords++
				}
			}
		}
	}

	// add the last chunk
	if currentChunkWords > 0 {
		chunks = append(chunks, currentChunkBuilder.String())
	}

	var b book
	if err := db.FirstOrCreate(&b, book{Title: "Meditations", Author: "Marcus Aurelius"}).Error; err != nil {
		log.Fatalln("failed to create book", err)
	}

	httpClient := http.Client{Timeout: 30 * time.Second}

	type embeddingRequest struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}

	type embeddingResponse struct {
		Embedding []float32 `json:"embedding"`
	}

	endpoint := "http://localhost:11434/api/embeddings"

	for _, chunk := range chunks {
		bs, err := json.Marshal(embeddingRequest{
			Model:  "all-minilm",
			Prompt: chunk,
		})
		if err != nil {
			log.Fatalln(err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bs))
		if err != nil {
			log.Fatalln(err)
		}

		req.Header.Set("Content-Type", "application/json")
		res, err := httpClient.Do(req)
		if err != nil {
			log.Fatalln(err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			log.Fatalf("unexpected status code: %s", res.Status)
		}

		var response embeddingResponse
		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			log.Fatalln(err)
		}

		be := bookEmbedding{
			BookID:    b.ID,
			Text:      chunk,
			Embedding: pgvector.NewVector(response.Embedding),
		}
		if err := db.Save(&be).Error; err != nil {
			log.Fatalln("failed to save book embedding", err)
		}
	}

	if err := db.Save(&b).Error; err != nil {
		log.Fatalln("failed to save book", err)
	}

	var strEmbedding []float32

	{
		str := "How short lived the praiser and the praised, the one who remembers and the remembered."
		bs, err := json.Marshal(embeddingRequest{
			Model:  "all-minilm",
			Prompt: str,
		})
		if err != nil {
			log.Fatalln(err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bs))
		if err != nil {
			log.Fatalln(err)
		}

		req.Header.Set("Content-Type", "application/json")
		res, err := httpClient.Do(req)
		if err != nil {
			log.Fatalln(err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			log.Fatalf("unexpected status code: %s", res.Status)
		}

		var response embeddingResponse
		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			log.Fatalln(err)
		}

		strEmbedding = response.Embedding
	}

	var bookEmbeddings []bookEmbedding
	db.Clauses(
		clause.OrderBy{
			Expression: clause.Expr{
				SQL: "embedding <-> ?",
				Vars: []interface{}{
					pgvector.NewVector(strEmbedding),
				},
			},
		},
	).Limit(5).Find(&bookEmbeddings)

	for _, be := range bookEmbeddings {
		log.Println(be.Text)
	}

	log.Println("Done")
}
