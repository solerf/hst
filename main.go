package main

import "github.com/alecthomas/kong"

func main() {
	kCtx := kong.Parse(&hst{}, kong.UsageOnError())
	kCtx.FatalIfErrorf(kCtx.Run())
}
