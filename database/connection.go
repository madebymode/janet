package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// V2DB represents the new unified database with enhanced features
type V2DB struct {
	Config *Config
	SQL    *sql.DB
}

// NewV2 creates a new V2 database instance
func NewV2(config *Config) (*V2DB, error) {
	instance := &V2DB{
		Config: config,
	}

	err := instance.Init()
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// Init initializes the V2 database connection
func (db *V2DB) Init() error {
	conn, err := sql.Open(db.Config.Driver, db.Config.URL)
	if err != nil {
		return err
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(25)
	conn.SetConnMaxIdleTime(5 * time.Minute)
	conn.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return err
	}

	db.SQL = conn
	return nil
}

// Close closes the database connection
func (db *V2DB) Close() error {
	if db.SQL != nil {
		return db.SQL.Close()
	}
	return nil
}

// rebind converts ? placeholders to PostgreSQL $1, $2, etc. format
func rebind(query string) string {
	parts := strings.Split(query, "?")
	var result string
	for i := 0; i < len(parts)-1; i++ {
		result += parts[i] + fmt.Sprintf("$%d", i+1)
	}
	result += parts[len(parts)-1]
	return result
}

// buildWhereClause helps build dynamic WHERE clauses with proper parameter indexing
func buildWhereClause(baseClauses []string, additionalClauses map[string]interface{}, startIndex int) (string, []interface{}) {
	clauses := make([]string, len(baseClauses))
	copy(clauses, baseClauses)

	var args []interface{}
	paramIndex := startIndex

	for field, value := range additionalClauses {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", field, paramIndex))
		args = append(args, value)
		paramIndex++
	}

	whereClause := ""
	if len(clauses) > 0 {
		whereClause = " WHERE " + strings.Join(clauses, " AND ")
	}

	return whereClause, args
}
