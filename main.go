package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type stationData struct {
	min, max, sum float64
	count int
}
type measurement struct	{
	name string
	temp float64
}

type timing struct {
	start time.Time
	elapsed time.Duration
}

func (t timing) GetElapsedMS() time.Duration {
	return t.elapsed// /time.Millisecond
}

func workGen(scanner *bufio.Scanner) <- chan string {
	genOut := make(chan string)
	go func() {
		for scanner.Scan() {
			genOut <- scanner.Text()
		}
		close(genOut)
	}()
	return genOut
}

// Line parser
func measurementParser(in <- chan string) <- chan measurement {
	parseOut := make(chan measurement)
	go func() {
		for line := range in {
			splits := strings.Split(line, ";")

			temp, _ := strconv.ParseFloat(splits[1], 64)
			m := measurement{name: splits[0], temp: temp}
			parseOut <-m
		}
		close(parseOut)
	}()
	return parseOut
}

func readFile(file *os.File, numParsers int) map[string]*stationData {
	// Work generator -- generates lines that need to be processed
	genOut := workGen(bufio.NewScanner(file))

	// Parsers -- receives lines from the generators and parse the work.
	parseOut := []<-chan measurement{}
	for i := 0; i < numParsers; i++ {
		parseOut = append(parseOut, measurementParser(genOut))
	}

	// Everything is lined up now, all you have to do is implement the merge sink and do line 79 processing on the 
	// parser output
	var wg sync.WaitGroup
	aggOutput := make(chan measurement)
	sink := func(in <- chan measurement)	{
		for m := range in {
			aggOutput <- m
		}
		wg.Done()
	}
	wg.Add(numParsers)

	for _, c := range parseOut {
		go sink(c)	
	}

	go func() {
		wg.Wait()
		close(aggOutput)
	}()

	results := make(map[string]*stationData)
	for m := range aggOutput {
		station := m.name
		temp := m.temp
		r, ok := results[station]
		if !ok {
			r = &stationData{min: temp, max: temp, sum: temp}
			results[station] = r
		} else {
			r.min = min(r.min, temp)
			r.max = max(r.max, temp)
			r.sum += temp
		}
		r.count++
	}
	return results
}

func main() {
	filename := "measurements.txt"
	file, err := os.Open(filename)

	if err != nil {
		log.Fatalf("Failed to load file %s, err = %v\n", filename, err)
		os.Exit(1)
	}
	defer file.Close()

	scanTiming := timing{start: time.Now()}
	results := readFile(file, 5)
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
