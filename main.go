package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ayufan/debian-repository/internal/apache_log"
	"github.com/ayufan/debian-repository/internal/deb"
	"github.com/ayufan/debian-repository/internal/deb_cache"
	"github.com/ayufan/debian-repository/internal/deb_key"
	"github.com/ayufan/debian-repository/internal/github_client"
)

var signingKey *deb_key.Key

func createRoutes() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/settings/cache/clear", clearHandler).Methods("GET", "POST")

	r.HandleFunc("/", mainHandler).Methods("GET")

	r.HandleFunc("/orgs/{owner}", indexHandler).Methods("GET")
	r.HandleFunc("/orgs/{owner}/", indexHandler).Methods("GET")
	r.HandleFunc("/orgs/{owner}/archive.key", archiveKeyHandler).Methods("GET")
	r.HandleFunc("/orgs/{owner}/{component}", distributionIndexHandler).Methods("GET")
	r.HandleFunc("/orgs/{owner}/{component}/", distributionIndexHandler).Methods("GET")
	r.HandleFunc("/orgs/{owner}/{sucomponentite}/Packages", packagesHandler).Methods("GET")
	r.HandleFunc("/orgs/{owner}/{component}/Packages.gz", packagesGzHandler).Methods("GET")
	r.HandleFunc("/orgs/{owner}/{component}/Release", releaseHandler).Methods("GET")
	r.HandleFunc("/orgs/{owner}/{component}/Release.gpg", releaseGpgHandler).Methods("GET")
	r.HandleFunc("/orgs/{owner}/{component}/InRelease", inReleaseHandler).Methods("GET")
	r.HandleFunc("/orgs/{owner}/{component}/download/{repo}/{tag_name}/{file_name}", downloadHandler).Methods("GET", "HEAD")

	r.HandleFunc("/{owner}/{repo}", indexHandler).Methods("GET")
	r.HandleFunc("/{owner}/{repo}/", indexHandler).Methods("GET")
	r.HandleFunc("/{owner}/{repo}/archive.key", archiveKeyHandler).Methods("GET")
	r.HandleFunc("/{owner}/{repo}/{component}", distributionIndexHandler).Methods("GET")
	r.HandleFunc("/{owner}/{repo}/{component}/", distributionIndexHandler).Methods("GET")
	r.HandleFunc("/{owner}/{repo}/{component}/Packages", packagesHandler).Methods("GET")
	r.HandleFunc("/{owner}/{repo}/{component}/Packages.gz", packagesGzHandler).Methods("GET")
	r.HandleFunc("/{owner}/{repo}/{component}/Release", releaseHandler).Methods("GET")
	r.HandleFunc("/{owner}/{repo}/{component}/Release.gpg", releaseGpgHandler).Methods("GET")
	r.HandleFunc("/{owner}/{repo}/{component}/InRelease", inReleaseHandler).Methods("GET")
	r.HandleFunc("/{owner}/{repo}/{component}/download/{tag_name}/{file_name}", downloadHandler).Methods("GET", "HEAD")

	return r
}

func main() {
	var err error

	flag.Parse()

	if *parseDeb != "" {
		deb, err := deb.ReadFromFile(*parseDeb)
		if err != nil {
			log.Fatalln(err)
		}

		log.Println(string(deb.Control))
		return
	}

	githubAPI = github_client.New(os.Getenv("GITHUB_TOKEN"), *requestCacheExpiration)
	packagesCache = deb_cache.New(*packageLruCache)

	signingKey, err = deb_key.New(os.Getenv("GPG_KEY"))
	if err != nil {
		log.Fatalln(err)
	}

	allowedOwners = strings.Split(os.Getenv("ALLOWED_ORGS"), ",")
	if len(allowedOwners) == 0 {
		log.Println("Allowed owners: none")
	} else {
		log.Println("Allowed owners:", strings.Join(allowedOwners, ", "))
	}

	routes := createRoutes()

	loggingHandler := apache_log.NewApacheLoggingHandler(routes, os.Stdout)
	http.Handle("/", loggingHandler)

	log.Println("Starting web-server on", *httpAddr, "...")
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}
