package main

import "game-scouter-api/internal/data"

type Session struct {
	data    any
	written bool
}

var SessKey = data.ContextKey("SessKey")
