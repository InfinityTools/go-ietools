# Go-Infinity-Tools
*Collection of functions for modifying resources of Infinity Engine games, written in Go.*

## About

*go-infinity-tools* provides functionality to access and modify structured or textual resource types commonly found in Infinity Engine games, such as Baldur's Gate or Icewind Dale.

The package is written in [Go](https://golang.org/). It currently provides three sub-packages: *buffers*, *pvrz* and *tables*.

Package *ietools* contains several helpful constants and functions that are used by the sub-packages. External dependencies: `golang.org/x/text/encoding/charmap`.

Package *buffers* contains a set of functions for reading, creating or modifying structured resources. It is loosely based on a subset of functions provided by [WeiDU](http://www.weidu.org/%7Ethebigg/README-WeiDU.html). The package has no external dependencies.

Package *pvrz* implements a high-level PVR/PVRZ texture manager. External dependencies: `github.com/InfinityTools/squish` (see [go-squish](http://github.com/InfinityTools/go-squish) for more information).

Package *tables* allows you to read and modify table-like content in text format, such as 2DA or IDS. Functionality has also been inspired by WeiDU. External dependencies: `golang.org/x/text/encoding/charmap`.

## Building

*go-infinity-tools* package path is `github.com/InfinityTools/ietools`. Main package and each sub-package can be built via `go build`.

You may have to specify additional options, e.g. via `CGO_LDFLAGS` environment variable, to compile the *pvrz* package.

## License

*go-infinity-tools* and all sub-packages are released under the BSD 2-clause license. See LICENSE for more details.
