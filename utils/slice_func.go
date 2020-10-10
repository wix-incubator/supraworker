package utils

func ContainsInts(in []int, search int) bool {
	for _, i := range in {
		if i == search {
			return true
		}
	}
	return false
}

func ConvertMapStringToInterface(in map[string]string) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range in {
		out[k] = v
	}
	return out
}
