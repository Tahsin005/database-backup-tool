package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

type Postgres struct {
	Host     string
	Port     int
	Username string
	Password string
	DBName   string
}

func NewPostgres(host string, port int, username, password, dbName string) *Postgres {
	return &Postgres{host, port, username, password, dbName}
}

func (p *Postgres) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		p.Host, p.Port, p.Username, p.Password, p.DBName,
	)
}

func (p *Postgres) Ping() error {
	db, err := sql.Open("postgres", p.DSN())
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping: %w", err)
	}
	return nil
}