package main

import (
	"fmt"
	"os"
	"time"

	"log"

	"github.com/spanditime/go-survey-bot/conversation"
	tg "github.com/spanditime/go-survey-bot/telegram"
	"github.com/spanditime/go-survey-bot/vk"
)

// library part

// app logic part

func newYesNoConversationHandler(question string, welcome conversation.Action, no conversation.Action, yes conversation.Action, cancel conversation.Action) *conversation.OptionsHandler {
	handlers := conversation.OptionsHandlers{
		Yes: yes,
		No:  no,
	}
	handlers[Cancel] = cancel
	return conversation.NewOptionsHandler(welcome, question, handlers, conversation.EmptyAction())
}

const (
	Cancel        = "Отмена"
	Submit        = "Отправить"
	ChangeName    = "Изменить имя"
	ChangeAge     = "Изменить возраст"
	ChangeCity    = "Изменить готовность к очным встречам"
	ChangeRequest = "Изменить запрос"
	ChangeHealth  = "Изменить информацию о здоровье"
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
	EnterHealth  = "Есть ли у Вас жалобы на здоровье, хронические заболевания? Если да, пожалуйста, укажите их."
	EnterContact = "Как мы можем связаться с вами? Просим оставить вас ссылку на соц. сети, почту или номер телефона (и предпочтительный тип связи по нему)."
	Accept       = "Информация верна?"
	Thanks       = "Благодарим за обращение! Мы рассмотрим заявку и свяжемся с Вами в случае, если найдется специалист."
	Yes          = "Да"
	No           = "Нет"

	NameKey    = "name"
	AgeKey     = "age"
	CityKey    = "city"
	RequestKey = "request"
	HealthKey  = "health"
	ContactKey = "contact"

	DATE_SAVE_LOCATION    = "SURV_DATE_SAVE_LOCATION"
	GOOGLE_CRED           = "GOOGLE_CREDENTIALS_FILE"
	GOOGLE_SHEET_NAME     = "GOOGLE_SHEET_NAME"
	GOOGLE_SPREADSHEET_ID = "GOOGLE_SPREADSHEET_ID"
	TELEGRAM_TOKEN        = "TELEGRAM_BOT_TOKEN"
	VK_TOKEN              = "VK_BOT_TOKEN"
)

type surveyFabric struct {
	db *SurveyDB
}

func (f *surveyFabric) newStartQuestion() conversation.Handler {
	handle := func(answer string, ctx conversation.Ctx) error {
		if answer == "/start" {
			return conversation.TransitionStageAction(f.newWelcomeQuestion)(answer, ctx)
		}
		return nil
	}
	cancel := conversation.TransitionStageAction(f.newStartQuestion)
	handlers := conversation.OptionsHandlers{
		"/start": handle,
	}
	return conversation.NewOptionsHandler(conversation.EmptyAction(), StartMessage, handlers, cancel)
}

func (f *surveyFabric) newWelcomeQuestion() conversation.Handler {
	next := conversation.TransitionStageActionCtx(f.newNameQuestion(false))
	cancel := conversation.TransitionStageAction(f.newStartQuestion)
	return newYesNoConversationHandler(GoToSurvey, conversation.SendTextAction(WelcomeMessage, conversation.EmptyAction()), cancel, next, cancel)
}

func saveSurveyAnswer(key string, fall bool, save conversation.Action, next conversation.Action) conversation.Action {
	var action conversation.Action
	if fall {
		action = conversation.SaveKeyAction(key, save)
	} else {
		action = conversation.SaveKeyAction(key, next)
	}
	return action
}

func (f *surveyFabric) newNameQuestion(fall bool) func(answer string, ctx conversation.Ctx) conversation.Handler {
	return func(answer string, ctx conversation.Ctx) conversation.Handler {
		cancel := conversation.TransitionStageAction(f.newStartQuestion)
		defaultName := ctx.Update().GetSender().FullName()
		save := saveSurveyAnswer(NameKey, fall, conversation.TransitionStageActionCtx(f.newSaveQuestion), conversation.TransitionStageActionCtx(f.newAgeQuestion(false)))
		handlers := conversation.OptionsHandlers{
			Cancel:      cancel,
			defaultName: save,
		}
		return conversation.NewOptionsHandler(conversation.EmptyAction(), EnterName, handlers, save)
	}
}

func (f *surveyFabric) newAgeQuestion(fall bool) func(answer string, ctx conversation.Ctx) conversation.Handler {
	return func(answer string, ctx conversation.Ctx) conversation.Handler {
		cancel := conversation.TransitionStageAction(f.newStartQuestion)
		save := saveSurveyAnswer(AgeKey, fall, conversation.TransitionStageActionCtx(f.newSaveQuestion), conversation.TransitionStageActionCtx(f.newCityQuestion(false)))
		handlers := conversation.OptionsHandlers{
			Cancel: cancel,
		}
		// todo: if have age - add age
		if false {
			defaultAge := "20"
			handlers[defaultAge] = save
		}
		return conversation.NewOptionsHandler(conversation.EmptyAction(), EnterAge, handlers, save)
	}
}

func (f *surveyFabric) newCityQuestion(fall bool) func(answer string, ctx conversation.Ctx) conversation.Handler {
	return func(answer string, ctx conversation.Ctx) conversation.Handler {
		cancel := conversation.TransitionStageAction(f.newStartQuestion)
		save := saveSurveyAnswer(CityKey, fall, conversation.TransitionStageActionCtx(f.newSaveQuestion), conversation.TransitionStageActionCtx(f.newRequestQuestion(false)))
		handlers := conversation.OptionsHandlers{
			Cancel: cancel,
			Yes:    save,
		}
		return conversation.NewOptionsHandler(conversation.EmptyAction(), EnterCity, handlers, save)
	}
}

func (f *surveyFabric) newRequestQuestion(fall bool) func(answer string, ctx conversation.Ctx) conversation.Handler {
	return func(answer string, ctx conversation.Ctx) conversation.Handler {
		cancel := conversation.TransitionStageAction(f.newStartQuestion)
		save := saveSurveyAnswer(RequestKey, fall, conversation.TransitionStageActionCtx(f.newSaveQuestion), conversation.TransitionStageActionCtx(f.newHealthQuestion(false)))
		handlers := conversation.OptionsHandlers{
			Cancel: cancel,
		}
		return conversation.NewOptionsHandler(conversation.EmptyAction(), EnterRequest, handlers, save)
	}
}

func (f *surveyFabric) newHealthQuestion(fall bool) func(answer string, ctx conversation.Ctx) conversation.Handler {
	return func(answer string, ctx conversation.Ctx) conversation.Handler {
		cancel := conversation.TransitionStageAction(f.newStartQuestion)
		save := saveSurveyAnswer(HealthKey, fall, conversation.TransitionStageActionCtx(f.newSaveQuestion), conversation.TransitionStageActionCtx(f.newContactQuestion))
		handlers := conversation.OptionsHandlers{
			Cancel: cancel,
			Yes:    save,
			No:     save,
		}
		return conversation.NewOptionsHandler(conversation.EmptyAction(), EnterHealth, handlers, save)
	}
}

func (f *surveyFabric) newContactQuestion(answer string, ctx conversation.Ctx) conversation.Handler {
	cancel := conversation.TransitionStageAction(f.newStartQuestion)
	save := conversation.SaveKeyAction(ContactKey, conversation.TransitionStageActionCtx(f.newSaveQuestion))
	handlers := conversation.OptionsHandlers{
		Cancel: cancel,
	}
	// todo: if have contact - add it
	username := ctx.Update().GetSender().UserName
	if username != "" {
		defaultContact := fmt.Sprint(ctx.Update().Provider(), ": ", username)
		handlers[defaultContact] = save
	}
	return conversation.NewOptionsHandler(conversation.EmptyAction(), EnterContact, handlers, save)
}

func (f *surveyFabric) newSaveQuestion(answer string, ctx conversation.Ctx) conversation.Handler {
	name, _ := ctx.GetKey(NameKey)
	age, _ := ctx.GetKey(AgeKey)
	city, _ := ctx.GetKey(CityKey)
	request, _ := ctx.GetKey(RequestKey)
	health, _ := ctx.GetKey(HealthKey)
	contact, _ := ctx.GetKey(ContactKey)
	question := fmt.Sprintf("%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s",
		EnterName, name,
		EnterAge, age,
		EnterCity, city,
		EnterRequest, request,
		EnterHealth, health,
		EnterContact, contact,
		Accept)
	saveSurvey := func(answer string, ctx conversation.Ctx) error {
		if err := conversation.SendTextAction(Thanks, conversation.EmptyAction())(answer, ctx); err != nil {
			return err
		}

		id := ctx.Update().ChatID()
		contact = fmt.Sprintf("%s (%s: %s)", contact, ctx.Update().Provider(), ctx.Update().GetSender().UserName)
		err := f.db.WriteAnswers(
			id,
			time.Now(),
			name,
			age,
			city,
			request,
			health,
			contact,
		)
		if err != nil {
			// todo: log an error
			log.Printf("Cant write survey results for user %s %s: %v", id, contact, err)
			// todo: notify user
		}
		return conversation.TransitionStageAction(f.newStartQuestion)(answer, ctx)
	}
	return conversation.NewOptionsHandler(conversation.EmptyAction(), question, conversation.OptionsHandlers{
		Submit:        saveSurvey,
		ChangeName:    conversation.TransitionStageActionCtx(f.newNameQuestion(true)),
		ChangeAge:     conversation.TransitionStageActionCtx(f.newAgeQuestion(true)),
		ChangeCity:    conversation.TransitionStageActionCtx(f.newCityQuestion(true)),
		ChangeRequest: conversation.TransitionStageActionCtx(f.newRequestQuestion(true)),
		ChangeHealth:  conversation.TransitionStageActionCtx(f.newHealthQuestion(true)),
		ChangeContact: conversation.TransitionStageActionCtx(f.newContactQuestion),
		Cancel: func(answer string, ctx conversation.Ctx) error {
			// note: clear context storage might be needed
			return conversation.TransitionStageAction(f.newStartQuestion)(answer, ctx)
		},
	}, conversation.EmptyAction())
}

func newSurveyFabric() *surveyFabric {
	return &surveyFabric{
		db: newSuveyDB(os.Getenv(GOOGLE_CRED), os.Getenv(GOOGLE_SPREADSHEET_ID), os.Getenv(GOOGLE_SHEET_NAME), os.Getenv(DATE_SAVE_LOCATION)),
	}
}

func main() {
	survey := newSurveyFabric()

	manager := conversation.NewManager(survey.newStartQuestion)

	// register tg bot agent
	if tgtoken, use := os.LookupEnv(TELEGRAM_TOKEN); use {
		tgbot, err := tg.NewBot(tgtoken)
		if err != nil {
			panic(err)
		}
		manager.AddAgent(tgbot)
	}

	// register vk bot agent
	if vktoken, use := os.LookupEnv(VK_TOKEN); use {
		vkbot, err := vk.NewBot(vktoken)
		if err != nil {
			panic(err)
		}
		manager.AddAgent(vkbot)
	}

	log.Println(manager.Run().Error())
}
