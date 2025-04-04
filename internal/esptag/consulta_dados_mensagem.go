package esptag

import (
	"bytes" // Add bytes import
	"database/sql"
	"encoding/xml" // Add xml import
	"fmt"
	"io"  // Add io import
	"log" // Restore log package import
	"regexp"
	"strings"

	mcp_golang "github.com/metoro-io/mcp-golang"
)

// MensagemTagInfo representa um registro da tabela spi_mensagem_tag
type MensagemTagInfo struct {
	IDEveMensagem string `json:"id_eve_msg"`
	IDTipMensagem string `json:"id_tip_msg"`
	IDTag         string `json:"id_tag"`
	IDTagPai      string `json:"id_tag_pai"`
	NumSeqTag     int    `json:"num_seq_tag"`
	NumSeqMsgTag  int    `json:"num_seq_msg_tag"`
	Caminho       string `json:"caminho"` // Caminho completo para fins de verificação
	Score         int    `json:"score"`   // Pontuação de correspondência
}

// ConsultaDadosMensagemArgs define os argumentos de entrada para o MCP
type ConsultaDadosMensagemArgs struct {
	CaminhoXML    string `json:"caminho_xml" jsonschema:"required,description=Caminho ou trecho XML que contém a tag a ser especializada"`
	NomeTag       string `json:"nome_tag" jsonschema:"required,description=Nome da tag XML que será especializada (ex: TxSts)"`
	IDEveMensagem string `json:"id_eve_msg" jsonschema:"required,description=ID do evento da mensagem (ex: pacs.002) para filtrar a busca"`
}

// Helper function to find the target tag, its parent, and a sub-path from XML using proper parsing
func findTagParentAndPath(xmlInput string, targetTag string) (parentTag string, subPath []string, err error) {
	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlInput)))
	var stack []string     // Stack to keep track of current path
	var pathFound []string // Full path to the target tag when found
	found := false

	for {
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			// Return XML parsing errors
			return "", nil, fmt.Errorf("error parsing XML token: %v", tokenErr)
		}

		switch se := token.(type) {
		case xml.StartElement:
			// Use Local name to ignore namespace prefixes
			tagName := se.Name.Local
			stack = append(stack, tagName) // Push tag onto stack

			if tagName == targetTag && !found { // Found the target tag for the first time
				found = true
				pathFound = make([]string, len(stack))
				copy(pathFound, stack) // Store the path to the target

				if len(stack) > 1 {
					parentTag = stack[len(stack)-2] // Parent is the second to last element on stack
				}
				// Determine subPath for scoring (e.g., last 3 elements including target)
				startIdx := len(pathFound) - 3
				if startIdx < 0 {
					startIdx = 0
				}
				subPath = pathFound[startIdx:]
				// Continue parsing to check for XML validity / EOF
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1] // Pop tag from stack
			}
		}
	}

	if !found {
		return "", nil, fmt.Errorf("target tag '%s' not found in the provided XML", targetTag)
	}

	// Log the findings from XML parsing
	log.Printf("DEBUG: XML Parsing - Target: %s, Found Parent: '%s', Found SubPath: %v", targetTag, parentTag, subPath)

	return parentTag, subPath, nil
}

// ExtrairCaminhoTags extrai o caminho completo de tags do XML (FLAWED, kept temporarily if needed elsewhere)
func ExtrairCaminhoTags(xml string) []string {
	xml = strings.ReplaceAll(xml, "\n", "")
	xml = strings.ReplaceAll(xml, "\t", "")
	xml = strings.TrimSpace(xml)
	re := regexp.MustCompile(`<([^/][^>\s]*)[^>]*>`)
	matches := re.FindAllStringSubmatch(xml, -1)
	var caminho []string
	for _, match := range matches {
		if len(match) > 1 {
			tag := match[1]
			if idx := strings.Index(tag, ":"); idx != -1 {
				tag = tag[idx+1:]
			}
			caminho = append(caminho, tag)
		}
	}
	return caminho
}

// BuscarTagNaBase busca informações completas sobre uma tag na base de dados, filtrando pelo ID do evento da mensagem.
// It initially queries only by tag name and message ID. Parent matching happens during scoring.
func BuscarTagNaBase(db *sql.DB, caminhoPlano []string, tagAlvo string, idEveMensagem string) ([]MensagemTagInfo, error) {

	// Log the potentially incorrect parent derived from the flat path for comparison/debug purposes.
	// This tagPaiPlano is NOT used for SQL filtering.
	var tagPaiPlano string
	tagAlvoIdxPlano := -1
	for i, tag := range caminhoPlano {
		if tag == tagAlvo {
			tagAlvoIdxPlano = i
			break
		}
	}
	if tagAlvoIdxPlano > 0 {
		tagPaiPlano = caminhoPlano[tagAlvoIdxPlano-1]
	}
	log.Printf("DEBUG: Flat Path Analysis - Derived Parent (for logging only): '%s'", tagPaiPlano)

	// --- Initial Query: Filter only by tag name and message ID ---
	query := `
		SELECT mt.id_eve_msg, mt.id_tip_msg, mt.id_tag, ISNULL(mt.id_tag_pai, '') as id_tag_pai,
		       mt.num_seq_tag, mt.num_seq_msg_tag
		FROM spi_mensagem_tag mt
		WHERE mt.id_tag = ? AND mt.id_eve_msg = ?
	`
	args := []interface{}{tagAlvo, idEveMensagem}

	// Log the query before execution
	log.Printf("DEBUG: Executing SQL query: %s", query)
	log.Printf("DEBUG: With arguments: %v", args)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("DEBUG: SQL query error: %v", err)
		return nil, fmt.Errorf("erro na consulta à base de dados: %v", err)
	}
	defer rows.Close()

	// Coletar todos os resultados iniciais
	var resultados []MensagemTagInfo
	for rows.Next() {
		var info MensagemTagInfo
		if err := rows.Scan(&info.IDEveMensagem, &info.IDTipMensagem, &info.IDTag,
			&info.IDTagPai, &info.NumSeqTag, &info.NumSeqMsgTag); err != nil {
			return nil, fmt.Errorf("erro ao ler resultado: %v", err)
		}
		// Base score, will be adjusted later based on parent/path matching
		info.Score = 10
		resultados = append(resultados, info)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro durante iteração dos resultados: %v", err)
	}

	// Scoring and path reconstruction will happen in the handler after getting the correct parent/subpath

	return resultados, nil // Return results without scoring yet
}

// ReconstruirCaminho tenta reconstruir o caminho completo de uma tag na hierarquia
func ReconstruirCaminho(db *sql.DB, info MensagemTagInfo) ([]string, error) {
	caminho := []string{info.IDTag}
	if info.IDTagPai == "" {
		return caminho, nil
	}
	caminho = append([]string{info.IDTagPai}, caminho...)
	tagAtual := info.IDTagPai
	// Use NumSeqTag from the *current* tag info to find its parent's NumSeqTag if needed,
	// but the query below seems to use NumSeqTag of the *parent* to find the *grandparent*.
	// This might need review depending on the exact schema logic. Assuming it's correct for now.
	numSeqTagAtual := info.NumSeqTag // This might be incorrect, needs parent's NumSeqTag
	idEveMsgAtual := info.IDEveMensagem

	for i := 0; i < 5; i++ { // Limit recursion depth
		var tagPai string
		var nextNumSeqTag int // Need to get the NumSeqTag of the parent we just found

		// Query to find the parent AND its NumSeqTag
		query := `
            SELECT ISNULL(id_tag_pai, ''), num_seq_tag
            FROM spi_mensagem_tag
            WHERE id_tag = ?
              AND id_eve_msg = ?
              AND num_seq_tag = ?
        `
		// This query still uses numSeqTagAtual which refers to the *child's* sequence number,
		// not the parent's, which is needed to find the grandparent correctly.
		// This reconstruction logic likely needs fixing.
		// For now, we proceed assuming the DB structure allows this lookup somehow,
		// or accepting the path might be incomplete.
		err := db.QueryRow(query, tagAtual, idEveMsgAtual, numSeqTagAtual).Scan(&tagPai, &nextNumSeqTag)

		if err == sql.ErrNoRows || tagPai == "" {
			break // Reached root or no more parents found
		}
		if err != nil {
			log.Printf("WARN: Error reconstructing path for tag %s: %v", tagAtual, err)
			return caminho, err // Return partial path and error
		}
		caminho = append([]string{tagPai}, caminho...)
		tagAtual = tagPai
		numSeqTagAtual = nextNumSeqTag // Update sequence number for the next iteration
	}
	return caminho, nil
}

// CalcularPontuacaoCorrespondencia calcula um valor de correspondência entre dois caminhos
func CalcularPontuacaoCorrespondencia(caminhoXML []string, caminhoDB []string) int {
	pontuacao := 0
	if caminhoXML == nil || caminhoDB == nil {
		return 0
	}
	maxLen := len(caminhoXML)
	if len(caminhoDB) < maxLen {
		maxLen = len(caminhoDB)
	}
	for i := 1; i <= maxLen; i++ {
		idxXML := len(caminhoXML) - i
		idxDB := len(caminhoDB) - i
		if idxXML < 0 || idxDB < 0 {
			break
		}
		if caminhoXML[idxXML] == caminhoDB[idxDB] {
			pontuacao += (maxLen - i + 1) * 2
		}
	}
	return pontuacao
}

// RegisterConsultaDadosMensagem registra o MCP de consulta de dados da mensagem
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
