package esptag

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
)

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
