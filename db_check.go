//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

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

	f, err := os.Create("db_output.txt")
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	rows, err := db.Query(ctx, "SELECT id, question_text, question_type, tags FROM questions")
	if err != nil {
		fmt.Fprintf(f, "Query error: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, text, qtype string
		var tags []string
		rows.Scan(&id, &text, &qtype, &tags)
		fmt.Fprintf(f, "ID: %s\nType: %s\nText: %s\nTags: %v\n\n", id, qtype, text, tags)
	}
}
