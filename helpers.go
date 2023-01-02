package main

import "unicode"

func isAlpha(s string) bool {
	for _,l := range s {
		if !unicode.IsLetter(l) {
			return false
		}
	}
	return true
}

func firstN(s string, n int) string {
    i := 0
    for j := range s {
        if i == n {
            return s[:j]
        }
        i++
    }
    return s
}
