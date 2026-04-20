package websocket_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/controller/websocket"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestNewSessionMessageAddedEvent_CarriesAuthorAndTurn(t *testing.T) {
	turnID := types.TurnID("turn_x")
	msg := &sessModel.Message{
		ID:        types.MessageID("msg_1"),
		SessionID: types.SessionID("sid_1"),
		TurnID:    &turnID,
		Type:      sessModel.MessageTypeUser,
		Content:   "hi",
		Author: &sessModel.Author{
			UserID:      "u",
			DisplayName: "Alice",
		},
		CreatedAt: time.Unix(1700000000, 0).UTC(),
	}
	env := websocket.NewSessionMessageAddedEvent(msg)
	gt.V(t, env.Event).Equal(websocket.EventKindSessionMessageAdded)
	gt.V(t, env.Message == nil).Equal(false)
	gt.V(t, env.Message.Content).Equal("hi")
	gt.V(t, env.Message.Type).Equal("user")
	gt.V(t, env.Message.TurnID != nil && *env.Message.TurnID == "turn_x").Equal(true)
	gt.V(t, env.Message.Author).NotNil()
	gt.V(t, env.Message.Author.DisplayName).Equal("Alice")
}

func TestNewSessionCreatedEvent_PreferesTicketIDPtr(t *testing.T) {
	tid := types.TicketID("tid_1")
	s := &sessModel.Session{
		ID:          "sid_1",
		TicketID:    "legacy_tid",
		TicketIDPtr: &tid,
		Source:      sessModel.SessionSourceSlack,
		UserID:      "u1",
	}
	env := websocket.NewSessionCreatedEvent(s)
	gt.V(t, env.Event).Equal(websocket.EventKindSessionCreated)
	gt.V(t, env.Session == nil).Equal(false)
	gt.V(t, env.Session.TicketID != nil && *env.Session.TicketID == "tid_1").Equal(true)
	gt.V(t, env.Session.Source).Equal("slack")
}

func TestNewSessionCreatedEvent_FallsBackToLegacyTicketID(t *testing.T) {
	s := &sessModel.Session{
		ID:       "sid_1",
		TicketID: "legacy_tid",
		Source:   sessModel.SessionSourceSlack,
	}
	env := websocket.NewSessionCreatedEvent(s)
	gt.V(t, env.Session.TicketID != nil && *env.Session.TicketID == "legacy_tid").Equal(true)
}

func TestNewTurnStartedAndEndedEvents(t *testing.T) {
	started := time.Unix(1700000000, 0).UTC()
	ended := started.Add(5 * time.Second)
	t1 := &sessModel.Turn{
		ID:        types.TurnID("turn_1"),
		SessionID: types.SessionID("sid_1"),
		Status:    sessModel.TurnStatusRunning,
		StartedAt: started,
	}
	env := websocket.NewTurnStartedEvent(t1)
	gt.V(t, env.Event).Equal(websocket.EventKindTurnStarted)
	gt.V(t, env.Turn == nil).Equal(false)
	gt.V(t, env.Turn.Status).Equal("running")

	t1.Status = sessModel.TurnStatusCompleted
	t1.EndedAt = &ended
	env2 := websocket.NewTurnEndedEvent(t1)
	gt.V(t, env2.Event).Equal(websocket.EventKindTurnEnded)
	gt.V(t, env2.Status).Equal(sessModel.TurnStatusCompleted)
	gt.V(t, env2.Timestamp.Equal(ended)).Equal(true)
}

func TestEnvelope_MarshalRoundTrip(t *testing.T) {
	msg := &sessModel.Message{
		ID:        "m",
		SessionID: "s",
		Type:      sessModel.MessageTypeTrace,
		Content:   "t",
		CreatedAt: time.Unix(1700000000, 0).UTC(),
	}
	env := websocket.NewSessionMessageAddedEvent(msg)
	raw, err := env.Marshal()
	gt.NoError(t, err)
	var back map[string]any
	gt.NoError(t, json.Unmarshal(raw, &back))
	gt.V(t, back["event"]).Equal("session_message_added")
	gt.V(t, back["session_id"]).Equal("s")
}
