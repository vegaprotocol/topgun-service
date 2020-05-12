package util

import (
	"bufio"
	"encoding/csv"
	"io"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func LoadPartiesFromCsv(excludeFile string) (map[string]byte, error) {
	if len(excludeFile) <= 0 {
		return nil, errors.New("No excluded parties file specified")
	}
	csvFile, _ := os.Open(excludeFile)
	reader := csv.NewReader(bufio.NewReader(csvFile))
	log.Infof("Parsing excluded parties csv file: %s", excludeFile)
	result := map[string]byte{}
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		if len(line) == 0 {
			// Empty line
			continue
		}
		if len(line) == 1 && line[0] == "Party" {
			// Ignore header
			continue
		}
		if len(line) == 2 && line[1] == "Description" {
			// Ignore header
			continue
		}
		result[line[0]] = 0xF
	}
	log.Infof("Found %d excluded parties in file specified", len(result))
	return result, nil
}
