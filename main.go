package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {

	filename := "measurements.txt"
	file, err := os.Open(filename)

	if err != nil {
		log.Fatalf("Failed to load file %s, err = %v\n", filename, err)
		os.Exit(1)
	}
	defer file.Close()

	type measurement struct {
		min, max, sum float64
		count int
	}

	results := make(map[string]measurement)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		splits := strings.Split(line, ";")

		station := splits[0]
		temp, err := strconv.ParseFloat(splits[1], 64)
		if err != nil {
			log.Fatalf("Failed to parse float [%s], err = %v\n", splits[1], err)
			os.Exit(1)
		}

		r, ok := results[station]
		if !ok {
			r.min = temp
			r.max = temp
			r.sum = temp
		} else {
			r.min = min(r.min, temp)
			r.max = max(r.max, temp)
			r.sum += temp
		}
		r.count++
	}
}


