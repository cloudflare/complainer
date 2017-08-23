package uploader

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

func download(url string) ([]byte, error) {
	logger := log.WithFields(log.Fields{"module": "uploader/download", "func": "download"})

	logger.Debugf("GETting %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Failed to download %s: %s", url, err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("Error closing response body for %s: %s", url, err)
		}
	}()

	return ioutil.ReadAll(resp.Body)
}
