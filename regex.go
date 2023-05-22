package janet

import (
	"fmt"
	"regexp"
	"strings"
)

type karmaRegex struct {
	user, autocomplete, explicitAutocomplete, goodPoints, badPoints, reason string
}

var karmaReg = &karmaRegex{
	user:                 `@??((?:<@)??\w[A-Za-z0-9_\-@<>]*?)`,
	autocomplete:         `:?? ??`,
	explicitAutocomplete: `(?:: )??`,
	goodPoints:           `([\+]{2,})`,
	badPoints:            `([\-]{2,})`,
	reason:               `(?:(?: for) +(.*))?`,
}

func (r *karmaRegex) MatchGive() *regexp.Regexp {
	expression := fmt.Sprintf(
		"(?:%s)|(?:%s)",
		strings.Join(
			[]string{
				"^",
				r.user,
				r.autocomplete,
				r.goodPoints,
				r.reason,
				"$",
			},
			"",
		),
		strings.Join(
			[]string{
				`\s+`,
				r.user,
				r.explicitAutocomplete,
				r.goodPoints,
				r.reason,
				"$",
			},
			"",
		),
	)

	return regexp.MustCompile(expression)
}

func (r *karmaRegex) MatchTake() *regexp.Regexp {
	expression := fmt.Sprintf(
		"(?:%s)|(?:%s)",
		strings.Join(
			[]string{
				"^",
				r.user,
				r.autocomplete,
				r.badPoints,
				r.reason,
				"$",
			},
			"",
		),
		strings.Join(
			[]string{
				`\s+`,
				r.user,
				r.explicitAutocomplete,
				r.badPoints,
				r.reason,
				"$",
			},
			"",
		),
	)

	return regexp.MustCompile(expression)
}

func (r *karmaRegex) MatchMotivate() *regexp.Regexp {
	expression := strings.Join(
		[]string{
			`^(?:\?|!)m +`,
			r.user,
			r.autocomplete,
			"$",
		},
		"",
	)

	return regexp.MustCompile(expression)
}

func (r *karmaRegex) MatchQuery() *regexp.Regexp {
	expression := strings.Join(
		[]string{
			`^`,
			r.user,
			r.autocomplete,
			"==",
			"$",
		},
		"",
	)

	return regexp.MustCompile(expression)
}

func (r *karmaRegex) MatchThrowback() *regexp.Regexp {
	expression := strings.Join(
		[]string{
			`^janet(?:bot)? (?:throwback) ?(`,
			r.user,
			r.autocomplete,
			`)?$`,
		},
		"",
	)

	return regexp.MustCompile(expression)
}
