package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const DefaultModel = "all-minilm"

type Generator struct {
	model        string
	httpClient   *http.Client
	httpEndpoint *url.URL
}

func (g *Generator) validate() error {
	if g.httpClient == nil {
		return errors.New("httpClient is required")
	}
	if g.httpEndpoint == nil {
		return errors.New("httpEndpoint is required")
	}
	if g.model == "" {
		return errors.New("model is required")
	}
	return nil
}

func NewGenerator(httpClient *http.Client, endpoint *url.URL, model string) (*Generator, error) {
	g := &Generator{
		httpClient:   httpClient,
		httpEndpoint: endpoint,
		model:        model,
	}
	if g.model == "" {
		g.model = DefaultModel
	}
	return g, g.validate()
}

func (g *Generator) Generate(ctx context.Context, text string) ([]float32, error) {
	bs, err := json.Marshal(EndpointRequest{
		Model:  g.model,
		Prompt: text,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		g.httpEndpoint.String(),
		bytes.NewReader(bs),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %s", res.Status)
	}

	var response EndpointResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Embedding, nil
}

type EndpointRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EndpointResponse struct {
	Embedding []float32 `json:"embedding"`
}
