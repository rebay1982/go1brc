# go1brc
1BRC challenge for fun.

## Iteration 1 (it1)
Basic implementation, nothing complicated and no optimization.

```
real    1m49.953s
user    1m49.002s
sys     0m4.124s
```

## Iteration 2 (it2)
Added timing metrics, slow part is the scanning.
```
Scan elapsed time: 1m51.292439661s
Sort elapsed time: 41.917µs
Print elapsed time: 611.694µs
```

Added a fan-out fan-in model to the parsing. This seems to have slowed down the performance of the application.
Wasn't expecting this but I'm assuming that there might be some slowdown caused by a lot of extra allocations in the
channels.

Let's scrap the idea and use real profiling instead of going with gut feeling optimizations.


## Iteration 3 (it3)
Added profiling using `pprof` and used [Brendan Gregg's scripts](https://github.com/brendangregg/FlameGraph) to build a
flame graph from the profiling information.

Because I just can't remember things I don't do on a daily basis, this is the commands I have used to generate the 
profiling information and then create the flame graph:

Once the profiling code is in, it generates a `.prof` profile file that we use to generate a new file containing raw
profiling information. The second command is using Brendan's scripts to generate the interactive flame graph (viewable
through your web browser)
```
$ go tool pprof -raw -output=profile.txt ./profile.prof
$ perl ./stackcolapse-go.pl profile.txt | perl ./flamegraph.pl > it1.svg
```

Iteration 1's flamegraph looks
like this:
![Iteration 1 flame graph](/profiling/it1.svg)

Replaced "strings.Split" with a custom GetSplit function:
```
func GetSplit(line string) (string, string) {

	length := len(line)
	if line[length - 5] == ';' {
		return line[:length - 5], line[length - 4:]

	} else if line[length - 4] == ';' {
		return line[:length - 4], line[length - 3:]
	}

	return line[:length - 6], line[length - 5:]
}
```
Here are the results:
```
real    1m16.697s
user    1m15.372s
sys     0m3.261s
```

Next step for Iteration 3 is to change how we're parsing the floats. The flame graph indicates that a considerable 
amount of time is being spent in `strconv.ParseFloat`. Since we know it's always a single fractional digit, the string 
always ends with `.X` where X is the digit. We can parse the left side of the `.` to an int, multiply by 10, and add
the fractional digit to that number.

To get the correct result in the end, we'll need to divide the value by 10. The assumption is that this will be quicker
than using a function designed to parse arbitrary 64 bit floats. Two functions have been created for this iteration.
One which uses the `strconv.Atoi` function, and one which has a custom implementation.

The implementation suing `strconv.Atoi` shaved off about 17 extra seconds off processing vs using the
`strconv.PasrseFloat` version:
```
real    0m59.459s
user    0m58.283s
sys     0m2.794s
```

This is what the flame graph looks like with this modification.
![Iteration 3-2 flame graph](/profiling/it3-2.svg)

Trying a custom function for parsing the float without the use of `strconv.Atoi` was attempted with the following
results:
```
real    0m58.221s
user    0m56.944s
sys     0m2.875s
```
Just slightly faster, 18 seconds faster than the previous iteration.
![Iteration 3-3 flame graph](/profiling/it3-3.svg)

An other issue we can see from Iteration 1's flame graph is the considerable time spent in the hash map access function
`mapaccess2_faststr`. Here, the map is not necessarily huge but we're accessing it very often (1 billion times) which
will accentuate the use of hashing.

We will attack this in a future iteration.


## Iteration 4 (it4)
Attacking the hashing problem, a basic hash table structure was created using a non cryptographic hashing fucntion:
[FNV-1a](https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function).

This provided to be faster and a useful optimization, bringing the time down to a little less than 42seconds.
```
real    0m41.774s
user    0m41.698s
sys     0m2.447s
```

Here is the flame graph associated with this iteration:
![Iteration 4 flame graph](/profiling/it4.svg)

We can observe that hashing the value twice for the get and then the "Add" is costly and we should look to refactor the
code to reduce the amount of hashing.

A second iteration was run while hashing the string only once and to my surprise didn't bring any time improvement. We
can see this on the flame graph below:
![Iteration 4-1 flame graph](/profiling/it4-1.svg)

## Iteration 5 (it5)
One last piece I'm interested in optimizing is the reading of the file. We're reading the file line by line and there is
probably a better way to do this.

Instead of reading the file line per line, I decided to go with reading large chunks of the file into a large memory
buffer. This preallocated memory is then broken down into "valid" chunks, containing only full lines, and a "left over"
chunk containing the tailing contents of the original large chunk. This is to be pre-pended to the next read chunk.

I have also tried multiple buffer sizes to see how it affects performance. Let's have a look at the results.

| Size   | Execucation time          |
|--------|---------------------------|
| 4Mb    | 36.210s, 37.111s, 37.912s |
| 8Mb    | 34.502s, 36.030s, 35.907s |
| 16Mb   | 33.496s, 34.093s, 33.988s |
| 32Mb   | 32.186s, 32.884s, 33.896s |
| 64Mb   | 32.160s, 32.554s, 32.958s |
| 128Mb  | 31.626s, 32,436s, 32.531s |
| 256Mb  | 34.067s, 32.634s, 33.858s |

This is really interesting. The hypothesis for this is that smaller chunk sizes will require more iterations and memory
allocations to get the work done. This creates more work for the garbage collector and it will hinder the application's
performance. The other really interesting part is that we actually hit a minimum at 128Mb chunk sizes and performance
decreases as we go for larger chunk sizes like 256Mb. We understand that reading larger chunks becomes less efficient.
Let's look at flame graphs for the 4Mb, the 128Mb, and the 256Mb runs to try to understand these two observations.

4Mb run
![Iteration 5-4mb flame graph](/profiling/it5-4mb.svg)

128Mb run
![Iteration 5-128mb flame graph](/profiling/it5-128mb.svg)

256Mb run
![Iteration 5-256mb flame graph](/profiling/it5-256mb.svg)

New development that I hadn't anticipated! I didn't notice the fact that the hashing function had some collisions. I
assumed so because the output results didn't align with the results.txt that was generated with the original data set.
Linear probing was implemented to solve this issue and avoid collisions in the hashing. The computed values are now 
back to being accurate and align nicely with the reference output file.

The nature of the linear probing added some extra computing cycles so let's see how this affected performance.

| Size   | Execucation time          |
|--------|---------------------------|
| 4Mb    | 38.112s, 38.127s, 38.723s |
| 8Mb    | 37.729s, 37.833s, 38.119s |
| 16Mb   | 36.392s, 36.204s, 36.697s |
| 32Mb   | 34.978s, 34.866s, 35.194s |
| 64Mb   | 34.965s, 34.750s, 35.067s |
| 128Mb  | 35.250s, 35.657s, 34.974s |
| 256Mb  | 34.748s, 34.335s, 34.037s |
| 512Mb  | 34.238s, 34.337s, 34.541s |

This is interesting. The actual "best value" for the buffer size changed. Passed 32mb, there isn't much of a possitive 
nor a negative impact on performance -- it pretty much flat lines. Let's see how the flame graph looks like now.

32Mb run
![Iteration 5-32mb flame graph](/profiling/it5-2-32.svg)

Results are somewhat the same, but we are spending more time in the get function which is expected because of the linear
probing. An other thing to note, I'm now using the Strings.Split function to split the strings. This is a bit
"backwards" in the sense that we previously gotten rid of this for spliting temperatures and created out very own custom
split function tailored specifically to our usecase because it was predictable: temperatures are either 4, 5, or 6
characters in length, 5 being the most common one. Since the names of the stations are arbitrary, it will be difficult 
to gain much from writting our own custom split function, I think.

Next step, would be to parallelize the code and make it function on multiple cores.
