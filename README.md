# koron/lsgotype

[![GoDoc](https://godoc.org/github.com/koron/lsgotype?status.svg)](https://godoc.org/github.com/koron/lsgotype)
[![Actions/Go](https://github.com/koron/lsgotype/workflows/Go/badge.svg)](https://github.com/koron/lsgotype/actions?query=workflow%3AGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/koron/lsgotype)](https://goreportcard.com/report/github.com/koron/lsgotype)

list go types.

## Example

This generates `goExtraType` syntax groups of Vim from `$GOROOT/src`

```console
$ go run github.com/koron/lsgotype -mode syntax | grep -v skipped > goextra.vim
```
