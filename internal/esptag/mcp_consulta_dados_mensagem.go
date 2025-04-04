package esptag

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	mcp_golang "github.com/metoro-io/mcp-golang"
)

// RegisterConsultaDadosMensagem registra o MCP de consulta de dados da mensagem
// Utiliza as funções do database.go para operações com o banco de dados
func RegisterConsultaDadosMensagem(server *mcp_golang.Server, db *sql.DB) error {
	return server.RegisterTool("sq_pix_esptag_consulta_dados_mensagem",
		"Consulta dados da mensagem a partir de um trecho XML",
		func(args ConsultaDadosMensagemArgs) (*mcp_golang.ToolResponse, error) {

			// Validação de entrada
			if args.CaminhoXML == "" {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: Caminho XML não pode ser vazio")), nil
			}
			if args.NomeTag == "" {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: Nome da tag não pode ser vazio")), nil
			}
			if args.IDEveMensagem == "" {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Erro: ID do evento da mensagem (id_eve_msg) não pode ser vazio")), nil
			}

			// --- Step 1: Parse XML correctly to find true parent and subpath ---
			tagPaiCorreta, subcaminhoXML, parseErr := findTagParentAndPath(args.CaminhoXML, args.NomeTag)
			if parseErr != nil {
				// Return parsing error to the user
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Erro ao processar XML: %v", parseErr))), nil
			}

			// --- Step 2: Get flat path (for BuscarTagNaBase signature - legacy) ---
			caminhoPlano := ExtrairCaminhoTags(args.CaminhoXML)

			// --- Step 3: Query Database (without parent filter initially) ---
			resultados, err := BuscarTagNaBase(db, caminhoPlano, args.NomeTag, args.IDEveMensagem)
			if err != nil {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Erro na consulta: %v", err))), nil
			}

			// --- Step 4: Score results using correct parent and path ---
			for i := range resultados {
				// Score starts at 10 (from BuscarTagNaBase)

				// Add score based on CORRECT parent match from XML parser
				if tagPaiCorreta != "" && resultados[i].IDTagPai == tagPaiCorreta {
					resultados[i].Score += 15 // Higher score for correct parent match
					log.Printf("DEBUG: Scoring - Correct parent '%s' matched DB parent '%s' for tag '%s'. Score +15.", tagPaiCorreta, resultados[i].IDTagPai, resultados[i].IDTag)
				} else if tagPaiCorreta != "" && resultados[i].IDTagPai != tagPaiCorreta {
					log.Printf("DEBUG: Scoring - Correct parent '%s' DID NOT match DB parent '%s' for tag '%s'.", tagPaiCorreta, resultados[i].IDTagPai, resultados[i].IDTag)
					// Optionally decrease score for mismatch?
					// resultados[i].Score -= 5
				} else {
					// No parent found in XML or DB parent is empty
					log.Printf("DEBUG: Scoring - No correct parent found in XML or DB parent empty for tag '%s'.", resultados[i].IDTag)
				}

				// Reconstruct DB path for path scoring
				caminhoDB, reconErr := ReconstruirCaminho(db, resultados[i])
				if reconErr != nil {
					resultados[i].Caminho = fmt.Sprintf("[%s] (Erro ao reconstruir caminho: %v)", resultados[i].IDTag, reconErr)
					// Keep base score if path reconstruction fails
				} else {
					resultados[i].Caminho = strings.Join(caminhoDB, " > ")
					// Add score based on path correspondence using CORRECT subpath from XML parser
					pontuacaoAdicional := CalcularPontuacaoCorrespondencia(subcaminhoXML, caminhoDB)
					resultados[i].Score += pontuacaoAdicional
					log.Printf("DEBUG: Scoring - Path match score for '%s': %d (XML subpath: %v)", resultados[i].IDTag, pontuacaoAdicional, subcaminhoXML)
				}
			}

			// --- Step 5: Sort results by final score ---
			// Using selection sort simple
			for i := 0; i < len(resultados); i++ {
				maxIdx := i
				for j := i + 1; j < len(resultados); j++ {
					if resultados[j].Score > resultados[maxIdx].Score {
						maxIdx = j
					}
				}
				if maxIdx != i {
					resultados[i], resultados[maxIdx] = resultados[maxIdx], resultados[i]
				}
			}

			// --- Step 6: Format the response ---
			var resposta strings.Builder
			if len(resultados) == 0 {
				resposta.WriteString(fmt.Sprintf("Nenhum registro encontrado para a tag '%s' na mensagem '%s'.\n", args.NomeTag, args.IDEveMensagem))
				resposta.WriteString("Verifique se o nome da tag e o ID da mensagem estão corretos e se a tag existe na estrutura desta mensagem no sistema.")
			} else {
				resposta.WriteString(fmt.Sprintf("Encontrados %d possíveis registros para a tag '%s' na mensagem '%s'.\n", len(resultados), args.NomeTag, args.IDEveMensagem))
				resposta.WriteString("Os resultados estão ordenados pelo melhor match (maior pontuação):\n\n")

				// Sinaliza se temos uma correspondência clara ou se existem múltiplas opções possíveis
				if len(resultados) == 1 || (len(resultados) > 1 && resultados[0].Score > resultados[1].Score+10) { // Increased threshold for "exact"
					resposta.WriteString("*** MELHOR CORRESPONDÊNCIA ENCONTRADA ***\n\n")
				} else if len(resultados) > 1 {
					resposta.WriteString("*** MÚLTIPLAS OPÇÕES POSSÍVEIS - VERIFICAÇÃO MANUAL RECOMENDADA ***\n\n")
				}

				for i, info := range resultados {
					resposta.WriteString(fmt.Sprintf("Opção %d (Pontuação: %d):\n", i+1, info.Score))
					resposta.WriteString(fmt.Sprintf("- Caminho DB: %s\n", info.Caminho)) // Show reconstructed DB path
					resposta.WriteString(fmt.Sprintf("- ID Evento Mensagem: %s\n", info.IDEveMensagem))
					// resposta.WriteString(fmt.Sprintf("- ID Tipo Mensagem: %s\n", info.IDTipMensagem)) // Maybe less relevant?
					resposta.WriteString(fmt.Sprintf("- ID Tag: %s\n", info.IDTag))
					resposta.WriteString(fmt.Sprintf("- ID Tag Pai (DB): %s\n", info.IDTagPai)) // Label as DB parent
					resposta.WriteString(fmt.Sprintf("- Num. Seq. Tag: %d\n", info.NumSeqTag))
					resposta.WriteString(fmt.Sprintf("- Num. Seq. Msg Tag: %d\n", info.NumSeqMsgTag))

					// Adiciona comando para usar este registro diretamente
					resposta.WriteString("\nPara criar vinculação com esta opção, use:\n")
					// Ensure the generated command uses the correct info from the database result
					resposta.WriteString(fmt.Sprintf("/mcp sq-pix-esptag sq_pix_esptag_gera_script_vinculacao {\"id_esp_tag\": SEU_ID_ESP_TAG, \"id_eve_msg\": \"%s\", \"id_tag\": \"%s\", \"id_tag_pai\": \"%s\", \"num_seq_tag\": %d, \"num_seq_msg_tag\": %d}\n\n",
						info.IDEveMensagem, info.IDTag, info.IDTagPai, info.NumSeqTag, info.NumSeqMsgTag))
				}
			}

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(resposta.String())), nil
		})
}
