//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Izone-hub/talent-backend/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := pgxpool.New(context.Background(), cfg.GetDatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	f, err := os.Create("populate_output.txt")
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	// Get tags mapping name -> id
	tagMap := make(map[string]string)
	rows, err := db.Query(ctx, "SELECT id, name FROM tags")
	if err != nil {
		fmt.Fprintf(f, "Failed to query tags: %v\n", err)
		return
	}
	for rows.Next() {
		var id, name string
		rows.Scan(&id, &name)
		tagMap[strings.ToLower(name)] = id
	}
	rows.Close()

	fmt.Fprintf(f, "Tags loaded: %v\n\n", tagMap)

	// Get all questions
	qRows, err := db.Query(ctx, "SELECT id, question_text FROM questions")
	if err != nil {
		fmt.Fprintf(f, "Failed to query questions: %v\n", err)
		return
	}
	defer qRows.Close()

	type QInfo struct {
		ID   string
		Text string
	}
	var questions []QInfo
	for qRows.Next() {
		var q QInfo
		qRows.Scan(&q.ID, &q.Text)
		questions = append(questions, q)
	}

	// Insert question_tags mappings
	insertedCount := 0
	for _, q := range questions {
		txt := strings.ToLower(q.Text)
		var matchedTags []string

		// Smart matching based on question content
		if strings.Contains(txt, "javascript") || strings.Contains(txt, "json") || strings.Contains(txt, "css") {
			if id, ok := tagMap["javascript"]; ok {
				matchedTags = append(matchedTags, id)
			}
		}
		if strings.Contains(txt, "react") {
			if id, ok := tagMap["react"]; ok {
				matchedTags = append(matchedTags, id)
			}
		}
		if strings.Contains(txt, "python") || strings.Contains(txt, "factorial") {
			if id, ok := tagMap["python"]; ok {
				matchedTags = append(matchedTags, id)
			}
		}
		if strings.Contains(txt, "query") || strings.Contains(txt, "database") || strings.Contains(txt, "sql") || strings.Contains(txt, "postgres") {
			if id, ok := tagMap["postgresql"]; ok {
				matchedTags = append(matchedTags, id)
			}
		}
		if strings.Contains(txt, "docker") || strings.Contains(txt, "container") {
			if id, ok := tagMap["docker"]; ok {
				matchedTags = append(matchedTags, id)
			}
		}

		// Fallback for general questions (like binary search, FIFO) to make sure they have tags
		if len(matchedTags) == 0 {
			// Assign to JavaScript and Python as general questions
			if id, ok := tagMap["javascript"]; ok {
				matchedTags = append(matchedTags, id)
			}
			if id, ok := tagMap["python"]; ok {
				matchedTags = append(matchedTags, id)
			}
		}

		for _, tagID := range matchedTags {
			_, err := db.Exec(ctx, "INSERT INTO question_tags (question_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", q.ID, tagID)
			if err != nil {
				fmt.Fprintf(f, "Error inserting mapping: q=%s t=%s error=%v\n", q.ID, tagID, err)
			} else {
				fmt.Fprintf(f, "Linked question %s (%s...) to tag %s\n", q.ID, q.Text[:20], tagID)
				insertedCount++
			}
		}
	}

	fmt.Fprintf(f, "\nSuccessfully inserted/verified %d mappings in question_tags.\n", insertedCount)
}
