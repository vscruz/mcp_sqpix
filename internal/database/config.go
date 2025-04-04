package database

import (
	"database/sql"
	"fmt"

	_ "github.com/denisenkom/go-mssqldb"
)

type DBConfig struct {
	Server   string
	Port     int
	User     string
	Password string
	Database string
}

// NewConnection cria uma nova conexão com o banco de dados SQL Server
func NewConnection(config DBConfig) (*sql.DB, error) {
	connectionString := fmt.Sprintf("server=%s;port=%d;user id=%s;password=%s;database=%s",
		config.Server, config.Port, config.User, config.Password, config.Database)

	// Abre uma conexão com o banco de dados
	db, err := sql.Open("mssql", connectionString)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar ao banco de dados: %v", err)
	}

	// Testa a conexão
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar conexão com banco de dados: %v", err)
	}

	return db, nil
}
