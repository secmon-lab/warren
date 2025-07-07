package interfaces

import "github.com/m-mizutani/gollem"

type Clients struct {
	repo    Repository
	llm     gollem.LLMClient
	storage StorageClient
}

type Option func(*Clients)

func WithLLMClient(llm gollem.LLMClient) Option {
	return func(c *Clients) {
		c.llm = llm
	}
}

func WithStorageClient(storage StorageClient) Option {
	return func(c *Clients) {
		c.storage = storage
	}
}

func WithRepository(repo Repository) Option {
	return func(c *Clients) {
		c.repo = repo
	}
}

func NewClients(opts ...Option) *Clients {
	c := &Clients{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Clients) LLM() gollem.LLMClient {
	return c.llm
}

func (c *Clients) Storage() StorageClient {
	return c.storage
}

func (c *Clients) Repository() Repository {
	return c.repo
}
