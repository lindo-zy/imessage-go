package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strings"
	"time"
)

type Message struct {
	RowID         int
	Date          string
	Body          string
	PhoneNumber   string
	IsFromMe      bool
	CacheRoomname string
	GroupChatName string
}

func getChatMapping(dbLocation string) (map[string]string, error) {
	db, err := sql.Open("sqlite3", dbLocation)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT room_name, display_name FROM chat")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mapping := make(map[string]string)
	for rows.Next() {
		var roomName, displayName sql.NullString
		if err := rows.Scan(&roomName, &displayName); err != nil {
			return nil, err
		}

		// 如果 roomName 或 displayName 为 NULL，则跳过该行
		if !roomName.Valid || !displayName.Valid {
			continue
		}

		mapping[roomName.String] = displayName.String
	}

	return mapping, nil
}
func readMessages(dbLocation string, n int, selfNumber string, humanReadableDate bool) ([]Message, error) {
	db, err := sql.Open("sqlite3", dbLocation)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
	SELECT message.ROWID, message.date, message.text, message.attributedBody, handle.id, message.is_from_me, message.cache_roomnames
	FROM message
	LEFT JOIN handle ON message.handle_id = handle.ROWID
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
		var handleID sql.NullString
		var isFromMe bool
		var cacheRoomname sql.NullString

		if err := rows.Scan(&rowid, &date, &text, &attributedBody, &handleID, &isFromMe, &cacheRoomname); err != nil {
			return nil, err
		}

		phoneNumber := selfNumber
		if handleID.Valid {
			phoneNumber = handleID.String
		}

		body := text.String
		if !text.Valid && attributedBody.Valid {
			body = attributedBody.String
			// 这里可以添加对attributedBody的进一步处理
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

		if humanReadableDate {
			dateString := "2001-01-01"
			modDate, _ := time.Parse("2006-01-02", dateString)
			unixTimestamp := modDate.UnixNano()
			newDate := (date + unixTimestamp) / 1e9
			dateStr := time.Unix(newDate, 0).Format("2006-01-02 15:04:05")
			date = newDate
			body = fmt.Sprintf("%s: %s", dateStr, body)
		}

		mapping, err := getChatMapping(dbLocation)
		if err != nil {
			return nil, err
		}

		var groupChatName string
		if cacheRoomname.Valid {
			groupChatName = mapping[cacheRoomname.String]
		}

		messages = append(messages, Message{
			RowID:         rowid,
			Date:          time.Unix(date, 0).Format("2006-01-02 15:04:05"),
			Body:          body,
			PhoneNumber:   phoneNumber,
			IsFromMe:      isFromMe,
			CacheRoomname: cacheRoomname.String,
			GroupChatName: groupChatName,
		})
	}

	return messages, nil
}

func printMessages(messages []Message) {
	for _, message := range messages {
		fmt.Printf("RowID: %d\n", message.RowID)
		fmt.Printf("Body: %s\n", message.Body)
		fmt.Printf("Phone Number: %s\n", message.PhoneNumber)
		fmt.Printf("Is From Me: %t\n", message.IsFromMe)
		fmt.Printf("Cache Roomname: %s\n", message.CacheRoomname)
		fmt.Printf("Group Chat Name: %s\n", message.GroupChatName)
		fmt.Printf("Date: %s\n", message.Date)
		fmt.Println()
	}
}

func main() {
	dbLocation := "/Users/xiao/Downloads/chat.db"
	messages, err := readMessages(dbLocation, 10, "Me", true)
	if err != nil {
		log.Fatal(err)
	}
	printMessages(messages)
}
