package uploader

import (
	"io/ioutil"
	"log"
	"net/http"
)

func download(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body for %s: %s", url, err)
		}
	}()

	return ioutil.ReadAll(resp.Body)
}
