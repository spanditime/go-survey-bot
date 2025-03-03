package main

import (
	"fmt"
	"os"
	"time"

	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// library part

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

type keystorage = map[string]interface{}

type ConversationManager struct {
	keyStorage map[string]keystorage
	sessions   map[string]ConversationHandler
	agent      Agent
	entryPoint func() ConversationHandler
}

func NewConversationManager(agent Agent, entryPoint func() ConversationHandler) *ConversationManager {
	return &ConversationManager{
		agent:      agent,
		entryPoint: entryPoint,
		sessions:   make(map[string]ConversationHandler),
		keyStorage: make(map[string]map[string]interface{}),
	}
}

func (m *ConversationManager) handle(handle *ConversationHandler, ctx ConvCtx) error {
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

func (m *ConversationManager) Run() error {
	ch, err := m.agent.Run()
	if err != nil {
		return err
	}

	for update := range ch {
		var ks keystorage
		var found bool
		if ks, found = m.keyStorage[update.ChatID()]; !found || ks == nil {
			ks = make(keystorage)
		}

		ctx := newConvContext(update, &ks)
		var handle ConversationHandler
		if handle, found = m.sessions[update.ChatID()]; !found || handle == nil {
			handle = m.entryPoint()
			m.sessions[update.ChatID()] = handle
			handle.Welcome(ctx)
		}
		if err = m.handle(&handle, ctx); err != nil {
			// todo: log errors
			log.Println(err)
		}
		m.sessions[update.ChatID()] = handle
		m.keyStorage[update.ChatID()] = ks
	}

	return nil
}

type ConvCtx interface {
	Close()
	SetNext(next ConversationHandler)
	Next() (ConversationHandler, bool)
	Transitioning() bool
	Closed() bool
	Update() Update
	Finalized() bool
	GetKey(key string) (interface{}, bool)
	SetKey(key string, value interface{})
}

type Ctx struct {
	update  Update
	next    ConversationHandler
	closed  bool
	storage *keystorage
}

func newConvContext(update Update, ks *keystorage) ConvCtx {
	return &Ctx{
		update:  update,
		closed:  false,
		storage: ks,
	}
}

func (ctx *Ctx) Close()                           { ctx.closed = true }
func (ctx *Ctx) SetNext(next ConversationHandler) { ctx.next = next }
func (ctx *Ctx) Next() (ConversationHandler, bool) {
	return ctx.next, ctx.Transitioning() && !ctx.Closed()
}
func (ctx *Ctx) Closed() bool        { return ctx.closed }
func (ctx *Ctx) Update() Update      { return ctx.update }
func (ctx *Ctx) Finalized() bool     { return false }
func (ctx *Ctx) Transitioning() bool { return ctx.next != nil }
func (ctx *Ctx) GetKey(key string) (interface{}, bool) {
	value, found := (*ctx.storage)[key]
	return value, found
}
func (ctx *Ctx) SetKey(key string, value interface{}) { (*ctx.storage)[key] = value }

type ConversationHandler interface {
	Handle(ctx ConvCtx) error
	Welcome(ctx ConvCtx) error
}

// Agent implementation tg

type TelegramAgent struct {
	api *tgbotapi.BotAPI
}

func NewTelegramBot(token string) (Agent, error) {
	botapi, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &TelegramAgent{
		api: botapi,
	}, err
}

func (tg *TelegramAgent) Run() (chan Update, error) {
	u := tgbotapi.NewUpdate(0)
	tg_updates := tg.api.GetUpdatesChan(u)
	updates := make(chan Update)
	go func() {
		for tg_update := range tg_updates {
			update := newTelegramUpdate(tg.api, tg_update)
			updates <- update
		}
	}()
	return updates, nil
}

type TelegramUpdate struct {
	api    *tgbotapi.BotAPI
	update tgbotapi.Update
}

func newTelegramUpdate(api *tgbotapi.BotAPI, tg_update tgbotapi.Update) *TelegramUpdate {
	return &TelegramUpdate{
		api:    api,
		update: tg_update,
	}
}

func (upd *TelegramUpdate) Provider() string {
	return "tg"
}
func (upd *TelegramUpdate) ChatID() string {
	ch := upd.update.FromChat()
	if ch != nil {
		return fmt.Sprint("tg", ch.ID)
	}
	return "tg" // todo: thats technically an error
}
func (upd *TelegramUpdate) GetSender() User {
	sent_from := upd.update.SentFrom()
	var name, surname, username, id string
	if sent_from != nil {
		name, surname, username, id = sent_from.FirstName, sent_from.LastName, sent_from.UserName, fmt.Sprint("tg", sent_from.ID)
	}
	return User{
		Name:     name,
		Surname:  surname,
		Id:       id,
		UserName: username,
	}
}
func (upd *TelegramUpdate) GetMessage() string {
	msg := upd.update.Message
	if msg != nil {
		return msg.Text
	}
	return ""
}
func (upd *TelegramUpdate) Reply(text string) error {
	reply_to := upd.update.Message
	if reply_to != nil {
		msg := tgbotapi.NewMessage(reply_to.Chat.ID, text)
		_, err := upd.api.Send(msg)
		if err != nil {
			return err
		}
	}
	//todo log an error here)
	return nil
}
func (upd *TelegramUpdate) ReplyWithKeyboard(text string, kb []string) error {
	reply_to := upd.update.Message
	if reply_to != nil {
		var buttons [][]tgbotapi.KeyboardButton = make([][]tgbotapi.KeyboardButton, len(kb))
		for i, b := range kb {
			buttons[i] = []tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton(b)}
		}
		keyb := tgbotapi.NewReplyKeyboard(buttons...)
		msg := tgbotapi.NewMessage(reply_to.Chat.ID, text)
		msg.ReplyMarkup = keyb
		_, err := upd.api.Send(msg)
		if err != nil {
			return err
		}
	}
	//todo log an error here)
	return nil
}

// conversation handler helpers

// conversation handler with predefined options
type ConversationAction = func(text string, ctx ConvCtx) error
type ConversationStage = func() ConversationHandler
type ConversationStageCtx = func(answer string, ctx ConvCtx) ConversationHandler

func EmptyAction() ConversationAction { return func(answer string, ctx ConvCtx) error { return nil } }
func SendTextAction(text string, next ConversationAction) ConversationAction {
	return func(answer string, ctx ConvCtx) error {
		ctx.Update().Reply(text)
		next(answer, ctx)
		return nil
	}
}
func SendTextWithKeyboardAction(text string, keyboard []string, next ConversationAction) ConversationAction {
	return func(answer string, ctx ConvCtx) error {
		ctx.Update().ReplyWithKeyboard(text, keyboard)
		next(answer, ctx)
		return nil
	}
}
func TransitionStageAction(next ConversationStage) ConversationAction {
	return func(answer string, ctx ConvCtx) error {
		ctx.SetNext(next())
		return nil
	}
}
func TransitionStageActionCtx(next ConversationStageCtx) ConversationAction {
	return func(answer string, ctx ConvCtx) error {
		ctx.SetNext(next(answer, ctx))
		return nil
	}
}

func SaveKeyAction(key string, next ConversationAction) ConversationAction {
	return func(answer string, ctx ConvCtx) error {
		ctx.SetKey(key, answer)
		return next(answer, ctx)
	}
}

type ConversationAnswerHandler struct {
	welcome       ConversationAction
	question      string
	answerHandler ConversationAction
}

func NewConversationAnswerHandler(welcome ConversationAction, question string, answerHandler ConversationAction) ConversationHandler {
	return &ConversationAnswerHandler{
		welcome:       welcome,
		question:      question,
		answerHandler: answerHandler,
	}
}

func (handler *ConversationAnswerHandler) sendQuestion(ctx ConvCtx) error {
	return ctx.Update().Reply(handler.question)
}
func (handler *ConversationAnswerHandler) Welcome(ctx ConvCtx) error {
	err := handler.welcome("", ctx)
	if err != nil {
		return err
	}
	return handler.sendQuestion(ctx)
}

func (handler *ConversationAnswerHandler) Handle(ctx ConvCtx) error {
	var answer string = ctx.Update().GetMessage()
	return handler.answerHandler(answer, ctx)
}

type ConversationOptionsHandlers map[string]ConversationAction

func (handlers *ConversationOptionsHandlers) Options() []string {
	keys := make([]string, len(*handlers))
	i := 0
	for k := range *handlers {
		keys[i] = k
		i++
	}
	return keys
}

type ConversationOptionsHandler struct {
	welcome        ConversationAction
	optionHandlers ConversationOptionsHandlers
	question       string
	answerHandler  ConversationAction
}

func NewConversationOptionsHandler(welcome ConversationAction, question string, handlers ConversationOptionsHandlers, answerHandler ConversationAction) *ConversationOptionsHandler {
	return &ConversationOptionsHandler{
		welcome:        welcome,
		optionHandlers: handlers,
		question:       question,
		answerHandler:  answerHandler,
	}
}

func (handler *ConversationOptionsHandler) sendQuestion(ctx ConvCtx) error {
	return SendTextWithKeyboardAction(handler.question, handler.optionHandlers.Options(), EmptyAction())("", ctx)
}

func (handler *ConversationOptionsHandler) handleOption(option string, ctx ConvCtx) error {
	var err error
	if h, found := handler.optionHandlers[option]; found {
		err = h(option, ctx)
	} else {
		err = handler.answerHandler(option, ctx)
	}
	return err
}

func (handler *ConversationOptionsHandler) Handle(ctx ConvCtx) error {
	var answer string = ctx.Update().GetMessage()
	if err := handler.handleOption(answer, ctx); err != nil {
		return err
	}
	if !ctx.Finalized() && !ctx.Transitioning() {
		return handler.sendQuestion(ctx)
	}
	return nil
}

func (handler *ConversationOptionsHandler) Welcome(ctx ConvCtx) error {
	if err := handler.welcome("", ctx); err != nil {
		return err
	}
	return handler.sendQuestion(ctx)
}

// app logic part

func newYesNoConversationHandler(question string, welcome ConversationAction, no ConversationAction, yes ConversationAction, cancel ConversationAction) *ConversationOptionsHandler {
	handlers := ConversationOptionsHandlers{
		Yes: yes,
		No:  no,
	}
	handlers[Cancel] = cancel
	return NewConversationOptionsHandler(welcome, question, handlers, EmptyAction())
}

const (
	Cancel        = "Отмена"
	Submit        = "Отправить"
	ChangeName    = "Изменить имя"
	ChangeAge     = "Изменить возраст"
	ChangeCity    = "Изменить готовность к очным встречам"
	ChangeRequest = "Изменить запрос"
	ChangeContact = "Изменить контактные данные"

	StartMessage = "Используйте /start что бы начать."

	WelcomeMessage = `Добрый день, уважаемые друзья! Мы - студенты направления клинический психологии в г. Дубна. 

  Здесь Вы можете оставить заявку на бесплатное психологическое консультирование. Консультации проводятся под супервизией преподавателей (разбором случаев без обозначения личных данных для определения корректного пути работы).
  В свою очередь, мы ожидаем от Вас готовность серьезно работать над своей проблемой совместно с психологом.

  Спектры проблем и переживаний, с которыми Вы можете к нам обратиться:
  - сложности в межличностных отношениях (дружеских, романтических, семейных и т.д.)
  - трудности в учёбе (стресс, страх публичных выступлений, тревожность, прокрастинация, тремор при общении с коллегами и преподавателями, страх совершать ошибки);
  - обеспокоенность своим психологическим состоянием (вредные привычки, нестабильная самооценка и эмоциональность, страхи, трудности в проявлении чувств и сопереживании, стремление к соперничеству, психосоматические симптомы, болезненное восприятие критики, невозможность "понять себя"). 

  Если у Вас есть вопросы - можете задать их в @karevaina или по почте: clin.psy@mail.ru.`
	GoToSurvey   = `В данный момент ведется активный набор на консультации. Хотите оставить заявку?`
	EnterName    = "Как мы можем к Вам обращаться?"
	EnterAge     = "Подскажите, сколько Вам лет?"
	EnterCity    = "Вы готовы приходить на встречи очно в городе Дубна? (К сожалению, не все студенты готовы брать на онлайн-консультации, поэтому вероятность попасть на очное консультирование выше, чем онлайн)"
	EnterRequest = "Пожалуйста, попробуйте описать Ваш запрос в одном или двух предложениях (что Вас беспокоит или что хотелось бы изменить)."
	EnterContact = "Как мы можем связаться с вами? Просим оставить вас ссылку на соц. сети, почту или номер телефона (и предпочтительный тип связи по нему)."
	Accept       = "Информация верна?"
	Thanks       = "Благодарим за обращение! Мы рассмотрим заявку и свяжемся с Вами в случае, если найдется специалист."
	Yes          = "Да"
	No           = "Нет"

	NameKey    = "name"
	AgeKey     = "age"
	CityKey    = "city"
	RequestKey = "request"
	ContactKey = "contact"

	GOOGLE_CRED           = "GOOGLE_CREDENTIALS_FILE"
	GOOGLE_SHEET_NAME     = "GOOGLE_SHEET_NAME"
	GOOGLE_SPREADSHEET_ID = "GOOGLE_SPREADSHEET_ID"
	TELEGRAM_TOKEN        = "TELEGRAM_BOT_TOKEN"
)

type surveyFabric struct {
	db *SurveyDB
}

func (f *surveyFabric) newStartQuestion() ConversationHandler {
	handle := func(answer string, ctx ConvCtx) error {
		if answer == "/start" {
			return TransitionStageAction(f.newWelcomeQuestion)(answer, ctx)
		}
		return nil
	}
	return NewConversationAnswerHandler(EmptyAction(), StartMessage, handle)
}

func (f *surveyFabric) newWelcomeQuestion() ConversationHandler {
	next := TransitionStageActionCtx(f.newNameQuestion(false))
	cancel := TransitionStageAction(f.newStartQuestion)
	return newYesNoConversationHandler(GoToSurvey, SendTextAction(WelcomeMessage, EmptyAction()), cancel, next, cancel)
}

func saveSurveyAnswer(key string, fall bool, save ConversationAction, next ConversationAction) ConversationAction {
	var action ConversationAction
	if fall {
		action = SaveKeyAction(key, save)
	} else {
		action = SaveKeyAction(key, next)
	}
	return action
}

func (f *surveyFabric) newNameQuestion(fall bool) func(answer string, ctx ConvCtx) ConversationHandler {
	return func(answer string, ctx ConvCtx) ConversationHandler {
		cancel := TransitionStageAction(f.newStartQuestion)
		defaultName := ctx.Update().GetSender().FullName()
		save := saveSurveyAnswer(NameKey, fall, TransitionStageActionCtx(f.newSaveQuestion), TransitionStageActionCtx(f.newAgeQuestion(false)))
		handlers := ConversationOptionsHandlers{
			Cancel:      cancel,
			defaultName: save,
		}
		return NewConversationOptionsHandler(EmptyAction(), EnterName, handlers, save)
	}
}

func (f *surveyFabric) newAgeQuestion(fall bool) func(answer string, ctx ConvCtx) ConversationHandler {
	return func(answer string, ctx ConvCtx) ConversationHandler {
		cancel := TransitionStageAction(f.newStartQuestion)
		save := saveSurveyAnswer(AgeKey, fall, TransitionStageActionCtx(f.newSaveQuestion), TransitionStageActionCtx(f.newCityQuestion(false)))
		handlers := ConversationOptionsHandlers{
			Cancel: cancel,
		}
		// todo: if have age - add age
		if false {
			defaultAge := "20"
			handlers[defaultAge] = save
		}
		return NewConversationOptionsHandler(EmptyAction(), EnterAge, handlers, save)
	}
}

func (f *surveyFabric) newCityQuestion(fall bool) func(answer string, ctx ConvCtx) ConversationHandler {
	return func(answer string, ctx ConvCtx) ConversationHandler {
		cancel := TransitionStageAction(f.newStartQuestion)
		save := saveSurveyAnswer(CityKey, fall, TransitionStageActionCtx(f.newSaveQuestion), TransitionStageActionCtx(f.newRequestQuestion(false)))
		handlers := ConversationOptionsHandlers{
			Cancel: cancel,
			Yes:    save,
		}
		return NewConversationOptionsHandler(EmptyAction(), EnterCity, handlers, save)
	}
}

func (f *surveyFabric) newRequestQuestion(fall bool) func(answer string, ctx ConvCtx) ConversationHandler {
	return func(answer string, ctx ConvCtx) ConversationHandler {
		cancel := TransitionStageAction(f.newStartQuestion)
		save := saveSurveyAnswer(RequestKey, fall, TransitionStageActionCtx(f.newSaveQuestion), TransitionStageActionCtx(f.newContactQuestion))
		handlers := ConversationOptionsHandlers{
			Cancel: cancel,
		}
		return NewConversationOptionsHandler(EmptyAction(), EnterRequest, handlers, save)
	}
}

func (f *surveyFabric) newContactQuestion(answer string, ctx ConvCtx) ConversationHandler {
	cancel := TransitionStageAction(f.newStartQuestion)
	save := SaveKeyAction(ContactKey, TransitionStageActionCtx(f.newSaveQuestion))
	handlers := ConversationOptionsHandlers{
		Cancel: cancel,
	}
	// todo: if have contact - add it
	username := ctx.Update().GetSender().UserName
	if username != "" {
		defaultContact := fmt.Sprint(ctx.Update().Provider(), ": @", username)
		handlers[defaultContact] = save
	}
	return NewConversationOptionsHandler(EmptyAction(), EnterContact, handlers, save)
}

func (f *surveyFabric) newSaveQuestion(answer string, ctx ConvCtx) ConversationHandler {
	name, _ := ctx.GetKey(NameKey)
	age, _ := ctx.GetKey(AgeKey)
	city, _ := ctx.GetKey(CityKey)
	request, _ := ctx.GetKey(RequestKey)
	contact, _ := ctx.GetKey(ContactKey)
	question := fmt.Sprintf("%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s",
		EnterName, name,
		EnterAge, age,
		EnterCity, city,
		EnterRequest, request,
		EnterContact, contact,
		Accept)
	saveSurvey := func(answer string, ctx ConvCtx) error {
		if err := SendTextAction(Thanks, EmptyAction())(answer, ctx); err != nil {
			return err
		}

		id := ctx.Update().ChatID()
		contact = fmt.Sprintf("%s (%s: @%s)", contact, ctx.Update().Provider(), ctx.Update().GetSender().UserName)
		err := f.db.WriteAnswers(
			id,
			time.Now(),
			name,
			age,
			city,
			request,
			contact,
		)
		if err != nil {
			// todo: log an error
			log.Printf("Cant write survey results for user %s %s: %v", id, contact, err)
			// todo: notify user
		}
		return TransitionStageAction(f.newStartQuestion)(answer, ctx)
	}
	return NewConversationOptionsHandler(EmptyAction(), question, ConversationOptionsHandlers{
		Submit:        saveSurvey,
		ChangeName:    TransitionStageActionCtx(f.newNameQuestion(true)),
		ChangeAge:     TransitionStageActionCtx(f.newAgeQuestion(true)),
		ChangeCity:    TransitionStageActionCtx(f.newCityQuestion(true)),
		ChangeRequest: TransitionStageActionCtx(f.newRequestQuestion(true)),
		ChangeContact: TransitionStageActionCtx(f.newContactQuestion),
		Cancel: func(answer string, ctx ConvCtx) error {
			// note: clear context storage might be needed
			return TransitionStageAction(f.newStartQuestion)(answer, ctx)
		},
	}, EmptyAction())
}

func newSurveyFabric() *surveyFabric {
	return &surveyFabric{
		db: newSuveyDB(os.Getenv(GOOGLE_CRED), os.Getenv(GOOGLE_SPREADSHEET_ID), os.Getenv(GOOGLE_SHEET_NAME)),
	}
}

func main() {
	tgbot, err := NewTelegramBot(os.Getenv(TELEGRAM_TOKEN))
	if err != nil {
		panic(err)
	}
	survey := newSurveyFabric()

	log.Println(NewConversationManager(tgbot, survey.newStartQuestion).Run().Error())
}
