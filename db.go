package main

import (
	"context"
	"log"
	"time"
	_ "time/tzdata"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type SurveyDB struct {
	list          string
	spreadsheetId string
	srv           *sheets.Service
	location      *time.Location
}

func newSuveyDB(credentialsFile string, spreadsheetId string, list string, location string) *SurveyDB {
	loc, err := time.LoadLocation(location);
	if err != nil {
		loc, err = time.LoadLocation("Local")
		if err != nil {
			return nil
		}
	}
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
		location:      loc,
	}
}

func (db *SurveyDB) WriteAnswers(ID string, time time.Time, name interface{}, age interface{}, city interface{}, request interface{}, health interface{}, contact interface{}) error {
	range_ := db.list + "!A:A"
	valuerange := sheets.ValueRange{

		Values: [][]interface{}{
			{
				ID,
				time.UTC().In(db.location).Format("02/01/2006 15:04:05") + " " + db.location.String(),
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
