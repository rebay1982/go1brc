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
