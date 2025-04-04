package esptag

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
