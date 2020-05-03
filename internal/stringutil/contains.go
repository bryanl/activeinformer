package stringutil

// Contains returns true if the string slice contains the string.
func Contains(sl []string, s string) bool {
	for _, v := range sl {
		if s == v {
			return true
		}
	}

	return false
}
