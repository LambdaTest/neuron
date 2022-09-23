package db

import (
	"fmt"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// Connect create connection with database
func Connect(cfg *config.Config, logger lumber.Logger) (core.DB, error) {
	connectionString := fmt.Sprintf("%s:%s@%s(%s:%s)/%s", cfg.DB.User, cfg.DB.Password, "tcp", cfg.DB.Host, cfg.DB.Port, cfg.DB.Name)
	db, err := sqlx.Connect("mysql", connectionString+"?parseTime=true&charset=utf8mb4")
	if err != nil {
		return nil, err
	}
	logger.Infof("Database connected successfully")

	db.SetMaxIdleConns(constants.MysqlMaxIdleConnection)
	db.SetMaxOpenConns(constants.MysqlMaxOpenConnection)
	db.SetConnMaxLifetime(constants.MysqlMaxConnectionLifetime)

	return &DB{conn: db, logger: logger}, nil
}
