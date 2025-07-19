package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if os.Getenv("RUN_MIGRATIONS") != "true" {
		log.Println("RUN_MIGRATIONS environment variable not set to 'true'. Skipping migrations.")
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		log.Fatal("Usage: migrate <project_root_path>")
	}

	projectRoot := os.Args[1]
	postgresURL := os.Getenv("JANET_DATABASE_URL")
	if postgresURL == "" {
		log.Fatal("JANET_DATABASE_URL environment variable not set.")
	}

	// Connect to Postgres
	pgDB, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatal("Failed to connect to Postgres:", err)
	}
	defer pgDB.Close()

	// Read and execute schema
	schemaPath := filepath.Join(projectRoot, "database", "migration.sql")
	schema, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		log.Fatal("Failed to read schema file:", err)
	}

	log.Println("Creating new database schema in Postgres...")
	if _, err := pgDB.Exec(string(schema)); err != nil {
		log.Fatal("Failed to create schema:", err)
	}

	// Clear existing data before migration
	log.Println("Clearing existing data in karma_transactions...")
	if _, err := pgDB.Exec(`DELETE FROM karma_transactions`); err != nil {
		log.Fatalf("Failed to clear existing data: %v", err)
	}

	// Migrate yearly databases
	databases := []string{
		filepath.Join(projectRoot, "2022.db.sqlite3"),
		filepath.Join(projectRoot, "2023.db.sqlite3"),
		filepath.Join(projectRoot, "2024.db.sqlite3"),
		filepath.Join(projectRoot, "2025.db.sqlite3"),
	}

	totalRecords := 0
	totalSkipped := 0
	for _, dbPath := range databases {
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			continue
		}

		log.Printf("Migrating data from: %s", dbPath)
		migrated, skipped, err := migrateDatabase(dbPath, pgDB)
		if err != nil {
			log.Printf("Error migrating %s: %v", dbPath, err)
			continue
		}
		totalRecords += migrated
		totalSkipped += skipped
		log.Printf("Migrated %d records from %s (skipped %d)", migrated, dbPath, skipped)
	}

	log.Printf("Migration completed! Total records migrated: %d, skipped: %d", totalRecords, totalSkipped)
}

func migrateDatabase(dbPath string, pgDB *sql.DB) (int, int, error) {
	sourceDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return 0, 0, err
	}
	defer sourceDB.Close()

	rows, err := sourceDB.Query(`
        SELECT id, "from", "to", points, reason, timestamp
        FROM karma
        ORDER BY timestamp
    `)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	basename := filepath.Base(dbPath)
	yearStr := basename[:4]
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year in database path: %s", yearStr)
	}

	tx, err := pgDB.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	insertStmt, err := tx.Prepare(`
        INSERT INTO karma_transactions
        (from_user, to_user, points, reason, transaction_type, emoji_name, timestamp, year)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `)
	if err != nil {
		return 0, 0, err
	}
	defer insertStmt.Close()

	recordCount := 0
	skipped := 0
	emojiRegex := regexp.MustCompile(`(?i)added a\s+:([a-zA-Z0-9_+-]+):\s+emoji`)

	for rows.Next() {
		var id int
		var fromUser, toUser, reason, timestampStr string
		var points int

		if err := rows.Scan(&id, &fromUser, &toUser, &points, &reason, &timestampStr); err != nil {
			log.Printf("Error scanning row (id=%d): %v", id, err)
			skipped++
			continue
		}

		timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr)
		if err != nil {
			log.Printf("Invalid timestamp '%s' for id=%d: %v", timestampStr, id, err)
			skipped++
			continue
		}

		transactionType := "manual"
		var emojiName *string
		if matches := emojiRegex.FindStringSubmatch(reason); len(matches) > 1 {
			transactionType = "emoji"
			emojiName = &matches[1]
		}

		if _, err := insertStmt.Exec(fromUser, toUser, points, reason, transactionType, emojiName, timestamp, year); err != nil {
			log.Printf("Error inserting record id=%d: %v", id, err)
			skipped++
			continue
		}

		recordCount++
	}
	if err := rows.Err(); err != nil {
		return recordCount, skipped, fmt.Errorf("row iteration error: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return recordCount, skipped, err
	}

	return recordCount, skipped, nil
}
