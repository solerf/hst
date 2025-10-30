package main

import "github.com/alecthomas/kong"

func main() {
	kong.UsageOnError()
	kCtx := kong.Parse(&cmd)
	kCtx.FatalIfErrorf(kCtx.Run())
}
