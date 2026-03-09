package gamepass

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	allList    = "https://catalog.gamepass.com/sigls/v2?id=fdd9e2a7-0fee-49f6-ad69-4354098401ff&language=en-us&market=US"
	detailList = "https://displaycatalog.mp.microsoft.com/v7.0/products?bigIds=%v&market=US&languages=en-us&MS-CV=DGU1mcuYo0WMMp+F.1"
)

type Gameid struct {
	ID string `json:"id"`
}

type games struct {
	LocalizedProperties []struct {
		DeveloperName string `json:"DeveloperName"`
		PublisherName string `json:"PublisherName"`
		// PublisherWebsiteUri string `json:"PublisherWebsiteUri"`
		// SupportUri          string `json:"SupportUri"`
		Images []struct {
			// 	FileId string `json:"FileId"`
			Uri string `json:"Uri"`
		} `json:"Images"`
		ProductDescription string `json:"ProductDescription"`
		ProductTitle       string `json:"ProductTitle"`
		ShortTitle         string `json:"ShortTitle"`
		// SortTitle          string      `json:"SortTitle"`
		// FriendlyTitle      interface{} `json:"FriendlyTitle"`
		ShortDescription string `json:"ShortDescription"`
	} `json:"LocalizedProperties"`
}

type AllGames struct {
	Products []games
}

func (app *GamepassApplication) GetFromAPI() error {
	var list []Gameid
	resp, err := app.HttpClient.Get(allList)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	listBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(listBody, &list)
	if err != nil {
		return err
	}
	var listIDs []string
	for _, val := range list[1:] {
		listIDs = append(listIDs, val.ID)
	}
	d := strings.Join(listIDs, ",")
	url := fmt.Sprintf(detailList, d)
	resp2, err := app.HttpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp2.Body)
	if err != nil {
		return err
	}
	// fmt.Println(string(b))
	// os.WriteFile("allGames.txt", b, 0644)
	var final AllGames
	err = json.Unmarshal(b, &final)
	if err != nil {
		return err
	}
	app.AllGames = &final
	newData, err := json.MarshalIndent(final, "", "\t")
	if err != nil {
		return err
	}
	err = os.WriteFile("allGames4.json", newData, 0644)
	return err
	// fmt.Println(string(newData))
}
