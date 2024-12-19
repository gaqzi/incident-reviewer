package web

import "embed"

var (
	//go:embed all:templates/*
	templates embed.FS
)
