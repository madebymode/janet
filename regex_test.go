package janet

import (
	"regexp"
	"testing"
)

type regexTestSuite map[bool][]string
type regexPattern struct {
	Regex *regexp.Regexp
	Name  string
}

var regexTests = map[regexPattern]regexTestSuite{
	regexPattern{
		Regex: karmaReg.MatchGive(),
		Name:  "karma operations",
	}: regexTestSuite{
		true: []string{
			"user++",
			"user--",
			"user+++++++",
			"user-------",
			"@user---",
			"user+++ for reason",
			"user--- because why not",
			"user: ---- autocomplete test",
			"user ++++ another autocomplete test",
			"<@U147391>++++ slack formatting test",
			"middle of the sentence--",
			"middle of the sentence-- for karma reasons",
			"middle of the sentence: ++++ for karma reasons",
		},
		false: []string{
			"user+-",
			"@user-+",
			"middle of the sentence -- test",
			"middle of the sentence ++",
			"middle of the sentence ---- another test",
		},
	},
	regexPattern{
		Regex: karmaReg.MatchQuery(),
		Name:  "print current karma points",
	}: regexTestSuite{
		true: []string{
			"user==",
			"@user==",
			"<@U1384>==",
		},
		false: []string{
			"user=",
			"user===",
			"@user=",
			"@user===",
			"middle of the sentence user==",
		},
	},
	regexPattern{
		Regex: karmaReg.MatchMotivate(),
		Name:  "motivate.im",
	}: regexTestSuite{
		true: []string{
			"!m user",
			"?m user",
			"?m     <@U1384>",
		},
		false: []string{
			"?m user for work",
			"middle of the sentence ?m user",
			"?muser",
			"!!muser",
		},
	},
	regexPattern{
		Regex: regexps.Leaderboard,
		Name:  "leaderboard",
	}: regexTestSuite{
		true: []string{
			"karma highscores",
			"janet top 10",
			"janet top 1001",
			"janet top ",
		},
		false: []string{
			"janet top 913f",
			"janet karma highscores",
		},
	},
	regexPattern{
		Regex: regexps.SlackUser,
		Name:  "slack user",
	}: regexTestSuite{
		true: []string{
			"<@U1934>",
			"<@P1934>",
			"<@whatever>",
		},
		false: []string{
			"<@user",
			"<user>",
			"user>",
			"<@>",
		},
	},
	regexPattern{
		Regex: regexps.URL,
		Name:  "janet web ui",
	}: regexTestSuite{
		true: []string{
			"janet web",
			"karma link",
			"janet link",
		},
		false: []string{
			"janet web 194",
		},
	},
	regexPattern{
		Regex: regexps.Throwback,
		Name:  "janet throwback",
	}: regexTestSuite{
		true: []string{
			"janet throwback <@U3494519>",
			"janet throwback <@B34839>",
			"janet throwback @name",
			"janet throwback user",
			"janet throwback",
		},
	},
}

func TestRegexes(t *testing.T) {
	for regex, suite := range regexTests {
		for res, lines := range suite {
			for _, line := range lines {
				if regex.Regex.MatchString(line) != res {
					t.Error(
						`For regex "`, regex.Name,
						`" and line "`, line,
						`" expected [`, res,
						`] got [`, !res, `]`,
					)
				}
			}
		}
	}
}
