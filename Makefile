all: mock

# ---------------------------
MOCK_OUT=pkg/mock/mock.go
MOCK_SRC=./pkg/interfaces
MOCK_INTERFACES=SlackService GenAIChatSession PolicyClient Repository Action

mock: $(MOCK_OUT)

$(MOCK_OUT): $(MOCK_SRC)/*
	go run github.com/matryer/moq@v0.5.1 -pkg mock -out $(MOCK_OUT) $(MOCK_SRC) $(MOCK_INTERFACES)

clean:
	rm -f $(MOCK_OUT)
