//go:build ruleguard
// +build ruleguard

package ruleguard

import (
	"github.com/quasilyte/go-ruleguard/dsl"
)

func execContext(m dsl.Matcher) {
	m.Import("github.com/gaqzi/incident-reviewer/internal/reviewing")

	// Extremely simplistic rule: if we ever assign the reviewing.Review
	// object into a map directly we're doing it wrong. It should ideally
	// be _any_ object from reviewing.* and only in the reviewing.http
	// package, but I don't know enough about ruleguard to handle that yet.
	//
	// TODO: make this matcher handle the above.
	//
	// To work on this use ruleguard directly: ruleguard -rules ruleguard/rules-dont-pass-domain-objects-to-templates.go internal/reviewing/http/handler.go
	m.Match(`map[string]any{$*_, $key: $val, $*_}`).
		Where(m["val"].Type.Is(`reviewing.Review`)).
		Report(`passing reviewing.Review into a template's SetData map. Use: convertToHttpObject($val)`)
}
