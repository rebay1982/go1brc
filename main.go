package main

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
	"strings"
)

const (
	hashOffset = 14695981039346656037
	hashPrime  = 1099511628211
	hashSize   = 1 << 17
)

type measurement struct {
	min, max, sum, count int
}

type HashMap struct {
	entries []*HashEntry
}

func NewHashMap() *HashMap {
	return &HashMap{entries: make([]*HashEntry, hashSize)}
}

func (m *HashMap) Add(hash uint64, key string, measurement *measurement) *HashEntry {
	entry := &HashEntry{
		key:         []byte(key),
		measurement: measurement,
	}

	pos := hash % uint64(hashSize)
	for {
		if m.entries[pos] == nil {
			break
		}
		pos++
	}
	m.entries[pos] = entry
	return entry
}

func (m *HashMap) Get(hash uint64, key string) *HashEntry {
	// Linear search on a key colission
	pos := hash % uint64(hashSize)
	for {
		entry := m.entries[pos]

		if entry != nil {
			if string(entry.key) == key {
				return entry

			} else {
				pos++
			}
		} else {
			return entry
		}
	}
}

type HashEntry struct {
	key         []byte
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

func GetSplit(line string) (string, string) {
	length := len(line)
	if line[length-5] == ';' {
		return line[:length-5], line[length-4:]

	} else if line[length-4] == ';' {
		return line[:length-4], line[length-3:]
	}

	return line[:length-6], line[length-5:]
}

func ParseTemp(temp string) int {
	length := len(temp)
	sum := 0
	factor := 1
	neg := 1

	stopIter := 0
	if temp[0] == '-' {
		neg = -1
		stopIter++
	}

	for i := length - 3; i >= stopIter; i-- {
		sum += int(temp[i]-'0') * factor
		factor *= 10
	}

	d := temp[length-1] - '0'

	return (sum*10)*neg + int(d)*neg
}

func processChunk(chunk, prevLeftoverChunk []byte, chunkSize int) (completeChunk, leftoverChunk []byte) {
	firstCR := -1
	lastCR := 0

	// Not sure about this, there can be better ways that iterate less over the chunk like starting from the end to find
	// the last CR, etc
	for i, char := range chunk {
		if char == '\n' {
			if firstCR == -1 {
				firstCR = i
				break
			}
		}
	}

	for i := chunkSize - 1; i > firstCR; i-- {
		if chunk[i] == '\n' {
			lastCR = i
			break
		}
	}

	// Copy whatever is left in the buffer.
	if firstCR != 1 {
		completeChunk = append(prevLeftoverChunk, chunk[:lastCR+1]...)
		leftoverChunk = append(leftoverChunk, chunk[lastCR+1:]...)
	} else {
		leftoverChunk = append(prevLeftoverChunk, chunk[:chunkSize]...)
	}

	return
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

	// Unfortunate but needed for sorting later.
	results := NewHashMap()
	stations := make([]string, 0, 500)

	const bufferSize = 32 * 1024 * 1024
	buf := make([]byte, bufferSize)
	leftover := make([]byte, 0, bufferSize)

	for {
		nbChars, err := file.Read(buf)
		if err != nil {
			break
		}

		if nbChars > 0 {
			validChunk, newLeftover := processChunk(buf, leftover, nbChars)
			leftover = newLeftover

			if len(validChunk) > 0 {
				// Process chunk
				lines := strings.Split(string(validChunk), "\n")
				lines = lines[:len(lines)-1]
				for _, line := range lines {
					station, stationTemp := GetSplit(line)
					stationHash := Hash(station)
					temp := ParseTemp(stationTemp)

					e := results.Get(stationHash, station)
					if e == nil {
						e = results.Add(stationHash, station, &measurement{min: temp, max: temp, sum: temp})
						stations = append(stations, station)
					} else {
						e.measurement.min = min(e.measurement.min, temp)
						e.measurement.max = max(e.measurement.max, temp)
						e.measurement.sum += temp
					}
					e.measurement.count++
				}
			}
		}
	}

	// Sort stuff
	// TODO: Find a way to append the hash to the station name so that we can pick it up and avoid hashing the name again.
	sort.Strings(stations)

	// Output stuff
	fmt.Print("{")
	nbKeys := len(stations) - 1
	for i, k := range stations {
		e := results.Get(Hash(k), k)

		if i != nbKeys {
			fmt.Printf("%s=%.1f/%.1f/%.1f, ", k, float64(e.measurement.min)/10, float64(e.measurement.sum)/float64(e.measurement.count)/10, float64(e.measurement.max)/10)
		} else {
			fmt.Printf("%s=%.1f/%.1f/%.1f}\n", k, float64(e.measurement.min)/10, float64(e.measurement.sum)/float64(e.measurement.count)/10, float64(e.measurement.max)/10)
		}
	}
}
