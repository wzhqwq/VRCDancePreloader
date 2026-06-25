package mixed_server

import "net/http"

func getHttpHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/download", handleDownloadRequest)
	mux.HandleFunc("/cached", handleCacheRequest)

	return mux
}
