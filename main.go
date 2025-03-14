package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os/exec"
	"os/user"
	"regexp"
	"strings"
	"time"
)

type Message struct {
	RowID int
	Date  string
	Body  string
}

func readMessages(dbLocation string, n int) ([]Message, error) {
	db, err := sql.Open("sqlite3", dbLocation)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
	SELECT 
	    message.ROWID, 
	    message.date, 
	    message.text, 
	    message.attributedBody
	FROM message
	`
	if n > 0 {
		query += fmt.Sprintf(" ORDER BY message.date DESC LIMIT %d", n)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var rowid int
		var date int64
		var text, attributedBody sql.NullString

		if err := rows.Scan(&rowid, &date, &text, &attributedBody); err != nil {
			return nil, err
		}

		body := text.String
		if !text.Valid && attributedBody.Valid {
			body = attributedBody.String
			if strings.Contains(body, "NSNumber") {
				body = strings.Split(body, "NSNumber")[0]
				if strings.Contains(body, "NSString") {
					body = strings.Split(body, "NSString")[1]
					if strings.Contains(body, "NSDictionary") {
						body = strings.Split(body, "NSDictionary")[0]
						if len(body) >= 18 {
							body = body[6 : len(body)-12]
						}
					}
				}
			}
		}

		dateString := "2001-01-01"
		modDate, _ := time.Parse("2006-01-02", dateString)
		unixTimestamp := modDate.UnixNano()
		newDate := (date + unixTimestamp) / 1e9
		//dateStr := time.Unix(newDate, 0).Format("2006-01-02 15:04:05")
		date = newDate
		//body = fmt.Sprintf("%s: %s", dateStr, body)

		messages = append(messages, Message{
			RowID: rowid,
			Date:  time.Unix(date, 0).Format("2006-01-02 15:04:05"),
			Body:  body,
		})
	}

	return messages, nil
}

func getVerifyCode(message Message) string {
	re := regexp.MustCompile(`\d{4,6}`)

	matches := re.FindAllString(message.Body, -1)

	for _, match := range matches {
		return match
	}
	return ""
}
func printMessages(messages []Message) {
	for _, message := range messages {
		fmt.Printf("Body: %s\n", message.Body)
		fmt.Printf("Date: %s\n", message.Date)
		fmt.Println()
	}
}

func main() {
	homeDir, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	dbLocation := homeDir.HomeDir + "/Library/Messages/chat.db"
	messages, err := readMessages(dbLocation, 1)
	if err != nil {
		log.Fatal(err)
	}
	//printMessages(messages)
	text := getVerifyCode(messages[0])
	cmd := exec.Command("pbcopy")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Println(err)
		return
	}

	_, err = stdin.Write([]byte(text))
	if err != nil {
		fmt.Println(err)
		return
	}

	stdin.Close()

	if err := cmd.Wait(); err != nil {
		fmt.Println(err)
		return
	}
}
