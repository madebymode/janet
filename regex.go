package janet

import "regexp"

// KarmaRegexps provides regex patterns for karma operations
type KarmaRegexps struct{}

var karmaReg = &KarmaRegexps{}

func (k *KarmaRegexps) MatchMotivate() *regexp.Regexp {
	return regexp.MustCompile(`^(.+)\s*motivate\s*$`)
}

func (k *KarmaRegexps) MatchGive() *regexp.Regexp {
	return regexp.MustCompile(`(<@[A-Za-z0-9]+>)\s*(\+{2,})(\s+for\s+(.+))?|(\S+)\s*(\+{2,})(\s+for\s+(.+))?`)
}

func (k *KarmaRegexps) MatchTake() *regexp.Regexp {
	return regexp.MustCompile(`(<@[A-Za-z0-9]+>)\s*(\-{2,})(\s+for\s+(.+))?|(\S+)\s*(\-{2,})(\s+for\s+(.+))?`)
}

func (k *KarmaRegexps) MatchQuery() *regexp.Regexp {
	return regexp.MustCompile(`^(?:goodplace\s+)?(?:karma|points)\s+for\s+(\S+)`)
}

func (k *KarmaRegexps) MatchThrowback() *regexp.Regexp {
	return regexp.MustCompile(`^(?:goodplace\s+)?throwback(?:\s+(\S+))?`)
}
