package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/idiomat/dodtnyt/e3/embed"
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

	if err := db.AutoMigrate(&embed.Book{}, &embed.BookEmbedding{}); err != nil {
		log.Fatalln("failed to migrate", err)
	}

	if err := db.Exec("CREATE INDEX ON book_embeddings USING hnsw (embedding vector_l2_ops)").Error; err != nil {
		log.Fatalln("failed to create index", err)
	}
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

	var b embed.Book
	if err := db.FirstOrCreate(&b, embed.Book{Title: "Meditations", Author: "Marcus Aurelius"}).Error; err != nil {
		log.Fatalln("failed to create book", err)
	}

	chunker, err := embed.NewChunker(
		embed.DefaultChunkSize,
		embed.DefaultChunkOverlap,
	)
	if err != nil {
		log.Fatalln(err)
	}

	httpClient := http.Client{Timeout: 30 * time.Second}

	var endpoint *url.URL
	if endpoint, err = url.Parse("http://localhost:11434/api/embeddings"); err != nil {
		log.Fatalln(err)
	}

	eg, err := embed.NewGenerator(&httpClient, endpoint, embed.DefaultModel)
	if err != nil {
		log.Fatalln(err)
	}

	chunks, err := chunker.Chunk(ctx, f)
	if err != nil {
		log.Fatalln(err)
	}

	for _, chunk := range chunks {
		vals, err := eg.Generate(ctx, chunk)
		if err != nil {
			log.Fatalln(err)
		}

		be := embed.BookEmbedding{
			BookID:    b.ID,
			Text:      chunk,
			Embedding: pgvector.NewVector(vals),
		}
		if err := db.Save(&be).Error; err != nil {
			log.Fatalln("failed to save book embedding", err)
		}
	}

	if err := db.Save(&b).Error; err != nil {
		log.Fatalln("failed to save book", err)
	}

	str := "How short lived the praiser and the praised, the one who remembers and the remembered."

	strEmbedding, err := eg.Generate(ctx, str)
	if err != nil {
		log.Fatalln(err)
	}

	var bookEmbeddings []embed.BookEmbedding
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
