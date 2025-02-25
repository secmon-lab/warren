// #nosec:G104
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
)

type Console struct {
	channelID string
	threadID  string
	writer    io.Writer
}

func (c *Console) TrimMention(message string) string {
	return message
}

var _ interfaces.SlackService = &Console{}

func (c *Console) NewThread(thread model.SlackThread) interfaces.SlackThreadService {
	return &ConsoleThread{
		channelID: thread.ChannelID,
		threadID:  thread.ThreadID,
		writer:    c.writer,
	}
}

func (c *Console) VerifyRequest(_ http.Header, _ []byte) error {
	return nil
}

func (c *Console) ShowCloseAlertModal(_ context.Context, _ model.Alert, _ string) error {
	return nil
}

type ConsoleThread struct {
	channelID string
	threadID  string
	writer    io.Writer
}

var _ interfaces.SlackThreadService = &ConsoleThread{}

func NewConsole(w io.Writer) *Console {
	return &Console{
		channelID: "console",
		threadID:  fmt.Sprintf("thread-%d", time.Now().Unix()),
		writer:    w,
	}
}

func NewConsoleWithWriter(w io.Writer) *Console {
	return &Console{
		channelID: "console",
		threadID:  fmt.Sprintf("thread-%d", time.Now().Unix()),
		writer:    w,
	}
}

func (c *Console) PostAlert(_ context.Context, alert model.Alert) (interfaces.SlackThreadService, error) {
	thread := &ConsoleThread{
		channelID: c.channelID,
		threadID:  c.threadID,
		writer:    c.writer,
	}

	thread.printHeader("🚨 New Alert")

	// Title in bold yellow
	color.New(color.FgYellow, color.Bold).Fprintf(c.writer, "Title: %s\n", alert.Title)
	fmt.Fprintf(c.writer, "Schema: %s\n", alert.Schema)

	if len(alert.Attributes) > 0 {
		fmt.Fprintln(c.writer, "\n📋 Attributes:")
		for _, attr := range alert.Attributes {
			if attr.Link != "" {
				color.New(color.FgCyan).Fprintf(c.writer, "• %s: %s (%s)\n", attr.Key, attr.Value, attr.Link)
			} else {
				color.New(color.FgCyan).Fprintf(c.writer, "• %s: %s\n", attr.Key, attr.Value)
			}
		}
	}

	if alert.Data != nil {
		fmt.Fprintln(c.writer, "\n🔍 Data:")
		data, err := json.MarshalIndent(alert.Data, "  ", "  ")
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal alert data")
		}
		color.New(color.FgBlue).Fprintf(c.writer, "%s\n", string(data))
	}

	fmt.Fprintln(c.writer, strings.Repeat("-", 80))

	return thread, nil
}

func (c *Console) ChannelID() string {
	return c.channelID
}

func (c *Console) ThreadID() string {
	return c.threadID
}

func (c *ConsoleThread) ChannelID() string {
	return c.channelID
}

func (c *ConsoleThread) ThreadID() string {
	return c.threadID
}

func (c *ConsoleThread) UpdateAlert(_ context.Context, alert model.Alert) error {
	c.printHeader("📝 Alert Update")

	color.New(color.FgYellow, color.Bold).Fprintf(c.writer, "Title: %s\n", alert.Title)

	if alert.Finding != nil {
		fmt.Fprintln(c.writer, "\n📊 Finding:")
		c.printFinding(*alert.Finding)
	}

	fmt.Fprintln(c.writer, strings.Repeat("-", 80))
	return nil
}

func (c *ConsoleThread) PostNextAction(_ context.Context, result prompt.ActionPromptResult) error {
	c.printHeader("⚡ Next Action")

	color.New(color.FgGreen).Fprintf(c.writer, "Action: %s\n", result.Action)

	if len(result.Args) > 0 {
		fmt.Fprintln(c.writer, "\n📋 Arguments:")
		args, err := json.MarshalIndent(result.Args, "  ", "  ")
		if err != nil {
			return goerr.Wrap(err, "failed to marshal action args")
		}
		color.New(color.FgBlue).Fprintf(c.writer, "%s\n", string(args))
	}

	fmt.Fprintln(c.writer, strings.Repeat("-", 80))
	return nil
}

func (c *ConsoleThread) AttachFile(_ context.Context, comment, filename string, content []byte) error {
	c.printHeader("📎 File Attachment")

	fmt.Fprintf(c.writer, "Comment: %s\n", comment)
	color.New(color.FgCyan).Fprintf(c.writer, "Filename: %s\n", filename)
	if len(content) > 10000 {
		content = append(content[:10000], []byte("...(truncated)")...)
	}
	fmt.Fprintf(c.writer, "\nContent:\n%s\n", string(content))

	fmt.Fprintln(c.writer, strings.Repeat("-", 80))
	return nil
}

func (c *ConsoleThread) PostFinding(_ context.Context, finding model.AlertFinding) error {
	c.printHeader("🎯 Finding")
	c.printFinding(finding)
	fmt.Fprintln(c.writer, strings.Repeat("-", 80))
	return nil
}

func (c *ConsoleThread) Reply(_ context.Context, msg string) {
	c.printHeader("💬 Reply")
	fmt.Fprintln(c.writer, msg)
	fmt.Fprintln(c.writer, strings.Repeat("-", 80))
}

func (c *ConsoleThread) printHeader(title string) {
	fmt.Fprintln(c.writer)
	color.New(color.FgMagenta, color.Bold).Fprintf(c.writer, "=== %s ===\n", title)
}

func (c *ConsoleThread) printFinding(finding model.AlertFinding) {
	severityColor := color.FgYellow
	switch finding.Severity {
	case model.AlertSeverityHigh:
		severityColor = color.FgRed
	case model.AlertSeverityLow:
		severityColor = color.FgGreen
	}

	color.New(severityColor).Fprintf(c.writer, "Severity: %s\n", finding.Severity)
	color.New(color.FgWhite).Fprintf(c.writer, "Summary: %s\n", finding.Summary)
	if finding.Reason != "" {
		color.New(color.FgWhite).Fprintf(c.writer, "Reason: %s\n", finding.Reason)
	}
	if finding.Recommendation != "" {
		color.New(color.FgGreen).Fprintf(c.writer, "Recommendation: %s\n", finding.Recommendation)
	}
}

func (c *ConsoleThread) PostAlertGroups(_ context.Context, groups []model.AlertGroup) error {
	c.printHeader("🔍 Alert Groups")

	for _, group := range groups {
		fmt.Fprintln(c.writer, strings.Repeat("=", 80))
		color.New(color.FgCyan, color.Bold).Fprintf(c.writer, "📦 Group: %s\n", group.Title)
		color.New(color.FgWhite).Fprintf(c.writer, "%s\n", group.Description)

		fmt.Fprintln(c.writer)
		color.New(color.FgYellow).Fprintln(c.writer, "Alerts in this group:")
		for _, alert := range group.Alerts {
			color.New(color.FgGreen).Fprintf(c.writer, "  • %s\n", alert.Title)
		}
		fmt.Fprintln(c.writer)
	}

	fmt.Fprintln(c.writer, strings.Repeat("=", 80))
	return nil
}
