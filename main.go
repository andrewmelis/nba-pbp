package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"
)

func main() {
	http.HandleFunc("/pbp/", pbpHandler)

	log.Fatal(http.ListenAndServe(":8081", nil))
}

func pbpHandler(w http.ResponseWriter, r *http.Request) {
	requestedGameCode := r.URL.Path[len("/pbp/"):]
	log.Printf("requested %s pbp\n", requestedGameCode)

	pbpGame, err := getPlayByPlayFromGameCode(requestedGameCode)

	if err != nil {
		log.Printf("error retrieving game: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error":"server error occurred"}`)
		return
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(&pbpGame)
	if err != nil {
		log.Printf("error encoding game: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error":"server error occurred"}`)
		return
	}
}

type PlayByPlayGame struct {
	Game
	Foo string
	// Plays []Play
}

func getPlayByPlayFromGameCode(gameCode string) (PlayByPlayGame, error) {
	games, err := getGames()
	if err != nil {
		log.Printf("error retrieving games: %s\n", err)
		return PlayByPlayGame{}, err
	}

	// hide everything above this?
	game, err := games.FindByGameCode(gameCode)
	if err != nil {
		log.Printf("error finding game %s\n", err)
		return PlayByPlayGame{}, err
	}

	pbpGame, err := getPlayByPlayFromGame(game)
	if err != nil {
		log.Printf("error retrieving pbp: %s\n", err)
		return PlayByPlayGame{}, err
	}

	return pbpGame, nil
}

func getPlayByPlayFromGame(g Game) (PlayByPlayGame, error) {
	// HACK: today url is "/data/10s/prod/v1/{{gameDate}}/{{gameId}}_pbp_{{periodNum}}.json"
	// figure out good way to replace in handlebars template

	t := template.New("nba pbp url")
	t, err := t.Parse("{{.BaseUrl}}/data/10s/prod/v1/{{.GameDate}}/{{.Id}}_pbp_{{.Period.Current}}.json")
	if err != nil {
		log.Printf("error parsing template: %s\n", err)
		return PlayByPlayGame{}, err
	}

	t.Execute(os.Stdout, struct {
		BaseUrl string
		Game
	}{
		NBABaseURL,
		g,
	})

	return PlayByPlayGame{Game: g}, nil

}

type NBATodayResponse struct {
	Links map[string]string `json:"links"`
}

const NBABaseURL = "http://data.nba.net"
const NBATodayRoute = "/10s/prod/v1/today.json"

func TodayPbpURL() (string, error) {
	todayResp, err := NBAToday()
	if err != nil {
		log.Printf("error getting today response: %s\n", err)
		return "", err
	}
	NBATodayPbpURL := fmt.Sprintf("%s%s", NBABaseURL, todayResp.Links["pbp"])
	return NBATodayPbpURL, nil
}

func NBAToday() (NBATodayResponse, error) {
	NBATodayURL := fmt.Sprintf("%s%s", NBABaseURL, NBATodayRoute)

	resp, err := http.Get(NBATodayURL)
	if err != nil {
		log.Printf("error retrieving today url: %s\n", err)
		return NBATodayResponse{}, err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var todayResp NBATodayResponse

	for dec.More() {
		err := dec.Decode(&todayResp)
		if err != nil {
			log.Printf("error decoding response: %s\n", err)
			return NBATodayResponse{}, err
		}
	}
	return todayResp, nil
}

type Games struct {
	Games []Game `json:"games"`
}

func getGames() (Games, error) {
	resp, err := http.Get("http://localhost:8080/games")
	if err != nil {
		log.Printf("error retrieving games %s\n", err)
		return Games{}, err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var games Games
	for dec.More() {
		err := dec.Decode(&games)
		if err != nil {
			log.Printf("error decoding games %s\n", err)
			return Games{}, err
		}
	}

	return games, nil
}

func (gs Games) FindByGameCode(gameCode string) (Game, error) {
	for _, game := range gs.Games {
		if game.GameCode() == gameCode {
			return game, nil
		}
	}
	return Game{}, errors.New(fmt.Sprintf("Game %s not scheduled for today!", gameCode))
}

type Game struct {
	Id           string    `json:"gameId"`
	StartTime    time.Time `json:"startTimeUTC"`
	VisitingTeam Team      `json:"vTeam"`
	HomeTeam     Team      `json:"hTeam"`
	Period       Period    `json:"period"`
}

func (g Game) GameCode() string {
	return fmt.Sprintf("%s%s", g.VisitingTeam.TriCode, g.HomeTeam.TriCode)
}

// GameDate returns the start date of game (YYYYMMDD format) in US/Eastern tz
// TODO: make sure output is in eastern
func (g Game) GameDate() string {
	easternTime, err := time.LoadLocation("America/New_York")
	if err != nil {
		os.Exit(1)
	}
	return g.StartTime.In(easternTime).Format("20060102")
}

type Team struct {
	Id      string `json:"teamId"`
	TriCode string `json:"triCode"`
}

type Period struct {
	Current int
}
