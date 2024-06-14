package embed

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
)

const DefaultChunkSize = 512
const DefaultChunkOverlap = 128

type Chunker struct {
	chunkSize    int
	chunkOverlap int
}

func (c *Chunker) validate() error {
	if c.chunkSize == 0 {
		return errors.New("chunkSize is required")
	}
	if c.chunkOverlap == 0 {
		return errors.New("chunkOverlap is required")
	}
	return nil
}

func NewChunker(chunkSize, chunkOverlap int) (*Chunker, error) {
	c := &Chunker{chunkSize: chunkSize, chunkOverlap: chunkOverlap}
	if c.chunkSize == 0 {
		c.chunkSize = DefaultChunkSize
	}
	if c.chunkOverlap == 0 {
		c.chunkOverlap = DefaultChunkOverlap
	}
	return c, c.validate()
}

func (c *Chunker) Chunk(ctx context.Context, rdr io.Reader) ([]string, error) {
	var chunks []string                     // store the final chunks of text
	var currentChunkBuilder strings.Builder // helps efficiently build the current chunk of text
	var currentChunkWords int               // keeps track of the number of words in the current chunk

	scanner := bufio.NewScanner(rdr)
	var text []string
	for scanner.Scan() {
		text = append(text, scanner.Text())
	}

	for _, line := range text {
		words := strings.Fields(line) // split the line into words
		for _, word := range words {
			if currentChunkWords > 0 {
				currentChunkBuilder.WriteString(" ") // add a space before adding the next word
			}
			currentChunkBuilder.WriteString(word) // add the word to the current chunk
			currentChunkWords++                   // increment the number of words in the current chunk

			// build the full chunk
			if currentChunkWords >= c.chunkSize {
				chunks = append(chunks, currentChunkBuilder.String())
				overlapWords := strings.Fields(currentChunkBuilder.String())
				currentChunkBuilder.Reset()
				currentChunkWords = 0
				for i := len(overlapWords) - c.chunkOverlap; i < len(overlapWords); i++ {
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

	return chunks, nil
}
