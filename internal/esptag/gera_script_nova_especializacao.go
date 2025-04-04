package esptag

import (
	"database/sql"
	"fmt"
	"strings"

	mcp_golang "github.com/metoro-io/mcp-golang"
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

// RegisterGeraScriptNovaEspecializacao registra o MCP de geração de script para nova especialização
func RegisterGeraScriptNovaEspecializacao(server *mcp_golang.Server, db *sql.DB) error {
	return server.RegisterTool("sq_pix_esptag_gera_script_nova_especializacao",
		"Gera script SQL para criar uma nova especialização de tag",
		func(args GeraScriptNovaEspecializacaoArgs) (*mcp_golang.ToolResponse, error) {

			// Validação de entrada
			if args.Descricao == "" {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: Descrição não pode ser vazia")), nil
			}

			// Consulta para verificar se a especialização já existe
			esps, err := ConsultaEspecializacao(db, args.Descricao)
			if err != nil {
				return nil, fmt.Errorf("erro ao consultar especialização: %v", err)
			}

			// Prepara a resposta
			var resultado strings.Builder
			var idParaUsar int

			// Verifica se o usuário forneceu um ID
			if args.ID != nil {
				idExiste, err := VerificarIDExistente(db, *args.ID)
				if err != nil {
					return nil, fmt.Errorf("erro ao verificar ID existente: %v", err)
				}

				if idExiste {
					// ID já está em uso, sugerir um novo
					proximoID, err := ObterProximoID(db)
					if err != nil {
						return nil, fmt.Errorf("erro ao obter próximo ID: %v", err)
					}

					resultado.WriteString(fmt.Sprintf("Atenção: O ID %d já está em uso. Sugerimos usar o ID %d.\n\n", *args.ID, proximoID))
					idParaUsar = proximoID
				} else {
					// ID fornecido está disponível
					idParaUsar = *args.ID
				}
			} else {
				// Usuário não forneceu ID, obter o próximo disponível
				proximoID, err := ObterProximoID(db)
				if err != nil {
					return nil, fmt.Errorf("erro ao obter próximo ID: %v", err)
				}

				idParaUsar = proximoID
			}

			// Verifica se existem especializações com descrição similar
			if len(esps) > 0 {
				resultado.WriteString(fmt.Sprintf("Atenção: Encontradas %d especializações similares. Verifique se a especialização já existe:\n\n", len(esps)))

				for i, e := range esps {
					resultado.WriteString(fmt.Sprintf("%d. ID: %d - Descrição: %s\n", i+1, e.ID, e.Descricao))
				}

				resultado.WriteString("\nCaso deseje prosseguir mesmo assim, segue o script para criar a nova especialização:\n\n")
			}

			// Gera o script SQL com o ID determinado
			script := GeraScriptNovaEspecializacao(args.Descricao, idParaUsar)
			resultado.WriteString(script)

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(resultado.String())), nil
		})
}
