/*

mybot - Illustrative Slack bot in Go

Copyright (c) 2015 RapidLoop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package main

import (
	"database/sql"
	"encoding/csv"
	_ "github.com/lib/pq"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s slack-bot-token\n", os.Args[0])
		os.Exit(1)
	}
	token := os.Args[1]

	// start a websocket-based Real Time API session
	ws, id := slackConnect(token)
	log.Printf("%s ready\n", token) // change this to log entry

	// open infobot db 'minerva'
	db, err := sql.Open("postgres", "user=sue dbname=minerva sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to open the database ", err)
	}
	botname, err := getUserInfo(token, id)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Botname is", botname)

	for {
		m, err := getMessage(ws)
		if err != nil {
			log.Print(err)
			continue
		}

		username, err := getUserInfo(token, m.User)
		// Look for karma changes (++/--) in the message
		handleKarma(m.Text, username, db)

		// Look for bot commands (prefaced by bot id)
		if m.Type == "message" && strings.HasPrefix(m.Text, "<@"+id+">") {
			args := strings.Fields(m.Text)
			// Pick out named commands
			switch args[1] {
			case "stock":
				if len(args) >= 3 {
					m.Text = getQuote(args[2])
				} else {
					m.Text = "You need to supply a stock ticker symbol (e.g. CSCO)."
				}
			case "karma":
				// Get the karma value for args[2]
				if len(args) >= 3 {
					m.Text = getKarma(args[2], db)
				} else {
					m.Text = "No nick, no karma."
				}
			case "summon":
				if len(args[2]) > 0 {
					phrase := strings.Join(args[2:], " ")
					m.Text = strings.ToUpper(fmt.Sprintf("%s %s %s COME TO ME!",
						phrase, phrase, phrase))
				} else {
					m.Text = "You have to tell me whom to summon!"
				}
			case "excuse":
				m.Text = "Your excuse is " + getExcuse(db)
			default:
				m.Text = "Huh?"
			}
			postMessage(ws, m)
		}
	}
}

// Get the quote via Yahoo.
func getQuote(sym string) string {
	sym = strings.ToUpper(sym)
	url := fmt.Sprintf("http://download.finance.yahoo.com/d/quotes.csv?s=%s&f=nsl1op&e=.csv", sym)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	rows, err := csv.NewReader(resp.Body).ReadAll()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	if len(rows) >= 1 && len(rows[0]) == 5 {
		return fmt.Sprintf("%s (%s) is trading at $%s", rows[0][0], rows[0][1], rows[0][2])
	}
	return fmt.Sprintf("Unknown response format (symbol was \"%s\")", sym)
}

// Get the karma value for nick from the database.
func getKarma(nick string, db *sql.DB) string {
	var karma int
	rows, err := db.Query("SELECT SUM(delta) FROM karma WHERE nick = $1", nick)
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
	}
	karmaStr := fmt.Sprintf("%s has no karma.", nick)
	if rows.Next() {
		rows.Scan(&karma)
		karmaStr = fmt.Sprintf("Karma for %s is %d.", nick, karma)
	}
	return karmaStr
}

// Check for karma change. Update if so.
func handleKarma(msg, by string, db *sql.DB) {
	index := strings.LastIndex(msg, "#")
	reason := ""
	if index != -1 {
		reason = msg[index+1: len(msg)]
	}
	f := strings.Fields(msg)
	for _, v := range f {
		var nick string
		if strings.HasSuffix(v, "++") {
			nick = strings.TrimSuffix(v, "++")
			updateKarma(db, by, nick, reason, 1)
		} else if strings.HasSuffix(v, "--") {
			nick = strings.TrimSuffix(v, "--")
			updateKarma(db, by, nick, reason, -1)
		}
	}
}

func updateKarma(db *sql.DB, by, nick string, reason string, delta int) {
	if delta != -1 && delta != 1 {
		log.Println("Karma delta must be 1 or -1 - ignoring.")
		return
	}
	_, err := db.Exec("INSERT INTO karma(\"nick\", \"delta\", \"by\", \"reason\") VALUES ($1, $2, $3, $4)",
		nick, delta, by, reason)
	if err != nil {
		log.Printf("Failed to insert karma entry - %s", err)
	}
}

func getExcuse(db *sql.DB) string {
	rows, err := db.Query("SELECT excuse FROM excuses ORDER BY random() LIMIT 1")
	if err != nil {
		log.Printf("Failed to get an excuse from the database:", err)
	}
	defer rows.Close()
	excuse := "positron router malfunction"
	if rows.Next() {
		rows.Scan(&excuse)
	}
	return excuse
}
