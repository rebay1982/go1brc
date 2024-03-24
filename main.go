package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
	"strconv"
)

type measurement struct {
	min, max, sum, count int 
}

func GetSplit(line string) (string, string) {
	length := len(line)
	if line[length - 5] == ';' {
		return line[:length - 5], line[length - 4:]

	} else if line[length - 4] == ';' {
		return line[:length - 4], line[length - 3:]
	}

	return line[:length - 6], line[length - 5:]
}

func ParseTempStrconv(temp string) int {
	length := len(temp)

	neg := 1
	startIndex := 0
	if temp[0] == '-' {
			neg =	-1 
			startIndex++
	}
	i, _ := strconv.Atoi(temp[startIndex:length-2])
	d := temp[length - 1] - '0'

	return (i * 10)*neg + int(d)*neg
}

func ParseTemp(temp string) int {
	length := len(temp)
	sum := 0
	factor := 1
	neg := 1

	stopIter := 0
	if temp[0] == '-' {
			neg =	-1 
			stopIter++
	}

	for i := length - 3; i >= stopIter; i-- {
		sum += int(temp[i] - '0') * factor
		factor *= 10
	}
	
	d := temp[length - 1] - '0'

	return (sum * 10)*neg + int(d)*neg
}

func main() {
	proffile, err := os.Create("./profile.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(proffile)
	defer pprof.StopCPUProfile()

	filename := "measurements.txt"
	file, err := os.Open(filename)

	if err != nil {
		log.Fatalf("Failed to load file %s, err = %v\n", filename, err)
		os.Exit(1)
	}
	defer file.Close()

	results := make(map[string]*measurement)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		station, stationTemp := GetSplit(line)
		temp := ParseTemp(stationTemp)
		//temp := ParseTempStrconv(stationTemp)

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

	// Sort stuff
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Output stuff
	fmt.Print("{")
	nbKeys := len(keys) - 1
	for i, k := range keys {
		result := results[k]

		if i != nbKeys {
			fmt.Printf("%s=%.1f/%.1f/%.1f, ", k, float64(result.min)/10, float64(result.sum)/float64(result.count)/10, float64(result.max)/10) 
		} else {
			fmt.Printf("%s=%.1f/%.1f/%.1f}\n", k, float64(result.min)/10, float64(result.sum)/float64(result.count)/10, float64(result.max)/10) 
		}
	}
}
