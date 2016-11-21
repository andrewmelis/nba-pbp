package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/pbp/", pbpHandler)

	log.Fatal(http.ListenAndServe(":8081", nil))
}

func pbpHandler(w http.ResponseWriter, r *http.Request) {
	requestedGameCode := r.URL.Path[len("/pbp/"):]
	log.Printf("requested %s pbp\n", requestedGameCode)

	// ideally -- game from gamecode?
	// PbpFromGameCode ?

	games, err := getGames()

	if err != nil {
		log.Printf("error retrieving games: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error":"server error occurred"}`)
		return
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(&games)
	if err != nil {
		log.Printf("error encoding games: %s\n", err)
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

func getPlayByPlayFromGameId(g Game) (PlayByPlayGame, error) {
	todayPbpUrl, err := TodayPbpURL()
	if err != nil {
		log.Printf("error retrieving today url: %s\n", err)
		return PlayByPlayGame{Game: game}, err
	}

	fmt.Printf("%s\n", todayPbpUrl)

	return PlayByPlayGame{Game: game}, nil

	// how to parse handlebars in linkrel?

}

type NBATodayResponse struct {
	Links map[string]string `json:"links"`
}

const NBABaseURL = "http://data.nba.net"
const NBATodayRoute = "/10s/prod/v1/today.json"

func TodayPbpURL() (string, error) {
	NBATodayPbpURL := fmt.Sprintf("%s%s", NBABaseURL, todayResp.Links["pbp"])
	return NBATodayPbpURL, nil
}

func NBAToday() (NBATodayResponse, err) {
	NBATodayURL := fmt.Sprintf("%s%s", NBABaseURL, NBATodayRoute)

	resp, err := http.Get(NBATodayURL)
	if err != nil {
		log.Printf("error retrieving today url: %s\n", err)
		return "", err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var todayResp NBATodayResponse

	for dec.More() {
		err := dec.Decode(&todayResp)
		if err != nil {
			log.Printf("error decoding response: %s\n", err)
			return "", err
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

func (gs Games) FindByGameCode(gameCode) (Game, error) {
	for _, game := range gs.Games {
		if game.GameCode() == gameCode {
			return game, nil
		}
	}
	return "", errors.New(fmt.Sprintf("Game %s not scheduled for today!", gameCode))
}

type Game struct {
	Id           string    `json:"gameId"`
	StartTime    time.Time `json:"startTimeUTC"`
	VisitingTeam Team      `json:"vTeam"`
	HomeTeam     Team      `json:"hTeam"`
	Period       Period    `json:"period"`
}

func (g Game) GameCode() string {
	fmt.Sprintf("%s%s", g.VisitingTeam.TriCode, g.HomeTeam.TriCode)
}

type Team struct {
	Id      string `json:"teamId"`
	TriCode string `json:"triCode"`
}

type Period struct {
	Current int
}
