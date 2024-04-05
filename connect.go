package main

import (
	"fmt"
	"os"
	"go.uber.org/zap"
	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var Conn *sqlx.DB

func ConnectDB() {
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")

	connStr := fmt.Sprintf("postgresql://%s:%s@%s:5432/%s", dbUser, dbPassword, dbHost, dbName)

	var err error
	Conn, err = sqlx.Connect("pgx", connStr)
	if err != nil {
		logger.Fatal("Невозможно подключиться к БД: ", zap.Error(err))
	}
}
