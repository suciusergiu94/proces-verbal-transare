// Package pdfdoc renders a proces verbal to a PDF mirroring the paper form.
package pdfdoc

import "strings"

// diacritics maps the Romanian letters to their ASCII equivalents. Both the
// comma-below (correct) and cedilla (legacy) forms of s and t are covered.
var diacritics = strings.NewReplacer(
	"ă", "a", "Ă", "A",
	"â", "a", "Â", "A",
	"î", "i", "Î", "I",
	"ș", "s", "Ș", "S",
	"ş", "s", "Ş", "S",
	"ț", "t", "Ț", "T",
	"ţ", "t", "Ţ", "T",
)

// Fold replaces Romanian diacritics with plain ASCII. The core PDF fonts are
// Latin-1, which cannot represent them, and the paper form is written without
// diacritics anyway.
func Fold(s string) string {
	return diacritics.Replace(s)
}
