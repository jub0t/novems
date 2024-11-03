package parser

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
  "bytes"
	"strings"

	"github.com/charmbracelet/log"
)

type LimitedInfo struct {
	Price int    `json:"price"`
	Id    string `json:"id"`
}

type LineFormatError struct {
	err string
}

func (e *LineFormatError) Error() string {
	return e.err
}

// FromFile reads the file contents line by line from the provided path
// and parses each line into LimitedInfo. Each line should follow the syntax: id<int>, price<int>.
func FromFile(path string) ([]LimitedInfo, error) {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Use bufio.Scanner for efficient line-by-line reading
	scanner := bufio.NewScanner(file)
	var infos []LimitedInfo

	for scanner.Scan() {
		// Read the line
		line := scanner.Text()

		if len(line) == 0 {
			log.Warn("Empty line detected in limiteds file, remove it or it can cause slower parsing times.")
			continue
		}

		// Split the line into parts (id and price)
		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line format: %s", line)
		}

		price, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("failed to parse price: %w", err)
		}

		// Append parsed struct to slice
		infos = append(infos, LimitedInfo{
			Id:    strings.TrimSpace(parts[0]),
			Price: price,
		})
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return infos, nil
}

func ParseItemDetails(body []byte) (int, int, int, int) {
	// Helper function to extract numbers between specific delimiters
	extractValue := func(body []byte, start, end string) (int, error) {
		startIdx := bytes.Index(body, []byte(start))
		if startIdx == -1 {
			return -1, fmt.Errorf("start delimiter not found")
		}
		startIdx += len(start)
		endIdx := bytes.Index(body[startIdx:], []byte(end))
		if endIdx == -1 {
			return -1, fmt.Errorf("end delimiter not found")
		}
		value := body[startIdx : startIdx+endIdx]
		return strconv.Atoi(string(value))
	}

	price, err := extractValue(body, `data-expected-price="`, `"`)
	if err != nil {
		return -1, -1, -1, -1
	}

	productID, err := extractValue(body, `data-product-id="`, `"`)
	if err != nil {
		return -1, -1, -1, -1
	}

	sellerID, err := extractValue(body, `data-expected-seller-id="`, `"`)
	if err != nil {
		return -1, -1, -1, -1
	}

	userAssetID, err := extractValue(body, `data-lowest-private-sale-userasset-id="`, `"`)
	if err != nil {
		return -1, -1, -1, -1
	}

	return price, productID, sellerID, userAssetID
}

type SingleProxy struct {
	IP   string
	Port string
}

// FromFile reads the file contents line by line from the provided path
// and parses each line into LimitedInfo. Each line should follow the syntax: id<int>, price<int>.
func ParseProxies(path string) ([]SingleProxy, error) {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Use bufio.Scanner for efficient line-by-line reading
	scanner := bufio.NewScanner(file)
	var infos []SingleProxy

	for scanner.Scan() {
		// Read the line
		line := scanner.Text()

		if len(line) == 0 {
			log.Warn("Empty line detected in limiteds file, remove it or it can cause slower parsing times.")
			continue
		}

		// Split the line into parts (id and price)
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line format: %s", line)
		}

		// Append parsed struct to slice
		infos = append(infos, SingleProxy{
			IP:   strings.TrimSpace(parts[0]),
			Port: strings.TrimSpace(parts[1]),
		})
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return infos, nil
}
