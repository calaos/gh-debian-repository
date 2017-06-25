package main

import (
	"net/http"
	"fmt"
	"io"
	"io/ioutil"
	"errors"
	"compress/gzip"
	"archive/tar"
	"strings"

	"github.com/blakesmith/ar"
	"os"
	"path/filepath"
)

func enumerateDebianArchive(r io.Reader, fn func(name string, r *ar.Reader) error) error {
	rd := ar.NewReader(r)

	for {
		header, err := rd.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		err = fn(header.Name, rd)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func parseDebianBinary(r io.Reader) (string, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	version := string(data)
	version = strings.TrimSpace(version)
	if version != "2.0" {
		return "", fmt.Errorf("only debian version 2.0 is supported, where received: %s", version)
	}
	return version, nil
}

func parseControlTarGz(url string, r io.Reader) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	rd := tar.NewReader(gz)
	for {
		header, err := rd.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Name == "control" || header.Name == "./control" {
			return ioutil.ReadAll(rd)
		}
	}
	return nil, errors.New("control not found in control.tar.gz")
}

func readDebianArchive(url string) (debianControl []byte, etag string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("http status code: %d %s", resp.StatusCode, resp.Status)
		return
	}
	defer resp.Body.Close()

	etag = resp.Header.Get("Etag")

	repositoryCache := os.Getenv("REPOSITORY_CACHE")
	if repositoryCache == "" {
		repositoryCache = "tmp-cache"
	}
	cachePath := filepath.Join(repositoryCache, etag + "-control")

	// read file from cache
	debianControl, err = ioutil.ReadFile(cachePath)
	if err == nil {
		return
	}

	debianVersion := ""

	err = enumerateDebianArchive(resp.Body, func(name string, r *ar.Reader) error {
		if name == "debian-binary" || name == "debian-binary/" {
			version, err := parseDebianBinary(r)
			debianVersion = version
			return err
		} else if name == "control.tar.gz" || name == "control.tar.gz/" {
			if debianVersion == "" {
				return errors.New("debian-binary has to be first")
			}
			control, err := parseControlTarGz(url, r)
			if err != nil {
				return err
			}
			debianControl = control
			return io.EOF
		} else {
			return nil
		}
	})
	if err == nil && (debianVersion == "" || debianControl == nil) {
		err = errors.New("missing debian-binary or control.tar.gz")
	}
	if err != nil {
		return
	}

	// write file to cache
	f, err2 := ioutil.TempFile(repositoryCache, "control-etag-cache")
	if err2 == nil {
		defer f.Close()
		defer os.Remove(f.Name())
		_, err2 = f.Write(debianControl)
		f.Close()
	}
	if err2 == nil {
		os.Remove(cachePath)
		os.Rename(f.Name(), cachePath)
	}
	return
}
