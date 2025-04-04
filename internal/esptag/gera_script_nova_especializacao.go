package esptag

import (
	"database/sql"
	"fmt"
	"strings"
)

// GeraScriptNovaEspecializacaoArgs define os argumentos de entrada para o MCP
type GeraScriptNovaEspecializacaoArgs struct {
	Descricao string `json:"descricao" jsonschema:"required,description=Descrição da nova especialização a ser criada"`
	ID        *int   `json:"id" jsonschema:"description=ID opcional para a especialização. Se não fornecido, será calculado automaticamente"`
}

// ObterProximoID consulta o próximo ID disponível para especialização
func ObterProximoID(db *sql.DB) (int, error) {
	query := `
		SELECT ISNULL(MAX(id_esp_tag), 0) + 1 
		FROM spi_especializacao_tag
	`

	var proximoID int
	err := db.QueryRow(query).Scan(&proximoID)
	if err != nil {
		return 0, fmt.Errorf("erro ao obter próximo ID: %v", err)
	}

	return proximoID, nil
}

// VerificarIDExistente verifica se um ID já está em uso
func VerificarIDExistente(db *sql.DB, id int) (bool, error) {
	// Use placeholder posicional (?) para compatibilidade
	query := `
		SELECT 1 
		FROM spi_especializacao_tag 
		WHERE id_esp_tag = ? 
	`

	var existe int
	// Passa o argumento diretamente para o placeholder posicional
	err := db.QueryRow(query, id).Scan(&existe)

	if err == sql.ErrNoRows {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("erro ao verificar ID existente: %v", err)
	}

	return true, nil
}

// GeraScriptNovaEspecializacao gera o script SQL para inserir uma nova especialização
func GeraScriptNovaEspecializacao(descricao string, id int) string {
	// Gera um script SQL com ID fixo
	script := strings.Builder{}

	script.WriteString("-- Script para criar nova especialização de tag\n")
	script.WriteString(fmt.Sprintf("-- Descrição: %s\n", descricao))
	script.WriteString(fmt.Sprintf("-- ID: %d\n\n", id))

	script.WriteString(fmt.Sprintf("IF NOT EXISTS (SELECT 1 FROM spi_especializacao_tag WHERE id_esp_tag = %d)\nBEGIN\n", id))

	script.WriteString(fmt.Sprintf("  INSERT INTO spi_especializacao_tag (id_esp_tag, dsc_esp_tag)\n"))
	script.WriteString(fmt.Sprintf("  VALUES (%d, '%s')\n", id, strings.Replace(descricao, "'", "''", -1)))

	script.WriteString("END\n")

	return script.String()
}
