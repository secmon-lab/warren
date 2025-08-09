package slack

import "time"

// SearchOptions represents the options for Slack message search
type SearchOptions struct {
	Query     string `json:"query"`
	Sort      string `json:"sort,omitempty"`     // score, timestamp
	SortDir   string `json:"sort_dir,omitempty"` // asc, desc
	Count     int    `json:"count,omitempty"`
	Page      int    `json:"page,omitempty"`
	Highlight bool   `json:"highlight,omitempty"`
}

// SearchResponse represents the response from Slack search API
type SearchResponse struct {
	OK       bool          `json:"ok"`
	Query    string        `json:"query"`
	Messages MessagesBlock `json:"messages"`
	Error    string        `json:"error,omitempty"`
}

// MessagesBlock represents the messages block in search response
type MessagesBlock struct {
	Total      int       `json:"total"`
	Pagination Paging    `json:"paging"`
	Matches    []Message `json:"matches"`
}

// Paging represents pagination information
type Paging struct {
	Count int `json:"count"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Pages int `json:"pages"`
}

// Message represents a single message in search results
type Message struct {
	Type        string       `json:"type"`
	Channel     ChannelInfo  `json:"channel"`
	User        string       `json:"user"`
	Username    string       `json:"username"`
	Text        string       `json:"text"`
	Timestamp   string       `json:"ts"`
	Permalink   string       `json:"permalink"`
	Team        string       `json:"team,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// ChannelInfo represents channel information
type ChannelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Attachment represents message attachments
type Attachment struct {
	ID      int    `json:"id"`
	Pretext string `json:"pretext,omitempty"`
	Text    string `json:"text,omitempty"`
}

// SearchOutput represents the tool output format
type SearchOutput struct {
	Total    int                 `json:"total"`
	Messages []SearchMessageItem `json:"messages"`
}

// SearchMessageItem represents a formatted message item for output
type SearchMessageItem struct {
	Channel       string    `json:"channel"`
	ChannelName   string    `json:"channel_name"`
	User          string    `json:"user"`
	UserName      string    `json:"user_name"`
	Text          string    `json:"text"`
	Timestamp     string    `json:"timestamp"`
	Permalink     string    `json:"permalink"`
	FormattedTime time.Time `json:"formatted_time,omitempty"`
}