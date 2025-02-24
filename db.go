package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	tok := getTokenFromWeb(config)
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

type SurveyDB struct{
  list string
  client *http.Client
  spreadsheetId string
  srv *sheets.Service
}

func newSuveyDB(credentialsFile string, spreadsheetId string,list string) *SurveyDB {
	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}
  return &SurveyDB{
    list: list,
    client: client,
    spreadsheetId: spreadsheetId,
    srv: srv,
  }
}


func (db *SurveyDB) WriteAnswers(ID string, time time.Time, name interface{}, age interface{}, city interface{}, request interface{}, contact interface{}) error{
	range_ := db.list + "!B:B"
	valuerange := sheets.ValueRange{
		Values: [][]interface{}{
      {
        ID, time.UTC(), name, age, city, request, contact,
      },
    },
	}

	db.srv.Spreadsheets.Values.Append(db.spreadsheetId, range_,&valuerange)

  return nil
}
