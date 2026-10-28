package testing

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func LoadCategoryMappings(filePath string) (map[string]map[string]int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file: %v", err)
	}
	defer file.Close()

	result := make(map[string]map[string]int)
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 3
	reader.TrimLeadingSpace = true
	recordNumber := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		recordNumber++
		if err != nil {
			return nil, fmt.Errorf("invalid mapping CSV record %d: %w", recordNumber, err)
		}

		colName := strings.TrimSpace(record[0])
		categoryName := strings.TrimSpace(record[1])
		idStr := strings.TrimSpace(record[2])
		if colName == "" || categoryName == "" || idStr == "" {
			return nil, fmt.Errorf("invalid mapping CSV record %d: all three fields are required", recordNumber)
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid mapping id %q at record %d: %w", idStr, recordNumber, err)
		}
		if result[colName] == nil {
			result[colName] = make(map[string]int)
		}
		if _, duplicate := result[colName][categoryName]; duplicate {
			return nil, fmt.Errorf("duplicate mapping for column %q category %q at record %d", colName, categoryName, recordNumber)
		}
		result[colName][categoryName] = id
	}

	return result, nil
}
