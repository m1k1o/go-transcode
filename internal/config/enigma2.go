package config

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

func parseEnigma2Config(conf Enigma2) (map[string]string, error) {
	// parse webif url
	webifUrl, err := url.Parse(conf.WebifUrl)
	if err != nil {
		return nil, fmt.Errorf("error while parsing enigma2 webif url: %w", err)
	}

	// if there is no streaming url, create it from webif url
	if conf.StreamUrl == "" {
		// create streaming url
		conf.StreamUrl = webifUrl.Scheme + "://"
		// add password and username if set
		if webifUrl.User != nil {
			conf.StreamUrl += webifUrl.User.String() + "@" + conf.StreamUrl
		}
		// add host and port
		conf.StreamUrl += webifUrl.Hostname() + ":8001/"
	}

	// parse streaming url
	streamUrl, err := url.Parse(conf.StreamUrl)
	if err != nil {
		return nil, fmt.Errorf("error while parsing enigma2 streaming url: %w", err)
	}

	// use default bouquet if not set
	if conf.Bouquet == "" {
		conf.Bouquet = "Favourites (TV)"
	}

	apiUrl := *webifUrl
	apiUrl.Path = path.Join(apiUrl.Path, "/web/getservices")

	// get services from webif
	services, err := enigma2Services(apiUrl.String())
	if err != nil {
		return nil, fmt.Errorf("error while getting enigma2 services: %w", err)
	}

	// find reference by bouquet name
	for _, service := range services {
		if service.Name == conf.Bouquet {
			conf.Reference = service.Reference
		}
	}

	if conf.Reference == "" {
		return nil, fmt.Errorf("could not find bouquet %s", conf.Bouquet)
	}

	// add reference to api url
	q := apiUrl.Query()
	q.Set("sRef", conf.Reference)
	apiUrl.RawQuery = q.Encode()

	// get services from webif
	services, err = enigma2Services(apiUrl.String())
	if err != nil {
		return nil, fmt.Errorf("error while getting enigma2 services: %w", err)
	}

	var streams = make(map[string]string)
	for _, service := range services {
		chUrl := *streamUrl
		chUrl.Path = path.Join(chUrl.Path, service.Reference)
		streams[enigma2ChannelName(service.Name)] = chUrl.String()
	}

	return streams, nil
}

// get services from webif
func enigma2Services(url string) ([]Service, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status error: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var obj ServiceList
	err = xml.Unmarshal(data, &obj)
	if err != nil {
		return nil, err
	}

	return obj.ServiceList, nil
}

func enigma2ChannelName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return name
}
