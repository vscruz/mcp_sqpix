package esptag

import (
	"database/sql"
	"fmt"
	"strings"

	mcp_golang "github.com/metoro-io/mcp-golang"
)

// RegisterGeraScriptSitMsgEmiDes registers the MCP tool for generating spi_sit_msg_emi_des insert script
func RegisterGeraScriptSitMsgEmiDes(server *mcp_golang.Server, db *sql.DB) error {
	return server.RegisterTool("sq_pix_esptag_gera_script_sit_msg_emi_des",
		"Gera script SQL para inserir um novo registro na tabela spi_sit_msg_emi_des (Situação Mensagem Emissor Destinatario), verificando se já existe",
		func(args GeraScriptSitMsgEmiDesArgs) (*mcp_golang.ToolResponse, error) {

			// --- Input Validation ---
			if args.IDSitMsgEmiDes == "" {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: id_sit_msg_emi_des não pode ser vazio")), nil
			}
			if args.IDTipEmiDes <= 0 { // Basic check, adjust if 0 is valid
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: id_tip_emi_des deve ser um número positivo")), nil
			}
			if args.IDSitMsg <= 0 { // Basic check, adjust if 0 is valid
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: id_sit_msg deve ser um número positivo")), nil
			}
			if args.DscSitMsgEmiDes == "" {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: dsc_sit_msg_emi_des não pode ser vazio")), nil
			}

			// --- Check if record already exists ---
			existe, err := VerificarSitMsgEmiDesExistente(db, args.IDSitMsgEmiDes, args.IDTipEmiDes, args.IDSitMsg)
			if err != nil {
				// Return internal error if DB check fails
				return nil, fmt.Errorf("erro ao verificar existência do registro: %v", err)
			}

			var resultado strings.Builder

			if existe {
				resultado.WriteString(fmt.Sprintf("Aviso: Já existe um registro na tabela spi_sit_msg_emi_des com id_sit_msg_emi_des = '%s'.\n", args.IDSitMsgEmiDes))
				resultado.WriteString("Nenhum script de inserção será gerado.\n")
			} else {
				// --- Generate the script ---
				script := GeraScriptSitMsgEmiDes(args)
				resultado.WriteString(script)
			}

			// --- Return the result ---
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(resultado.String())), nil
		})
}
