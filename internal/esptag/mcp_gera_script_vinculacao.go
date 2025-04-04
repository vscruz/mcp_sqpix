package esptag

import (
	"database/sql"
	"fmt"
	"strings"

	mcp_golang "github.com/metoro-io/mcp-golang"
)

// RegisterGeraScriptVinculacao registra o MCP de geração de script para vinculação
func RegisterGeraScriptVinculacao(server *mcp_golang.Server, db *sql.DB) error {
	return server.RegisterTool("sq_pix_esptag_gera_script_vinculacao",
		"Gera script SQL para vincular uma especialização a uma mensagem",
		func(args GeraScriptVinculacaoArgs) (*mcp_golang.ToolResponse, error) {

			// Validação de entrada
			if args.IDEspecializacao <= 0 {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: ID da especialização deve ser maior que zero")), nil
			}

			if args.IDEveMensagem == "" {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: ID do evento da mensagem não pode ser vazio")), nil
			}

			if args.IDTag == "" {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: ID da tag não pode ser vazio")), nil
			}

			// Verifica se a especialização existe
			especialiacaoExiste, err := VerificarEspecializacaoExiste(db, args.IDEspecializacao) // Assume VerificarEspecializacaoExiste exists
			if err != nil {
				return nil, fmt.Errorf("erro ao verificar especialização: %v", err)
			}

			var resultado strings.Builder

			if !especialiacaoExiste {
				resultado.WriteString(fmt.Sprintf("Aviso: Especialização com ID %d não foi encontrada na base. ", args.IDEspecializacao))
				resultado.WriteString("O script será gerado, mas certifique-se de que a especialização exista antes de executá-lo.\n\n")
			}

			// Gera o script SQL
			script := GeraScriptVinculacao(args) // Assume GeraScriptVinculacao exists
			resultado.WriteString(script)

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(resultado.String())), nil
		})
}
