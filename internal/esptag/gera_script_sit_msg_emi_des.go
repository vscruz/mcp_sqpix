package esptag

import (
	"database/sql"
	"fmt"
	"strings"
)

// GeraScriptSitMsgEmiDesArgs defines the arguments for the MCP tool
type GeraScriptSitMsgEmiDesArgs struct {
	IDSitMsgEmiDes  string `json:"id_sit_msg_emi_des" jsonschema:"required,description=ID da situação da mensagem (valor da tag XML, ex: RJCT)"`
	IDTipEmiDes     int    `json:"id_tip_emi_des" jsonschema:"required,description=ID do tipo de emissor/destinatário"`
	IDSitMsg        int    `json:"id_sit_msg" jsonschema:"required,description=ID da situação da mensagem (numérico)"`
	DscSitMsgEmiDes string `json:"dsc_sit_msg_emi_des" jsonschema:"required,description=Descrição da situação"`
	CodUsuUltMnt    *int   `json:"cod_usu_ult_mnt" jsonschema:"description=Código do usuário da última manutenção (opcional, default 0)"`
	// dat_ult_mnt will be handled by GETDATE() in the script
}

// VerificarSitMsgEmiDesExistente checks if a record exists in spi_sit_msg_emi_des based on the composite key
func VerificarSitMsgEmiDesExistente(db *sql.DB, idSitMsgEmiDes string, idTipEmiDes int, idSitMsg int) (bool, error) {
	query := `
		SELECT 1 
		FROM spi_sit_msg_emi_des 
		WHERE id_sit_msg_emi_des = ? 
		  AND id_tip_emi_des = ? 
		  AND id_sit_msg = ?
	`
	var existe int
	err := db.QueryRow(query, idSitMsgEmiDes, idTipEmiDes, idSitMsg).Scan(&existe)

	if err == sql.ErrNoRows {
		return false, nil // Not found
	}
	if err != nil {
		return false, fmt.Errorf("erro ao verificar existência de spi_sit_msg_emi_des: %v", err)
	}
	return true, nil // Found
}

// GeraScriptSitMsgEmiDes generates the SQL script to insert into spi_sit_msg_emi_des if not exists
func GeraScriptSitMsgEmiDes(args GeraScriptSitMsgEmiDesArgs) string {
	script := strings.Builder{}
	codUsu := 0 // Default user code
	if args.CodUsuUltMnt != nil {
		codUsu = *args.CodUsuUltMnt
	}

	// Escape single quotes in string values
	escapedIDSitMsgEmiDes := strings.Replace(args.IDSitMsgEmiDes, "'", "''", -1)
	escapedDscSitMsgEmiDes := strings.Replace(args.DscSitMsgEmiDes, "'", "''", -1)

	script.WriteString("-- Script para inserir registro em spi_sit_msg_emi_des se não existir\n")
	script.WriteString(fmt.Sprintf("-- ID Situação Mensagem Emissor Destinatário: %s\n", args.IDSitMsgEmiDes))
	script.WriteString(fmt.Sprintf("-- ID Tipo Emissor Destinatário: %d\n", args.IDTipEmiDes))
	script.WriteString(fmt.Sprintf("-- ID Situação Mensagem: %d\n", args.IDSitMsg))
	script.WriteString(fmt.Sprintf("-- Descrição: %s\n\n", args.DscSitMsgEmiDes))

	// Updated IF NOT EXISTS to include all key fields
	script.WriteString(fmt.Sprintf("IF NOT EXISTS (SELECT 1 FROM spi_sit_msg_emi_des WHERE id_sit_msg_emi_des = '%s' AND id_tip_emi_des = %d AND id_sit_msg = %d)\nBEGIN\n",
		escapedIDSitMsgEmiDes, args.IDTipEmiDes, args.IDSitMsg))

	script.WriteString("  INSERT INTO spi_sit_msg_emi_des (id_sit_msg_emi_des, id_tip_emi_des, id_sit_msg, dsc_sit_msg_emi_des, cod_usu_ult_mnt, dat_ult_mnt)\n")
	script.WriteString(fmt.Sprintf("  VALUES ('%s', %d, %d, '%s', %d, GETDATE())\n",
		escapedIDSitMsgEmiDes,
		args.IDTipEmiDes,
		args.IDSitMsg,
		escapedDscSitMsgEmiDes,
		codUsu,
	))

	script.WriteString("END\n")

	return script.String()
}
