ackage main

import (
	"fmt"
	"os"

	"log"

	"github.com/spanditime/go-survey-bot/conversation"
	"github.com/spanditime/go-survey-bot/survey"
	tg "github.com/spanditime/go-survey-bot/telegram"
	"github.com/spanditime/go-survey-bot/vk"
)

// library part

// app logic part


const (
	SURV_DATE_SAVE_LOCATION = "SURV_DATE_SAVE_LOCATION"
	GOOGLE_CREDENTIALS_JSON = "GOOGLE_CREDENTIALS_JSON"
	GOOGLE_SHEET_NAME       = "GOOGLE_SHEET_NAME"
	GOOGLE_SPREADSHEET_ID   = "GOOGLE_SPREADSHEET_ID"
	TELEGRAM_BOT_TOKEN      = "TELEGRAM_BOT_TOKEN"
	VK_BOT_TOKEN            = "VK_BOT_TOKEN"
)

type config struct {
	DateSaveLocation string

	GoogleCredsJson string
	GoogleSheetName string
	GoogleSheetId   string

	TgToken *string
	VkToken *string
}

func lookupVar(variable string) (string, error) {
	if val, use := os.LookupEnv(variable); use {
		return val, nil
	}
	return "", fmt.Errorf("Environment variable %v is not present", variable)
}

func getVar(variable string) (string, error) {
	val, err := lookupVar(variable)
	if err != nil {
		return "", nil
	}
	if val == "" {
		return "", fmt.Errorf("Environment variable %v is empty", variable)
	}
	return val, nil
}

func loadConfigFromEnv() (*config, error) {
	var conf *config = &config{}

	if location, err := lookupVar(SURV_DATE_SAVE_LOCATION); err != nil {
		fmt.Printf("%v, using \"Local\" as default", err);
		conf.DateSaveLocation = "Local"
	}else {
		conf.DateSaveLocation = location
	}

	if creds, err := getVar(GOOGLE_CREDENTIALS_JSON); err != nil {
		return nil, err
	}else{
		conf.GoogleCredsJson = creds
	}

	if sheetName, err := getVar(GOOGLE_SHEET_NAME); err != nil {
		return nil, err
	}else{
		conf.GoogleSheetName = sheetName
	}

	if sheetId, err := getVar(GOOGLE_SPREADSHEET_ID); err != nil {
		return nil, err
	}else{
		conf.GoogleSheetId = sheetId
	}

	if tg, err := getVar(TELEGRAM_BOT_TOKEN); err != nil {
		conf.TgToken = &tg
	}

	if vk, err := getVar(VK_BOT_TOKEN); err != nil {
		conf.VkToken = &vk
	}

	if conf.VkToken == nil && conf.TgToken == nil{
		return nil, fmt.Errorf("Cannot read config, both %v and %v environment variables are emtpy", VK_BOT_TOKEN, TELEGRAM_BOT_TOKEN)
	}
	
	return conf, nil
}

func main() {
	config, err := loadConfigFromEnv()
	if err != nil {
		log.Fatalf("failed to initialize config: %v", err.Error())
	}

	db, err := survey.NewSuveyDB([]byte(config.GoogleCredsJson), config.GoogleSheetId, config.GoogleSheetName, config.DateSaveLocation)
	if err != nil {
		panic(err)
	}
	survey := survey.NewSurveyFabric(db)

	manager := conversation.NewManager(survey.NewStartQuestion)

	// register tg bot agent
	if config.TgToken != nil{
		tglogger := log.New(log.Writer(), "tgbot: ", log.LstdFlags&log.Lshortfile)
		tgbot, err := tg.NewBot(*config.TgToken, tglogger)
		if err != nil {
			panic(err)
		}
		manager.AddAgent(tgbot)
	}

	// register vk bot agent
	if config.VkToken != nil {
		vklogger := log.New(log.Writer(), "vkbot: ", log.LstdFlags&log.Lshortfile)
		vkbot, err := vk.NewBot(*config.VkToken, vklogger)
		if err != nil {
			panic(err)
		}
		manager.AddAgent(vkbot)
	}

	log.Println(manager.Run().Error())
}
