package db

import (
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/Tahsin005/database-backup-tool/internal/config"
	_ "github.com/lib/pq"
)

type Postgres struct {
	Host     string
	Port     int
	Username string
	Password string
	DBName   string
	SSLMode  string
}

// creates a new postgres instance with connection parameters
func NewPostgres(host string, port int, username, password, dbName, sslMode string) *Postgres {
	if sslMode == "" {
		sslMode = "disable"
	}
	return &Postgres{host, port, username, password, dbName, sslMode}
}

// creates a new postgres instance from a db connection config
func NewPostgresFromConfig(cfg config.DBConnConfig) *Postgres {
	return NewPostgres(cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DBName, cfg.SSLMode)
}

// returns a safely url-encoded connection string with ssl mode
func (p *Postgres) DSN() string {
	sslMode := p.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(p.Username, p.Password),
		Host:   net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
		Path:   "/" + p.DBName,
	}
	q := u.Query()
	q.Set("sslmode", sslMode)
	u.RawQuery = q.Encode()

	return u.String()
}

// verifies connectivity to the postgres instance
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