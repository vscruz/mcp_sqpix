# Servidor MCP - Especialização de Tags PIX (sq-pix-esptag)

## Overview

Um servidor Model Context Protocol (MCP) para interagir com a funcionalidade de especialização de tags do sistema PIX da Sinqia. Este servidor fornece ferramentas para consultar dados, gerar scripts SQL para novas especializações e vincular especializações existentes a mensagens PIX, facilitando a interação via Large Language Models.

## Tools

As seguintes ferramentas são expostas por este servidor MCP:

1.  **`sq_pix_esptag_consulta_dados_mensagem`**
    *   Consulta dados detalhados de uma tag em uma mensagem PIX específica.
    *   **Input:**
        *   `caminho_xml` (string, required): Trecho XML contendo a tag a ser consultada.
        *   `nome_tag` (string, required): Nome da tag XML a ser consultada (ex: `TxSts`).
        *   `id_eve_msg` (string, required): ID do evento da mensagem (ex: `pacs.002.001.10`).
    *   **Returns:** Lista de possíveis registros da tag encontrados na base, ordenados por relevância, com informações detalhadas e sugestão de comando para vinculação.

2.  **`sq_pix_esptag_consulta_especializacao`**
    *   Busca por especializações de tag existentes por termo ou ID.
    *   **Input:**
        *   `termo` (string): Termo para busca parcial na descrição da especialização.
        *   `id` (integer): ID exato da especialização para busca direta.
        *   *(Pelo menos um dos campos `termo` ou `id` deve ser fornecido)*
    *   **Returns:** Lista de especializações encontradas com ID e Descrição.

3.  **`sq_pix_esptag_gera_script_nova_especializacao`**
    *   Gera script SQL para criar uma nova especialização de tag.
    *   **Input:**
        *   `descricao` (string, required): Descrição da nova especialização.
        *   `id` (integer, optional): ID sugerido para a nova especialização. Se omitido ou já existente, um novo ID será sugerido.
    *   **Returns:** Script SQL formatado para inserir a nova especialização, com avisos sobre IDs existentes ou descrições similares.

4.  **`sq_pix_esptag_gera_script_vinculacao`**
    *   Gera script SQL para vincular uma especialização existente a uma tag específica em uma mensagem.
    *   **Input:**
        *   `id_esp_tag` (integer, required): ID da especialização a ser vinculada.
        *   `id_eve_msg` (string, required): ID do evento da mensagem (ex: `pain.012.001.03`).
        *   `id_tag` (string, required): ID (nome) da tag a ser vinculada (ex: `MndtId`).
        *   `id_tag_pai` (string, optional): ID (nome) da tag pai direta. Ajuda a desambiguar.
        *   `num_seq_tag` (integer, required): Número sequencial da tag na hierarquia da mensagem.
        *   `num_seq_msg_tag` (integer, required): Número sequencial único da tag na tabela `spi_mensagem_tag`.
    *   **Returns:** Script SQL formatado para inserir o vínculo na tabela `spi_especializacao_msg_tag`, com aviso caso a especialização informada não exista.

## Build

Para compilar o servidor MCP, execute o seguinte comando na raiz do projeto:

```bash
make build
```

Ou diretamente:

```bash
go build -o bin/sq_pix_esptag.exe cmd/mcp/main.go
```

O executável será gerado em `bin/sq_pix_esptag.exe`.

## Configuration

O servidor requer acesso ao banco de dados SQL Server do PIX. As credenciais podem ser fornecidas via:

1.  **Flags de Linha de Comando:**
    *   `-server <endereço>`: Endereço do SQL Server.
    *   `-port <porta>`: Porta do SQL Server (padrão: 1433).
    *   `-user <usuário>`: Usuário do SQL Server.
    *   `-password <senha>`: Senha do SQL Server.
    *   `-database <nome_db>`: Nome do banco de dados.

2.  **Variáveis de Ambiente (utilizadas se as flags correspondentes não forem fornecidas):**
    *   `DB_SERVER`
    *   `DB_USER`
    *   `DB_PASSWORD`
    *   `DB_NAME`
    *   *(A porta não pode ser configurada via variável de ambiente)*

### Exemplo de Configuração (Claude Desktop `cline_mcp_settings.json`)

```json
{
  "mcpServers": {
    "sq-pix-esptag": {
      "command": "F:/src/sinqia/sqpix/mcp/sq_pix/bin/sq_pix_esptag.exe",
      "args": [
        "-server", "SEU_SERVIDOR_SQL",
        "-user", "SEU_USUARIO_SQL",
        "-password", "SUA_SENHA_SQL",
        "-database", "SEU_BANCO_PIX"
      ],
      "disabled": false,
      "autoApprove": []
    }
  }
}
```

*Substitua os placeholders (`SEU_SERVIDOR_SQL`, etc.) pelos valores corretos.*
*Certifique-se de que o caminho para o executável (`command`) está correto.*

## Development / Debugging

Para executar o servidor localmente para desenvolvimento ou depuração (usando as credenciais do `Makefile` como exemplo):

```bash
make run
```

Ou diretamente:

```bash
go run cmd/mcp/main.go -server 10.110.104.4 -user sa -password P@ssw0rd -database DSV_PIX
```

Você pode usar o MCP Inspector para interagir com o servidor em execução:

```bash
# Exemplo assumindo que o servidor está rodando e escutando em stdio
npx @modelcontextprotocol/inspector stdio --cmd "go run cmd/mcp/main.go -server <server> -user <user> -password <pass> -database <db>"
```

---
*Esta documentação descreve o estado atual das ferramentas e configuração. Consulte o código-fonte para detalhes de implementação.*
