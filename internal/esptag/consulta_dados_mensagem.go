package esptag

import (
	"database/sql"
	"fmt"
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
	CaminhoXML string `json:"caminho_xml" jsonschema:"required,description=Caminho ou trecho XML que contém a tag a ser especializada"`
	NomeTag    string `json:"nome_tag" jsonschema:"required,description=Nome da tag XML que será especializada (ex: TxSts)"`
}

// ExtrairCaminhoTags extrai o caminho completo de tags do XML
func ExtrairCaminhoTags(xml string) []string {
	// Remove espaços em branco e quebras de linha
	xml = strings.ReplaceAll(xml, "\n", "")
	xml = strings.ReplaceAll(xml, "\t", "")
	xml = strings.TrimSpace(xml)

	// Regex para encontrar tags de abertura (ignorando atributos e fechamentos)
	re := regexp.MustCompile(`<([^/][^>\s]*)[^>]*>`)
	matches := re.FindAllStringSubmatch(xml, -1)

	var caminho []string
	for _, match := range matches {
		if len(match) > 1 {
			tag := match[1]
			// Ignora namespaces
			if idx := strings.Index(tag, ":"); idx != -1 {
				tag = tag[idx+1:]
			}
			caminho = append(caminho, tag)
		}
	}

	return caminho
}

// BuscarTagNaBase busca informações completas sobre uma tag na base de dados
func BuscarTagNaBase(db *sql.DB, caminho []string, tagAlvo string) ([]MensagemTagInfo, error) {
	if len(caminho) == 0 {
		return nil, fmt.Errorf("nenhuma tag encontrada no XML fornecido")
	}

	// Encontrar a posição da tag alvo no caminho
	tagAlvoIdx := -1
	for i, tag := range caminho {
		if tag == tagAlvo {
			tagAlvoIdx = i
			break
		}
	}

	if tagAlvoIdx == -1 {
		return nil, fmt.Errorf("tag '%s' não encontrada no XML fornecido", tagAlvo)
	}

	// Obter a tag pai direta e construir um subcaminho com alguns níveis acima
	var tagPai string
	var subcaminho []string

	if tagAlvoIdx > 0 {
		tagPai = caminho[tagAlvoIdx-1]

		// Pegar até 3 níveis acima para melhor contexto
		startIdx := tagAlvoIdx - 3
		if startIdx < 0 {
			startIdx = 0
		}
		subcaminho = caminho[startIdx : tagAlvoIdx+1]
	} else {
		subcaminho = caminho[0 : tagAlvoIdx+1]
	}

	// Primeiro, buscar todas as ocorrências da tag alvo
	query := `
		SELECT mt.id_eve_msg, mt.id_tip_msg, mt.id_tag, ISNULL(mt.id_tag_pai, '') as id_tag_pai, 
		       mt.num_seq_tag, mt.num_seq_msg_tag
		FROM spi_mensagem_tag mt
		WHERE mt.id_tag = ?
	`

	// Se temos uma tag pai, refinar a busca
	args := []interface{}{tagAlvo}
	if tagPai != "" {
		query += " AND mt.id_tag_pai = ?"
		args = append(args, tagPai)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
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

		// Iniciar com pontuação base de correspondência
		info.Score = 10

		// Incrementa pontos se a tag pai corresponde
		if tagPai != "" && info.IDTagPai == tagPai {
			info.Score += 5
		}

		resultados = append(resultados, info)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro durante iteração dos resultados: %v", err)
	}

	// Para cada resultado, obter o caminho completo para comparação
	for i := range resultados {
		// Consulta para reconstruir o caminho completo
		caminhoDB, err := ReconstruirCaminho(db, resultados[i])
		if err != nil {
			// Se não conseguir reconstruir, apenas continua
			resultados[i].Caminho = fmt.Sprintf("[%s]", resultados[i].IDTag)
			continue
		}

		resultados[i].Caminho = strings.Join(caminhoDB, " > ")

		// Calcular pontuação adicional baseada na correspondência do caminho
		pontuacaoAdicional := CalcularPontuacaoCorrespondencia(subcaminho, caminhoDB)
		resultados[i].Score += pontuacaoAdicional
	}

	// Ordenar resultados por pontuação (do maior para o menor)
	// Usando selection sort simples para este exemplo
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

	return resultados, nil
}

// ReconstruirCaminho tenta reconstruir o caminho completo de uma tag na hierarquia
func ReconstruirCaminho(db *sql.DB, info MensagemTagInfo) ([]string, error) {
	caminho := []string{info.IDTag}

	// Se não tiver tag pai, retorna apenas a própria tag
	if info.IDTagPai == "" {
		return caminho, nil
	}

	// Adiciona a tag pai conhecida
	caminho = append([]string{info.IDTagPai}, caminho...)

	// Tenta reconstruir mais níveis acima, até 5 níveis ou até chegar à raiz
	tagAtual := info.IDTagPai
	numSeqTagAtual := info.NumSeqTag
	idEveMsgAtual := info.IDEveMensagem

	for i := 0; i < 5; i++ {
		var tagPai string
		query := `
			SELECT ISNULL(id_tag_pai, '') 
			FROM spi_mensagem_tag 
			WHERE id_tag = ? 
			  AND id_eve_msg = ? 
			  AND num_seq_tag = ?
		`

		err := db.QueryRow(query, tagAtual, idEveMsgAtual, numSeqTagAtual).Scan(&tagPai)

		if err == sql.ErrNoRows || tagPai == "" {
			break
		}

		if err != nil {
			return caminho, err
		}

		// Adiciona ao caminho e continua para cima
		caminho = append([]string{tagPai}, caminho...)
		tagAtual = tagPai
	}

	return caminho, nil
}

// CalcularPontuacaoCorrespondencia calcula um valor de correspondência entre dois caminhos
func CalcularPontuacaoCorrespondencia(caminhoXML, caminhoDB []string) int {
	pontuacao := 0

	// Compara de trás para frente (das folhas para a raiz)
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
			// Tags mais próximas da tag alvo têm maior peso
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

			// Extrai o caminho completo do XML
			caminho := ExtrairCaminhoTags(args.CaminhoXML)

			// Busca informações na base de dados com análise de correspondência
			resultados, err := BuscarTagNaBase(db, caminho, args.NomeTag)
			if err != nil {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Erro: %v", err))), nil
			}

			// Formata o resultado
			var resposta strings.Builder

			if len(resultados) == 0 {
				resposta.WriteString(fmt.Sprintf("Nenhum registro encontrado para a tag '%s'.\n", args.NomeTag))
				resposta.WriteString("Verifique se o nome da tag está correto e se ela existe na estrutura de mensagens do sistema.")
			} else {
				resposta.WriteString(fmt.Sprintf("Encontrados %d possíveis registros para a tag '%s'.\n", len(resultados), args.NomeTag))
				resposta.WriteString("Os resultados estão ordenados pelo melhor match com o caminho XML fornecido:\n\n")

				// Sinaliza se temos uma correspondência clara ou se existem múltiplas opções possíveis
				if len(resultados) == 1 || (len(resultados) > 1 && resultados[0].Score > resultados[1].Score+10) {
					resposta.WriteString("*** CORRESPONDÊNCIA EXATA ENCONTRADA ***\n\n")
				} else {
					resposta.WriteString("*** MÚLTIPLAS OPÇÕES POSSÍVEIS - VERIFICAÇÃO MANUAL NECESSÁRIA ***\n\n")
				}

				for i, info := range resultados {
					resposta.WriteString(fmt.Sprintf("Opção %d (Pontuação: %d):\n", i+1, info.Score))
					resposta.WriteString(fmt.Sprintf("- Caminho: %s\n", info.Caminho))
					resposta.WriteString(fmt.Sprintf("- ID Evento Mensagem: %s\n", info.IDEveMensagem))
					resposta.WriteString(fmt.Sprintf("- ID Tipo Mensagem: %s\n", info.IDTipMensagem))
					resposta.WriteString(fmt.Sprintf("- ID Tag: %s\n", info.IDTag))
					if info.IDTagPai != "" {
						resposta.WriteString(fmt.Sprintf("- ID Tag Pai: %s\n", info.IDTagPai))
					}
					resposta.WriteString(fmt.Sprintf("- Num. Seq. Tag: %d\n", info.NumSeqTag))
					resposta.WriteString(fmt.Sprintf("- Num. Seq. Msg Tag: %d\n", info.NumSeqMsgTag))

					// Adiciona comando para usar este registro diretamente
					resposta.WriteString("\nPara criar vinculação com esta opção, use:\n")
					resposta.WriteString(fmt.Sprintf("/mcp sq-pix-esptag sq_pix_esptag_gera_script_vinculacao {\"id_esp_tag\": SEU_ID_ESP_TAG, \"id_eve_msg\": \"%s\", \"id_tag\": \"%s\", \"id_tag_pai\": \"%s\", \"num_seq_tag\": %d, \"num_seq_msg_tag\": %d}\n\n",
						info.IDEveMensagem, info.IDTag, info.IDTagPai, info.NumSeqTag, info.NumSeqMsgTag))
				}
			}

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(resposta.String())), nil
		})
}
