package util

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// Helper function to find the target tag, its parent, and a sub-path from XML using proper parsing
func FindTagParentAndPath(xmlInput string, targetTag string) (parentTag string, subPath []string, err error) {
	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlInput)))
	var stack []string // Stack para rastrear o caminho atual
	found := false

	for {
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return "", nil, fmt.Errorf("erro ao analisar token XML: %v", tokenErr)
		}

		switch se := token.(type) {
		case xml.StartElement:
			// Usa o nome Local para ignorar prefixos de namespace
			tagName := se.Name.Local
			stack = append(stack, tagName) // Adiciona tag ao stack

			if tagName == targetTag && !found {
				found = true

				// Define a tag pai (se existir)
				if len(stack) > 1 {
					parentTag = stack[len(stack)-2]
				}

				// Para o subPath, queremos o caminho completo desde a raiz até a tag alvo
				// inclusive a própria tag alvo
				subPath = make([]string, len(stack))
				copy(subPath, stack)
			}

		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1] // Remove tag do stack
			}
		}
	}

	if !found {
		return "", nil, fmt.Errorf("tag alvo '%s' não encontrada no XML fornecido", targetTag)
	}

	return parentTag, subPath, nil
}
