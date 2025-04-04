package esptag

import (
	"database/sql"
	"fmt"
	"strings"

	mcp_golang "github.com/metoro-io/mcp-golang"
)

// ConsultaEspecializacaoArgs define os argumentos de entrada para o MCP
type ConsultaEspecializacaoArgs struct {
	Termo string `json:"termo" jsonschema:"required,description=Termo para buscar especializações (busca parcial)"`
}

// RegisterConsultaEspecializacao registra o MCP de consulta de especialização
func RegisterConsultaEspecializacao(server *mcp_golang.Server, db *sql.DB) error {
	return server.RegisterTool("sq_pix_esptag_consulta_especializacao",
		"Consulta especializações de tag que correspondem a um termo de busca",
		func(args ConsultaEspecializacaoArgs) (*mcp_golang.ToolResponse, error) {

			// Validação de entrada
			if args.Termo == "" {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: Termo de busca não pode ser vazio")), nil
			}

			// Consulta as especializações
			especializacoes, err := ConsultaEspecializacao(db, args.Termo)
			if err != nil {
				return nil, fmt.Errorf("erro ao consultar especializações: %v", err)
			}

			// Formata o resultado como texto
			var resultado strings.Builder

			if len(especializacoes) == 0 {
				resultado.WriteString(fmt.Sprintf("Nenhuma especialização encontrada para o termo '%s'.", args.Termo))
			} else {
				resultado.WriteString(fmt.Sprintf("Encontradas %d especializações para o termo '%s':\n\n", len(especializacoes), args.Termo))

				for i, esp := range especializacoes {
					resultado.WriteString(fmt.Sprintf("%d. ID: %d - Descrição: %s\n", i+1, esp.ID, esp.Descricao))
				}

				resultado.WriteString("\nUtilize o ID da especialização desejada para criar a vinculação.")
			}

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(resultado.String())), nil
		})
}
