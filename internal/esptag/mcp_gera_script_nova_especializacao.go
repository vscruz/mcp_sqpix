package esptag

import (
	"database/sql"
	"fmt"
	"strings"

	mcp_golang "github.com/metoro-io/mcp-golang"
)

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
			esps, err := ConsultaEspecializacao(db, args.Descricao) // Assume ConsultaEspecializacao exists in this package
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
