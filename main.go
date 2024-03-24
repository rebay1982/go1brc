package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
)
const (
  hashOffset = 14695981039346656037
	hashPrime = 1099511628211
	hashSize = 1 << 17
)

type measurement struct {
	min, max, sum, count int 
}

type HashEntry struct {
	key []byte
	measurement *measurement
}

func Hash(station string) uint64 {
	hash := uint64(hashOffset)

	for _, c := range station {
		hash ^= uint64(c)
		hash *= hashPrime
	}

	return hash
}

type HashMap struct {
	entries []HashEntry
}

func NewHashMap() *HashMap {
	return &HashMap{entries: make([]HashEntry, hashSize)}
}

func (m *HashMap) Add(key string, measurement *measurement) *HashEntry {
	hash := Hash(key) 
	entry := &HashEntry{
		key: []byte(key),
		measurement: measurement,
	}

	m.entries[hash % uint64(hashSize)] = *entry
	return entry
}

func (m *HashMap) Get(key string) *HashEntry {
	return &m.entries[Hash(key) % uint64(hashSize)]
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

	results := NewHashMap()
	scanner := bufio.NewScanner(file)

	// Unfortunate but needed for sorting later.
	stations := make([]string, 0, 500)
	for scanner.Scan() {
		line := scanner.Text()

		station, stationTemp := GetSplit(line)
		temp := ParseTemp(stationTemp)

		e := results.Get(station)
		if e.measurement == nil {
			e = results.Add(station, &measurement{min: temp, max: temp, sum: temp})
			stations = append(stations, station)
		} else {
			e.measurement.min = min(e.measurement.min, temp)
			e.measurement.max = max(e.measurement.max, temp)
			e.measurement.sum += temp
		}
		e.measurement.count++
	}

	// Sort stuff
	sort.Strings(stations)

	// Output stuff
	fmt.Print("{")
	nbKeys := len(stations) - 1
	for i, k := range stations {
		e := results.Get(k)

		if i != nbKeys {
			fmt.Printf("%s=%.1f/%.1f/%.1f, ", k, float64(e.measurement.min)/10, float64(e.measurement.sum)/float64(e.measurement.count)/10, float64(e.measurement.max)/10) 
		} else {
			fmt.Printf("%s=%.1f/%.1f/%.1f}\n", k, float64(e.measurement.min)/10, float64(e.measurement.sum)/float64(e.measurement.count)/10, float64(e.measurement.max)/10) 
		}
	}
}
