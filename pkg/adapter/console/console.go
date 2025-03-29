package console

import (
	"context"
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

type Client struct{}

func New() *Client {
	return &Client{}
}

func (c *Client) PostMessageContext(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
	ts := fmt.Sprintf("%d", time.Now().UnixNano())
	return channelID, ts, nil
}

func (c *Client) UpdateMessageContext(ctx context.Context, channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error) {
	ts := fmt.Sprintf("%d", time.Now().UnixNano())
	return channelID, ts, ts, nil
}

func (c *Client) AuthTest() (*slack.AuthTestResponse, error) {
	return &slack.AuthTestResponse{
		Team:         "console",
		User:         "console_user",
		TeamID:       "T00000000",
		UserID:       "U00000000",
		BotID:        "B00000000",
		EnterpriseID: "",
	}, nil
}

func (c *Client) OpenView(triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error) {
	return &slack.ViewResponse{}, nil
}

func (c *Client) UploadFileV2Context(ctx context.Context, params slack.UploadFileV2Parameters) (*slack.FileSummary, error) {
	return &slack.FileSummary{
		ID: "F00000000",
	}, nil
}
