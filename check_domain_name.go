/*
Based on the from https://gist.github.com/chmike/d4126a3247a6d9a70922fc0e8b4f4013
*/

package fancyfmt

import (
	"strings"
	"unicode/utf8"

	"github.com/sirkon/errors"
)

// checkPackageNameDomain returns an error if the domain name is not valid
// See https://tools.ietf.org/html/rfc1034#section-3.5 and
// https://tools.ietf.org/html/rfc1123#section-2.
func checkPackageNameDomain(name string) error {
	switch {
	case len(name) == 0:
		return nil // an empty domain name will result in a cookie without a domain restriction
	case len(name) > 255:
		return errors.Newf("domain: name length is %d, can't exceed 255", len(name))
	}
	var l int
	for i := 0; i < len(name); i++ {
		b := name[i]
		if b == '.' {
			// check domain labels validity
			switch {
			case i == l:
				return errors.Newf("domain: invalid character '%c' at offset %d: label can't begin with a period", b, i)
			case i-l > 63:
				return errors.Newf("domain: byte length of label '%s' is %d, can't exceed 63", name[l:i], i-l)
			case name[l] == '-':
				return errors.Newf("domain: label '%s' at offset %d begins with a hyphen", name[l:i], l)
			case name[i-1] == '-':
				return errors.Newf("domain: label '%s' at offset %d ends with a hyphen", name[l:i], l)
			}
			l = i + 1
			continue
		}
		// test label character validity, note: tests are ordered by decreasing validity frequency
		if !(b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '-' || b >= 'A' && b <= 'Z') {
			// show the printable unicode character starting at byte offset i
			c, _ := utf8.DecodeRuneInString(name[i:])
			if c == utf8.RuneError {
				return errors.Newf("domain: invalid rune at offset %d", i)
			}
			return errors.Newf("domain: invalid character '%c' at offset %d", c, i)
		}
	}
	// check top level domain validity
	switch {
	case l == len(name):
		return errors.Newf("domain: missing top level domain, domain can't end with a period")
	case len(name)-l > 63:
		return errors.Newf("domain: byte length of top level domain '%s' is %d, can't exceed 63", name[l:], len(name)-l)
	case name[l] == '-':
		return errors.Newf("domain: top level domain '%s' at offset %d begins with a hyphen", name[l:], l)
	case name[len(name)-1] == '-':
		return errors.Newf("domain: top level domain '%s' at offset %d ends with a hyphen", name[l:], l)
	case name[l] >= '0' && name[l] <= '9':
		return errors.Newf("domain: top level domain '%s' at offset %d begins with a digit", name[l:], l)
	}

	// check dots, they should be here
	if !strings.Contains(name, ".") {
		return errors.New("domain: missing dots")
	}

	return nil
}
