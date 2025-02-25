package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)


type jsondata struct{
  Email string `json:"email"`
  PrivateKey string `json:"private_key"`
  PrivateKeyID string `json:"private_key_id"`
  TokenURL string `json:"token_url"`
  Scopes []string `json:"scopes"`
}
func getConfigFromFile(configFile string) (*jwt.Config, error) {
  data, err :=os.ReadFile(configFile)
  if err != nil{
    return nil, err
  }
  j := &jsondata{}
  err = json.Unmarshal(data, j)
  if err != nil{
    return nil, err
  }
  conf := &jwt.Config{
    Email: j.Email,
    PrivateKey: []byte(j.PrivateKey),
    PrivateKeyID: j.PrivateKeyID,
    TokenURL: j.TokenURL,
    Scopes: j.Scopes,
  }
  return conf, err
}


type SurveyDB struct{
  list string
  client *http.Client
  spreadsheetId string
  srv *sheets.Service
}

func newSuveyDB(credentialsFile string, spreadsheetId string,list string) *SurveyDB {
  conf, err := getConfigFromFile(credentialsFile)
  if err != nil{
    log.Fatalf("Unable to read credentials: %v", err)
  }
	client := conf.Client(context.TODO())

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
