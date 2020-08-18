package types

import (
	"fmt"
	"regexp"
	"strings"
)

type Matcher struct {
	string
	*regexp.Regexp
}

func (m *Matcher) Match(candidate string) bool {
	if m.Regexp != nil {
		return m.Regexp.MatchString(candidate)
	}

	return strings.ToLower(candidate) == m.string
}

func NewMatchers(in []string) ([]*Matcher, error) {
	if len(in) == 0 {
		return []*Matcher{{
			Regexp: regexp.MustCompile(".*"),
		}}, nil
	}

	var ret []*Matcher

	for _, name := range in {
		if len(name) >= 2 && name[0] == '/' && name[len(name)-1] == '/' {
			re, err := regexp.Compile(`(?i:` + name[1:len(name)-1] + `)`)
			if err != nil {
				return nil, fmt.Errorf("compiling %q as regexp: %w", err)
			}
			ret = append(ret, &Matcher{Regexp: re})
		} else {
			ret = append(ret, &Matcher{string: strings.ToLower(name)})
		}
	}

	return ret, nil
}
