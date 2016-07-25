// loadexcuses.go
// Bung some excuses into the infobot database

package main

import (
	"bufio"
	"database/sql"
	_ "github.com/lib/pq"
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("Reading excuses file into excuses table in minerva db")
	// open infobot db 'minerva'
	db, err := sql.Open("postgres", "user=sue dbname=minerva sslmode=disable")
	if err != nil {
		log.Fatal("Failed to open the database ", err)
	}
	// open file 'excuses'
	file, err := os.Open("excuses")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		log.Println(line)
		_, err := db.Exec("INSERT INTO excuses(\"excuse\") VALUES ($1)", line)
		if err != nil {
			log.Fatal("Failed to insert excuse row: ", err)
		}
	}
}
