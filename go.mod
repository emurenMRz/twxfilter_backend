module github.com/emurenMRz/twxfilter_backend

go 1.21.4

require (
	github.com/lib/pq v1.10.9
	mediadata v0.0.0
	router v0.0.0
	diffhash v0.0.0
)

replace mediadata => ./mod/mediadata

replace router => ./mod/router

replace diffhash => ./mod/diffhash
