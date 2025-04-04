package esptag

import (
	"database/sql"
	"fmt"
	"log"
)

// BuscarTagNaBase busca informações completas sobre uma tag na base de dados
func BuscarTagNaBase(db *sql.DB, caminhoPlano []string, tagAlvo string, idEveMensagem string) ([]MensagemTagInfo, error) {
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

	query := `
		SELECT mt.id_eve_msg, mt.id_tip_msg, mt.id_tag, ISNULL(mt.id_tag_pai, '') as id_tag_pai,
		       mt.num_seq_tag, mt.num_seq_msg_tag
		FROM spi_mensagem_tag mt
		WHERE mt.id_tag = ? AND mt.id_eve_msg = ?
	`
	args := []interface{}{tagAlvo, idEveMensagem}

	log.Printf("DEBUG: Executing SQL query: %s", query)
	log.Printf("DEBUG: With arguments: %v", args)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("DEBUG: SQL query error: %v", err)
		return nil, fmt.Errorf("erro na consulta à base de dados: %v", err)
	}
	defer rows.Close()

	var resultados []MensagemTagInfo
	for rows.Next() {
		var info MensagemTagInfo
		if err := rows.Scan(&info.IDEveMensagem, &info.IDTipMensagem, &info.IDTag,
			&info.IDTagPai, &info.NumSeqTag, &info.NumSeqMsgTag); err != nil {
			return nil, fmt.Errorf("erro ao ler resultado: %v", err)
		}
		info.Score = 10
		resultados = append(resultados, info)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro durante iteração dos resultados: %v", err)
	}

	return resultados, nil
}

// ReconstruirCaminho tenta reconstruir o caminho completo de uma tag na hierarquia
func ReconstruirCaminho(db *sql.DB, info MensagemTagInfo) ([]string, error) {
	caminho := []string{info.IDTag}
	if info.IDTagPai == "" {
		return caminho, nil
	}
	caminho = append([]string{info.IDTagPai}, caminho...)
	tagAtual := info.IDTagPai
	numSeqTagAtual := info.NumSeqTag
	idEveMsgAtual := info.IDEveMensagem

	for i := 0; i < 5; i++ {
		var tagPai string
		var nextNumSeqTag int

		query := `
            SELECT ISNULL(id_tag_pai, ''), num_seq_tag
            FROM spi_mensagem_tag
            WHERE id_tag = ?
              AND id_eve_msg = ?
              AND num_seq_tag = ?
        `
		err := db.QueryRow(query, tagAtual, idEveMsgAtual, numSeqTagAtual).Scan(&tagPai, &nextNumSeqTag)

		if err == sql.ErrNoRows || tagPai == "" {
			break
		}
		if err != nil {
			log.Printf("WARN: Error reconstructing path for tag %s: %v", tagAtual, err)
			return caminho, err
		}
		caminho = append([]string{tagPai}, caminho...)
		tagAtual = tagPai
		numSeqTagAtual = nextNumSeqTag
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
