package config

import (
	"reflect"
	"strings"

	"github.com/tessellated-io/pickaxe/log"
	"gopkg.in/yaml.v2"
)

func WriteYamlWithComments(config interface{}, header string, filename string, logger *log.Logger) error {
	fileData, err := addCommentsToYaml(config, header)
	if err != nil {
		return err
	}

	return SafeWrite(filename, fileData, logger)
}

func addCommentsToYaml(config interface{}, header string) ([]byte, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	var result strings.Builder

	// Add the custom header at the beginning
	if header != "" {
		result.WriteString("# " + header + "\n")
	}

	// Handle both struct and pointer to struct
	v := reflect.ValueOf(config)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	yamlStr := string(data)
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)

		// Extract YAML tag and comment from struct field
		yamlTag := field.Tag.Get("yaml")
		comment := field.Tag.Get("comment")

		// Find the line in YAML string that corresponds to this field
		lineStart := strings.Index(yamlStr, yamlTag+":")
		if lineStart >= 0 {
			lineEnd := strings.Index(yamlStr[lineStart:], "\n")
			if lineEnd < 0 {
				lineEnd = len(yamlStr)
			} else {
				lineEnd += lineStart
			}

			result.WriteString(yamlStr[:lineStart])

			// Write the comment with a preceding blank line
			if comment != "" {
				result.WriteString("\n# " + comment + "\n")
			}

			result.WriteString(yamlStr[lineStart:lineEnd])
			yamlStr = yamlStr[lineEnd:]
		}
	}

	result.WriteString(yamlStr)
	return []byte(result.String()), nil
}
