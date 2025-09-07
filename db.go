package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SurveyDB struct {
	list          string
	spreadsheetId string
	srv           *sheets.Service
}

func newSuveyDB(credentialsFile string, spreadsheetId string, list string) *SurveyDB {
	ctx := context.Background()
	// client := conf.Client(ctx)

	srv, err := sheets.NewService(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}
	return &SurveyDB{
		list:          list,
		spreadsheetId: spreadsheetId,
		srv:           srv,
	}
}

func (db *SurveyDB) WriteAnswers(ID string, time time.Time, name interface{}, age interface{}, city interface{}, request interface{}, health interface{}, contact interface{}) error {
	range_ := db.list + "!A:A"
	valuerange := sheets.ValueRange{

		Values: [][]interface{}{
			{
				ID,
				time.UTC(),
				name,
				age,
				city,
				request,
				health,
				contact,
			},
		},
	}

	_, err := db.srv.Spreadsheets.Values.Append(db.spreadsheetId, range_, &valuerange).ValueInputOption("RAW").Do()

	return err
}
