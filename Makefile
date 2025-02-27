all: mock

# ---------------------------
MOCK_OUT=pkg/mock/mock.go
MOCK_SRC=./pkg/interfaces
MOCK_INTERFACES=SlackService SlackThreadService LLMSession PolicyClient Repository Action UseCase LLMClient EmbeddingClient GitHubAppClient

mock: $(MOCK_OUT)

$(MOCK_OUT): $(MOCK_SRC)/*
	go run github.com/matryer/moq@v0.5.3 -pkg mock -out $(MOCK_OUT) $(MOCK_SRC) $(MOCK_INTERFACES)

clean:
	rm -f $(MOCK_OUT)
