package main

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
)

const (
	hashOffset        = 14695981039346656037
	hashPrime         = 1099511628211
	hashSize          = 1 << 17
	chunkSize         = 1
	chunkChannelSize  = 1024
	resultChannelSize = 1024
	workerPoolSize    = 12
	filename          = "measurements.txt"
	tmpSliceSize      = 80000
)

type measurement struct {
	min, max, sum, count int
}

type tempReading struct {
	hash    uint64
	station string
	temp    int
}

// chunkProcessor This is the chunk processing function, which is essentially our worker from the worker pool.
func chunkWorker(wg *sync.WaitGroup, chunks <-chan []byte, results chan<- []tempReading) {
	defer wg.Done()

	for chunk := range chunks {
		// 80000 because the %age of chunks that have more than 80k lines in them is less than 10%.
		//   This is a compromise between always allocating way too much memory and requiring garbage collection vs spending
		//	 time increasing slice capacity.
		tmpSlice := make([]tempReading, 0, tmpSliceSize)

		// Process chunk
		lines := strings.Split(string(chunk), "\n")
		lines = lines[:len(lines)-1]
		for _, line := range lines {
			tmp := tempReading{}
			var stationTemp string

			tmp.station, stationTemp = getSplit(line)
			tmp.hash = Hash(tmp.station)
			tmp.temp = parseTemp(stationTemp)

			tmpSlice = append(tmpSlice, tmp)
		}

		results <- tmpSlice
	}
}

// resultsAggregator The resultsAggregator function's purpose is to aggreate results from the workers and compute them
//
//	into the measurements hashmap. The waitgroup is to signify that aggregation is complete and we can move forward with
//	sorting the results.
func resultsAggregator(wg *sync.WaitGroup, results <-chan []tempReading, resultsMap *HashMap, stationsOut chan<- []string) {
	defer wg.Done()
	stations := make([]string, 0, 500)

	for tmpSlice := range results {
		for _, tmp := range tmpSlice {
			e := resultsMap.Get(tmp.hash, tmp.station)
			if e == nil {
				e = resultsMap.Add(tmp.hash, tmp.station, &measurement{min: tmp.temp, max: tmp.temp, sum: tmp.temp})
				stations = append(stations, tmp.station)
			} else {
				e.measurement.min = min(e.measurement.min, tmp.temp)
				e.measurement.max = max(e.measurement.max, tmp.temp)
				e.measurement.sum += tmp.temp
			}
			e.measurement.count++
		}
	}

	sort.Strings(stations)
	stationsOut <- stations
}

func main() {
	proffile, err := os.Create("./profile.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(proffile)
	defer pprof.StopCPUProfile()

	// Create the channels for the workers and aggregator.
	chunks := make(chan []byte, chunkChannelSize)
	results := make(chan []tempReading, resultChannelSize)
	stationsIn := make(chan []string, 1)

	// Create the worker pool
	var workerWG sync.WaitGroup
	workerWG.Add(workerPoolSize)

	for i := 0; i < workerPoolSize; i++ {
		go chunkWorker(&workerWG, chunks, results)
	}

	// once all workers are done, we close the results channel
	go func() {
		workerWG.Wait()
		close(results)
	}()

	// Create the aggregator
	resultsMap := NewHashMap()
	var aggregatorWG sync.WaitGroup
	aggregatorWG.Add(1)

	go resultsAggregator(&aggregatorWG, results, resultsMap, stationsIn)

	file, err := os.Open(filename)

	if err != nil {
		log.Fatalf("Failed to load file %s, err = %v\n", filename, err)
		os.Exit(1)
	}
	defer file.Close()

	const bufferSize = chunkSize * 1024 * 1024
	buf := make([]byte, bufferSize)
	leftover := make([]byte, 0, bufferSize)

	// Start reading the file and chunking stuff up for workers to consume.
	for {
		nbChars, err := file.Read(buf)
		if err != nil {
			close(chunks)
			break
		}

		if nbChars > 0 {
			validChunk, newLeftover := processChunk(buf, leftover, nbChars)
			leftover = newLeftover

			if len(validChunk) > 0 {
				chunks <- validChunk
			}
		} else {
			// Done creating chunks
			fmt.Println("Done reading file...")
			close(chunks)
			break
		}
	}

	// Wait until the aggregator is done processing results.
	aggregatorWG.Wait()

	// Sort stuff
	stations := <-stationsIn

	// Output stuff
	fmt.Print("{\n")
	nbKeys := len(stations) - 1
	for i, k := range stations {
		e := resultsMap.Get(Hash(k), k)

		if i != nbKeys {
			fmt.Printf("%s=%.1f/%.1f/%.1f, ", k, float64(e.measurement.min)/10, float64(e.measurement.sum)/float64(e.measurement.count)/10, float64(e.measurement.max)/10)
		} else {
			fmt.Printf("%s=%.1f/%.1f/%.1f}\n", k, float64(e.measurement.min)/10, float64(e.measurement.sum)/float64(e.measurement.count)/10, float64(e.measurement.max)/10)
		}
	}
}

func getSplit(line string) (string, string) {
	length := len(line)
	if line[length-5] == ';' {
		return line[:length-5], line[length-4:]

	} else if line[length-4] == ';' {
		return line[:length-4], line[length-3:]
	}

	return line[:length-6], line[length-5:]
}

func parseTemp(temp string) int {
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

// Hashmap stuff
type HashEntry struct {
	key         string
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

func NewHashMap() *HashMap {
	return &HashMap{entries: make([]*HashEntry, hashSize)}
}

type HashMap struct {
	entries []*HashEntry
}

func (m *HashMap) Add(hash uint64, key string, measurement *measurement) *HashEntry {
	entry := &HashEntry{
		key:         key,
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
