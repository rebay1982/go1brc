package main

import (
	"bufio"
	"fmt"	
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type measurement struct {
	min, max, sum float64
	count int
}

type timing struct {
	start time.Time
	elapsed time.Duration
}

func (t timing) GetElapsedMS() time.Duration {
	return t.elapsed// /time.Millisecond
}

func main() {

	filename := "measurements.txt"
	file, err := os.Open(filename)

	if err != nil {
		log.Fatalf("Failed to load file %s, err = %v\n", filename, err)
		os.Exit(1)
	}
	defer file.Close()

	results := make(map[string]*measurement)
	scanner := bufio.NewScanner(file)

	scanTiming := timing{start: time.Now()}
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
			r = &measurement{min: temp, max: temp, sum: temp}
			results[station] = r
		} else {
			r.min = min(r.min, temp)
			r.max = max(r.max, temp)
			r.sum += temp
		}

		r.count++
	}
	scanTiming.elapsed = time.Since(scanTiming.start)

	// Sort stuff
	sortTiming := timing{start: time.Now()}
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sortTiming.elapsed = time.Since(sortTiming.start)

	// Output stuff
	printTiming := timing{start: time.Now()}
	fmt.Print("{")
	nbKeys := len(keys) - 1
	for i, k := range keys {
		result := results[k]

		if i != nbKeys {
			fmt.Printf("%s=%.1f/%.1f/%.1f, ", k, result.min, result.sum/float64(result.count), result.max) 
		} else {
			fmt.Printf("%s=%.1f/%.1f/%.1f}\n", k, result.min, result.sum/float64(result.count), result.max) 
		}
	}
	printTiming.elapsed = time.Since(printTiming.start)

	fmt.Println()
	fmt.Printf("Scan elapsed time: %s\n", scanTiming.GetElapsedMS())
	fmt.Printf("Sort elapsed time: %s\n", sortTiming.GetElapsedMS())
	fmt.Printf("Print elapsed time: %s\n", printTiming.GetElapsedMS())
}
