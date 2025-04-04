package esptag

import (
	"database/sql"
	"fmt"
	"strings"
)

// GeraScriptVinculacaoArgs define os argumentos de entrada para o MCP
type GeraScriptVinculacaoArgs struct {
	IDEspecializacao int    `json:"id_esp_tag" jsonschema:"required,description=ID da especialização que será vinculada"`
	IDEveMensagem    string `json:"id_eve_msg" jsonschema:"required,description=ID do evento da mensagem (ex: pain.012)"`
	IDTag            string `json:"id_tag" jsonschema:"required,description=ID da tag (ex: MndtId)"`
	IDTagPai         string `json:"id_tag_pai" jsonschema:"required,description=ID da tag pai (ex: OrgnlMndt)"`
	NumSeqTag        int    `json:"num_seq_tag" jsonschema:"required,description=Número sequencial da tag"`
	NumSeqMsgTag     int    `json:"num_seq_msg_tag" jsonschema:"required,description=Número sequencial da mensagem tag"`
}

// VerificarEspecializacaoExiste verifica se uma especialização existe
func VerificarEspecializacaoExiste(db *sql.DB, idEspTag int) (bool, error) {
	query := `
		SELECT 1 
		FROM spi_especializacao_tag 
		WHERE id_esp_tag = ?
	`

	var existe int
	err := db.QueryRow(query, idEspTag).Scan(&existe)

	if err == sql.ErrNoRows {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("erro ao verificar ID existente: %v", err)
	}

	return true, nil
}

// GeraScriptVinculacao gera o script SQL para vincular uma especialização a uma mensagem
func GeraScriptVinculacao(args GeraScriptVinculacaoArgs) string {
	script := strings.Builder{}

	script.WriteString("-- Script para vincular especialização a uma mensagem\n")
	script.WriteString(fmt.Sprintf("-- ID Especialização: %d\n", args.IDEspecializacao))
	script.WriteString(fmt.Sprintf("-- ID Evento Mensagem: %s\n", args.IDEveMensagem))
	script.WriteString(fmt.Sprintf("-- ID Tag: %s\n", args.IDTag))
	if args.IDTagPai != "" {
		script.WriteString(fmt.Sprintf("-- ID Tag Pai: %s\n", args.IDTagPai))
	}
	script.WriteString(fmt.Sprintf("-- Num. Seq. Tag: %d\n", args.NumSeqTag))
	script.WriteString(fmt.Sprintf("-- Num. Seq. Msg Tag: %d\n\n", args.NumSeqMsgTag))

	// Construir cláusula WHERE para o script
	whereClause := strings.Builder{}
	whereClause.WriteString(fmt.Sprintf("mt.id_eve_msg = '%s'\n", strings.Replace(args.IDEveMensagem, "'", "''", -1)))
	whereClause.WriteString(fmt.Sprintf("                 AND mt.id_tag = '%s'\n", strings.Replace(args.IDTag, "'", "''", -1)))
	if args.IDTagPai != "" {
		whereClause.WriteString(fmt.Sprintf("                 AND mt.id_tag_pai = '%s'\n", strings.Replace(args.IDTagPai, "'", "''", -1)))
	}
	whereClause.WriteString(fmt.Sprintf("                 AND mt.num_seq_tag = %d\n", args.NumSeqTag))
	whereClause.WriteString(fmt.Sprintf("                 AND mt.num_seq_msg_tag = %d\n", args.NumSeqMsgTag))
	whereClause.WriteString(fmt.Sprintf("                 AND em.id_esp_tag = %d", args.IDEspecializacao))

	// Script IF NOT EXISTS + INSERT
	script.WriteString("IF NOT EXISTS (SELECT 1\n")
	script.WriteString("               FROM spi_mensagem_tag mt\n")
	script.WriteString("                    JOIN spi_especializacao_msg_tag em\n")
	script.WriteString("                    ON em.num_seq_msg_tag = mt.num_seq_msg_tag\n")
	script.WriteString("                   AND em.num_seq_tag = mt.num_seq_tag\n")
	script.WriteString("                   AND em.id_eve_msg = mt.id_eve_msg\n")
	script.WriteString("                   AND em.id_tip_msg = mt.id_tip_msg\n")
	script.WriteString("                   AND em.id_tag = mt.id_tag\n")
	script.WriteString("               WHERE ")
	script.WriteString(whereClause.String())
	script.WriteString(")\nBEGIN\n")

	script.WriteString("  INSERT INTO spi_especializacao_msg_tag (id_esp_tag, id_eve_msg, id_tip_msg, id_tag, num_seq_tag, num_seq_msg_tag)\n")
	script.WriteString("  SELECT ")
	script.WriteString(fmt.Sprintf("%d", args.IDEspecializacao))
	script.WriteString(", id_eve_msg, id_tip_msg, id_tag, num_seq_tag, num_seq_msg_tag\n")
	script.WriteString("  FROM spi_mensagem_tag\n")
	script.WriteString("  WHERE ")

	// WHERE para a tabela spi_mensagem_tag
	whereTagOnly := strings.Builder{}
	whereTagOnly.WriteString(fmt.Sprintf("id_eve_msg = '%s'\n", strings.Replace(args.IDEveMensagem, "'", "''", -1)))
	whereTagOnly.WriteString(fmt.Sprintf("    AND id_tag = '%s'\n", strings.Replace(args.IDTag, "'", "''", -1)))
	if args.IDTagPai != "" {
		whereTagOnly.WriteString(fmt.Sprintf("    AND id_tag_pai = '%s'\n", strings.Replace(args.IDTagPai, "'", "''", -1)))
	}
	whereTagOnly.WriteString(fmt.Sprintf("    AND num_seq_tag = %d\n", args.NumSeqTag))
	whereTagOnly.WriteString(fmt.Sprintf("    AND num_seq_msg_tag = %d", args.NumSeqMsgTag))

	script.WriteString(whereTagOnly.String())
	script.WriteString("\nEND")

	return script.String()
}
