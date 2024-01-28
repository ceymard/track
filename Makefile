

track: *.go *.mod *.sum *.sql
	go build -tags "icu json1 sqlite_json"
