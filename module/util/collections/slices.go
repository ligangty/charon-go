package collections

func RemoveFromStringSlice(s []string, e string) []string {
	set := map[string]bool{}
	for _, v := range s {
		set[v] = true
	}
	delete(set, e)
	r := []string{}
	for k := range set {
		r = append(r, k)
	}
	return r
}
