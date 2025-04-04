package main

import (
	"flag"
	"log"
	"os"
	"sq_pix/internal/database"
	"sq_pix/internal/esptag"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

func main() {
	// Configuração do banco via flags
	var dbServer, dbUser, dbPassword, dbName string
	var dbPort int

	flag.StringVar(&dbServer, "server", "", "SQL Server address")
	flag.IntVar(&dbPort, "port", 1433, "SQL Server port")
	flag.StringVar(&dbUser, "user", "", "SQL Server user")
	flag.StringVar(&dbPassword, "password", "", "SQL Server password")
	flag.StringVar(&dbName, "database", "", "SQL Server database name")
	flag.Parse()

	// Verifica variáveis de ambiente caso os argumentos não sejam fornecidos
	if dbServer == "" {
		dbServer = os.Getenv("DB_SERVER")
	}
	if dbUser == "" {
		dbUser = os.Getenv("DB_USER")
	}
	if dbPassword == "" {
		dbPassword = os.Getenv("DB_PASSWORD")
	}
	if dbName == "" {
		dbName = os.Getenv("DB_NAME")
	}

	// Configuração do banco de dados
	dbConfig := database.DBConfig{
		Server:   dbServer,
		Port:     dbPort,
		User:     dbUser,
		Password: dbPassword,
		Database: dbName,
	}

	// Conecta ao banco de dados
	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer db.Close()

	// Cria o servidor MCP com transporte stdio
	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())

	// Registra os MCPs disponíveis
	if err := esptag.RegisterConsultaEspecializacao(server, db); err != nil {
		log.Fatalf("Erro ao registrar MCP de consulta: %v", err)
	}

	// Adicione esta linha para registrar o novo MCP
	if err := esptag.RegisterGeraScriptNovaEspecializacao(server, db); err != nil {
		log.Fatalf("Erro ao registrar MCP de geração de script: %v", err)
	}

	// Inicia o servidor
	log.Println("Iniciando servidor MCP para Especialização de Tags...")
	if err := server.Serve(); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}

	// Mantém o servidor em execução
	select {}
}
