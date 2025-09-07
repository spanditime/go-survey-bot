package conversation

import (
	"fmt"
	"log"
	"sync"
)

// interfaces for conversation management
type User struct {
	Name     string
	Surname  string
	Id       string
	UserName string
	// bio string,
}

func (u User) FullName() string {
	return fmt.Sprintf("%s %s", u.Name, u.Surname)
}

type Update interface {
	Provider() string
	ChatID() string
	GetSender() User
	GetMessage() string
	Reply(text string) error
	ReplyWithKeyboard(text string, kb []string) error
}

type Agent interface {
	Run() (chan Update, error)
}

type Handler interface {
	Handle(ctx Ctx) error
	Welcome(ctx Ctx) error
}

type keystorage = map[string]interface{}

type session struct {
	Handler    Handler
	KeyStorage keystorage
}

type agentRunner struct{
	agent Agent
	sessions map[string]session
	channel chan Update
}

type Manager struct {
	runners    []agentRunner
	entryPoint func() Handler
}

type Ctx interface {
	Close()
	SetNext(next Handler)
	Next() (Handler, bool)
	Transitioning() bool
	Closed() bool
	Update() Update
	Finalized() bool
	GetKey(key string) (interface{}, bool)
	SetKey(key string, value interface{})
}

type ctx struct {
	update  Update
	next    Handler
	closed  bool
	storage *keystorage
}

func newContext(update Update, ks *keystorage) Ctx {
	return &ctx{
		update:  update,
		closed:  false,
		storage: ks,
	}
}

func (ctx *ctx) Close()               { ctx.closed = true }
func (ctx *ctx) SetNext(next Handler) { ctx.next = next }
func (ctx *ctx) Next() (Handler, bool) {
	return ctx.next, ctx.Transitioning() && !ctx.Closed()
}
func (ctx *ctx) Closed() bool        { return ctx.closed }
func (ctx *ctx) Update() Update      { return ctx.update }
func (ctx *ctx) Finalized() bool     { return false }
func (ctx *ctx) Transitioning() bool { return ctx.next != nil }
func (ctx *ctx) GetKey(key string) (interface{}, bool) {
	value, found := (*ctx.storage)[key]
	return value, found
}
func (ctx *ctx) SetKey(key string, value interface{}) { (*ctx.storage)[key] = value }

func newAgentRunner(agent Agent) agentRunner{
	return agentRunner{
		agent: agent,
		sessions: make(map[string]session),
	};
}

func NewManager(entryPoint func() Handler) *Manager {
	return &Manager{
		runners:    make([]agentRunner,0),
		entryPoint: entryPoint,
	}
}


func (m *Manager) AddAgent(agent Agent){
	m.runners = append(m.runners, newAgentRunner(agent))
}

func (m *Manager) Run() error {
	var wg sync.WaitGroup
	if len(m.runners) == 0 {
		return fmt.Errorf("Trying to start with no registered agents");
	}
	defer m.Stop();
	for _, runner := range m.runners{
		ec := runner.Run(m.entryPoint, &wg);
		if ec != nil {
			return ec;
		}
	}

	// wait for all runners to stop
	wg.Wait()
	return nil
}

func (m *Manager) Stop() {
	for _, runner := range m.runners{
		runner.Stop()
	}
}

func (m *agentRunner) handle(handle *Handler, ctx Ctx) error {
	(*handle).Handle(ctx)
	if next, v := ctx.Next(); v {
		(*handle) = next
		(*handle).Welcome(ctx)
	}
	if closed := ctx.Closed(); closed {
		(*handle) = nil
	}
	return nil
}

func (m *agentRunner) Run(entryPoint func() Handler, wg *sync.WaitGroup) error {
	wg.Add(1)
	ch, err := m.agent.Run()
	if err != nil {
		return err
	}
	m.channel = ch
	go func() {
		for update := range m.channel {
			var sess session = session{}
			var found bool
			chatID := update.ChatID()
			if sess, found = m.sessions[chatID]; !found {
				m.sessions[chatID] = sess
			}
			if sess.KeyStorage == nil {
				sess.KeyStorage = make(keystorage)
			}

			ctx := newContext(update, &sess.KeyStorage)
			if sess.Handler == nil {
				sess.Handler = entryPoint()
				sess.Handler.Welcome(ctx)
			}
			if err = m.handle(&sess.Handler, ctx); err != nil {
				// todo: log errors
				log.Println(err)
			}
			m.sessions[chatID] = sess
		}
		wg.Done()
	}()

	return nil
}

func (r *agentRunner) Stop() {
	close(r.channel)
}

