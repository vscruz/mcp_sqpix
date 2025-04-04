package esptag

import (
	"database/sql"
	"fmt"
	"strings"

	mcp_golang "github.com/metoro-io/mcp-golang"
)

// ConsultaEspecializacaoArgs define os argumentos de entrada para o MCP
type ConsultaEspecializacaoArgs struct {
	Termo string `json:"termo" jsonschema:"description=Termo para buscar especializações (busca parcial)"`
	ID    int    `json:"id" jsonschema:"description=ID da especialização para busca direta"`
}

// RegisterConsultaEspecializacao registra o MCP de consulta de especialização
func RegisterConsultaEspecializacao(server *mcp_golang.Server, db *sql.DB) error {
	return server.RegisterTool("sq_pix_esptag_consulta_especializacao",
		"Consulta especializações de tag que correspondem a um termo de busca ou ID específico",
		func(args ConsultaEspecializacaoArgs) (*mcp_golang.ToolResponse, error) {

			var resultado strings.Builder

			// Verifica se foi fornecido um ID
			if args.ID > 0 {
				// Consulta por ID
				esp, err := ConsultaEspecializacaoPorID(db, args.ID)
				if err != nil {
					return nil, fmt.Errorf("erro ao consultar especialização por ID: %v", err)
				}

				if esp == nil {
					resultado.WriteString(fmt.Sprintf("Nenhuma especialização encontrada com o ID %d.", args.ID))
				} else {
					resultado.WriteString(fmt.Sprintf("Especialização encontrada:\n\nID: %d - Descrição: %s\n", esp.ID, esp.Descricao))
					resultado.WriteString("\nEsta especialização já existe. Não é necessário gerar script para criá-la.")
				}
			} else {
				// Validação de entrada para busca por termo
				if args.Termo == "" {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: É necessário fornecer um termo de busca ou um ID válido.")), nil
				}

				// Consulta as especializações por termo
				especializacoes, err := ConsultaEspecializacao(db, args.Termo)
				if err != nil {
					return nil, fmt.Errorf("erro ao consultar especializações: %v", err)
				}

				// Formata o resultado como texto
				if len(especializacoes) == 0 {
					resultado.WriteString(fmt.Sprintf("Nenhuma especialização encontrada para o termo '%s'.", args.Termo))
				} else {
					resultado.WriteString(fmt.Sprintf("Encontradas %d especializações para o termo '%s':\n\n", len(especializacoes), args.Termo))

					for i, esp := range especializacoes {
						resultado.WriteString(fmt.Sprintf("%d. ID: %d - Descrição: %s\n", i+1, esp.ID, esp.Descricao))
					}

					resultado.WriteString("\nUtilize o ID da especialização desejada para criar a vinculação.")
				}
			}

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(resultado.String())), nil
		})
}
