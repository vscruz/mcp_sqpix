package esptag

import (
	"database/sql"
	"fmt"
)

// EspecializacaoTag representa uma especialização de tag no sistema
type EspecializacaoTag struct {
	ID        int    `json:"id_esp_tag"`
	Descricao string `json:"dsc_esp_tag"`
}

// ConsultaEspecializacaoPorID retorna uma especialização específica pelo seu ID
func ConsultaEspecializacaoPorID(db *sql.DB, id int) (*EspecializacaoTag, error) {
	query := `
		SELECT id_esp_tag, dsc_esp_tag 
		FROM spi_especializacao_tag 
		WHERE id_esp_tag = ?
	`

	var esp EspecializacaoTag
	err := db.QueryRow(query, id).Scan(&esp.ID, &esp.Descricao)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Retorna nil se não encontrar
		}
		return nil, fmt.Errorf("erro ao consultar especialização por ID: %v", err)
	}

	return &esp, nil
}

// ConsultaEspecializacao retorna uma lista de especializações que correspondem ao termo de busca
func ConsultaEspecializacao(db *sql.DB, termo string) ([]EspecializacaoTag, error) {
	// Consulta especializações usando LIKE para busca parcial
	query := `
		SELECT id_esp_tag, dsc_esp_tag 
		FROM spi_especializacao_tag 
		WHERE dsc_esp_tag LIKE ?
		ORDER BY dsc_esp_tag
	`

	// Adiciona caracteres curinga para busca parcial
	termoBusca := "%" + termo + "%"

	rows, err := db.Query(query, termoBusca)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar especializações: %v", err)
	}
	defer rows.Close()

	var especializacoes []EspecializacaoTag

	for rows.Next() {
		var esp EspecializacaoTag
		if err := rows.Scan(&esp.ID, &esp.Descricao); err != nil {
			return nil, fmt.Errorf("erro ao ler linha de resultado: %v", err)
		}
		especializacoes = append(especializacoes, esp)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro durante iteração dos resultados: %v", err)
	}

	return especializacoes, nil
}
