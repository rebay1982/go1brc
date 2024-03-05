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

Let's scrap the idea and try to measure parsing and data collection separately and chalk iteration 2 as a failed
experiment.
