package db

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var DB *sql.DB

// 1) Parse your DB_DSN (must be in URL form: postgres://user:pass@host:port/dbname?…)
// 2) Connect to the “postgres” DB as admin to CREATE DATABASE if it doesn't exist
// 3) Re-open a connection to your target DB
// 4) Read & exec internal/db/init.sql to CREATE TABLE IF NOT EXISTS …
func Init() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on environment variables")
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN environment variable is required")
	}

	u, err := url.Parse(dsn)
	if err != nil {
		log.Fatalf("invalid DB_DSN: %v", err)
	}
	targetDB := strings.TrimPrefix(u.Path, "/")

	u.Path = "/postgres"
	adminDSN := u.String()

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		log.Fatalf("cannot open admin connection: %v", err)
	}
	defer adminDB.Close()

	var exists bool
	err = adminDB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)",
		targetDB,
	).Scan(&exists)
	if err != nil {
		log.Fatalf("checking for database existence failed: %v", err)
	}

	if !exists {
		_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", pqQuoteIdentifier(targetDB)))
		if err != nil {
			log.Fatalf("could not create database %q: %v", targetDB, err)
		}
		log.Printf("created database %q", targetDB)
	}

	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("cannot open target database connection: %v", err)
	}
	dbConn.SetMaxOpenConns(10)
	dbConn.SetMaxIdleConns(2)

	if err := dbConn.Ping(); err != nil {
		log.Fatalf("cannot ping target database: %v", err)
	}

	ddl, err := os.ReadFile("internal/db/init.sql")
	if err != nil {
		log.Fatalf("cannot read init.sql: %v", err)
	}
	if _, err := dbConn.Exec(string(ddl)); err != nil {
		log.Fatalf("error executing init.sql: %v", err)
	}

	DB = dbConn
	log.Println("database initialized and tables ensured")
}

func pqQuoteIdentifier(id string) string {
	return `"` + strings.ReplaceAll(id, `"`, `""`) + `"`
}
