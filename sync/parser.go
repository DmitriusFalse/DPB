package sync

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

func ParseCSV(r io.Reader) ([]TagResult, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	var tags []TagResult
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("csv read: %w", err)
		}

		if len(record) < 4 {
			continue
		}

		tag := TagResult{
			TagName:         strings.TrimSpace(record[0]),
			CategoryName:    strings.TrimSpace(record[1]),
			SubcategoryName: strings.TrimSpace(record[2]),
			Aliases:         strings.TrimSpace(record[3]),
		}

		if tag.TagName == "" {
			continue
		}

		tags = append(tags, tag)
	}

	return tags, nil
}

func ParseTXT(r io.Reader, categoryName, subcategoryName string) ([]TagResult, error) {
	var tags []TagResult
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tags = append(tags, TagResult{
			TagName:         line,
			CategoryName:    categoryName,
			SubcategoryName: subcategoryName,
			Aliases:         "",
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("txt read: %w", err)
	}
	return tags, nil
}
