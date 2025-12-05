package memory

import (
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/activity"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/model/notice"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type Memory struct {
	mu         sync.RWMutex
	activityMu sync.RWMutex
	tagMu      sync.RWMutex
	noticeMu   sync.RWMutex
	memoryMu   sync.RWMutex

	alerts         map[types.AlertID]*alert.Alert
	lists          map[types.AlertListID]*alert.List
	histories      map[types.TicketID][]*ticket.History
	tickets        map[types.TicketID]*ticket.Ticket
	ticketComments map[types.TicketID][]ticket.Comment
	tokens         map[auth.TokenID]*auth.Token
	activities     map[types.ActivityID]*activity.Activity
	tagsV2         map[string]*tag.Tag // New ID-based tags
	notices        map[types.NoticeID]*notice.Notice
	agentMemories  map[types.AgentMemoryID]*memory.AgentMemory

	// Session management
	session *sessionStore

	// Call counter for tracking method invocations
	callCounts map[string]int
	callMu     sync.RWMutex

	eb *goerr.Builder
}

var _ interfaces.Repository = &Memory{}

func New() *Memory {
	return &Memory{
		alerts:         make(map[types.AlertID]*alert.Alert),
		lists:          make(map[types.AlertListID]*alert.List),
		histories:      make(map[types.TicketID][]*ticket.History),
		tickets:        make(map[types.TicketID]*ticket.Ticket),
		ticketComments: make(map[types.TicketID][]ticket.Comment),
		tokens:         make(map[auth.TokenID]*auth.Token),
		activities:     make(map[types.ActivityID]*activity.Activity),
		tagsV2:         make(map[string]*tag.Tag),
		notices:        make(map[types.NoticeID]*notice.Notice),
		agentMemories:  make(map[types.AgentMemoryID]*memory.AgentMemory),
		session:        newSessionStore(),
		callCounts:     make(map[string]int),
		eb:             goerr.NewBuilder(goerr.TV(errs.RepositoryKey, "memory")),
	}
}

// incrementCallCount safely increments the call counter for a method
func (r *Memory) incrementCallCount(methodName string) {
	r.callMu.Lock()
	defer r.callMu.Unlock()
	r.callCounts[methodName]++
}

// GetCallCount returns the number of times a method has been called
func (r *Memory) GetCallCount(methodName string) int {
	r.callMu.RLock()
	defer r.callMu.RUnlock()
	return r.callCounts[methodName]
}

// GetAllCallCounts returns a copy of all call counts
func (r *Memory) GetAllCallCounts() map[string]int {
	r.callMu.RLock()
	defer r.callMu.RUnlock()

	counts := make(map[string]int)
	for k, v := range r.callCounts {
		counts[k] = v
	}
	return counts
}

// ResetCallCounts clears all call counters
func (r *Memory) ResetCallCounts() {
	r.callMu.Lock()
	defer r.callMu.Unlock()
	r.callCounts = make(map[string]int)
}
