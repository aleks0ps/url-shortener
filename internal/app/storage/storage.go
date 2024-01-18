package storage

var Urls = make(map[string]string)

func StoreURL(key string, origURL string) {
	Urls[key] = origURL
}

func GetOrigURL(key string) (string, bool) {
	URL, ok := Urls[key]
	return URL, ok
}
