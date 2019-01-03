// Copyright (c) 2018, 2019, Oracle and/or its affiliates. All rights reserved.

package downloadserver

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/wercker/pkg/log"
)

/*
 * This module is the driver for the download feature. It will handle http post requests
 * for a download request from the Web API. The request payload is parsed into a DownloadRequest
 * object and then the download code is called. Either an error is returned or the PAR for the
 * download object. The PAR is returned in the POST response with the Referer header set to the
 * URL which causes the browser to download the artifact archive.
 */

// DownloadServer implements contains the cconfigured credentials for this instance
type DownloadServer struct {
	Tenancy     string
	User        string
	Region      string
	Privatekey  string
	Fingerprint string
	Passphrase  string
	Namespace   string
	BucketName  string
	Debug       bool
	// Following are values for HTTPS operation
	CertPemFile string
	KeyPemFile  string
}

var downloadServer *DownloadServer

// NewDispatchServer creates a DispatchServer
func NewDownloadServer() *DownloadServer {
	server := &DownloadServer{}
	server.getOCICredentials()
	downloadServer = server
	return server
}

// Fill DownloadServer with OCI credentials
func (ds *DownloadServer) getOCICredentials() {
	ds.Tenancy = os.Getenv("WERCKER_OCI_TENANCY_OCID")
	ds.User = os.Getenv("WERCKER_OCI_USER_OCID")
	ds.Region = os.Getenv("WERCKER_OCI_REGION")
	ds.Privatekey = os.Getenv("WERCKER_OCI_PRIVATE_KEY")
	if ds.Privatekey == "" && ds.Tenancy != "" {
		keyfile := os.Getenv("WERCKER_OCI_PRIVATE_KEY_PATH")
		filekey, err := ioutil.ReadFile(keyfile)
		if err != nil {
			log.Fatal(err)
		}
		ds.Privatekey = string(filekey)
	}
	ds.Fingerprint = os.Getenv("WERCKER_OCI_FINGERPRINT")
	ds.Passphrase = os.Getenv("WERCKER_OCI_PRIVATE_KEY_PASSPHRASE")
	ds.Namespace = os.Getenv("WERCKER_OCI_NAMESPACE")
	ds.BucketName = os.Getenv("WERCKER_OCI_BUCKETNAME")
}

// OCIdownloadSErver setsup the http protocol for the GETs
func (ds *DownloadServer) OCIdownloadServer(portNumber int) error {
	http.HandleFunc("/", download)
	port := fmt.Sprintf(":%d", portNumber)

	if ds.CertPemFile != "" && ds.KeyPemFile != "" {
		log.Info("Artifact download server is using HTTPS protocol.")
		// When both certificate and key are present start the service accepting HTTPS
		if err := http.ListenAndServeTLS(port, ds.CertPemFile, ds.KeyPemFile, nil); err != nil {
			return err
		}
	} else {
		log.Info("Artifact Download server is using HTTP protocol")
		if err := http.ListenAndServe(port, nil); err != nil {
			return err
		}
	}
	return nil
}

// Download handler. Called by the http layer when a request is picked up. Verify the request
// and do the appropirate processing.
func download(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v3/operator/artifact/download" {
		http.Error(w, "Download URL is incorrect, 404 not found", http.StatusNotFound)
		return
	}

	// GET is provided specifically for unmanaged runners to fetch the artifact directly
	// from the local file system and stream it back to the browser
	if r.Method != "GET" {
		http.Error(w, "protocol error", http.StatusMethodNotAllowed)
		return
	}

	// Break out query parameters
	qstring := r.URL.RawQuery
	parms, err := url.ParseQuery(qstring)
	if err != nil {
		errstr := fmt.Sprintf("%s", err)
		http.Error(w, errstr, http.StatusBadRequest)
		return
	}

	artifact := parms["a"]
	storepath := parms["s"]

	if len(artifact) < 1 {
		http.Error(w, "missing artifact a=", http.StatusBadRequest)
	}
	if len(storepath) > 0 {
		// Storepath is present so handle local file system download
		err := downloadServer.streamTheArtifact(w, r, artifact[0], storepath[0])
		if err != nil {
			msg := fmt.Sprintf("%s", err)
			http.Error(w, msg, 500)
		}
		r.Body.Close()
		return
	}

	// Assume oci artifact when tenancy is provided
	tenancy := parms["t"]
	if len(tenancy) < 1 {
		http.Error(w, "missing OCI specifier", http.StatusPartialContent)
		return
	}

	if tenancy[0] != downloadServer.Tenancy {
		http.Error(w, "wrong tenancy", http.StatusForbidden)
		return
	}

	// Strip off environment specific prefixes. OCI objects are store without these.
	artstr := artifact[0]
	prefixStaging := "wercker-development/"
	if strings.HasPrefix(artstr, prefixStaging) {
		artifact[0] = artstr[len(prefixStaging):]
	}
	prefixProduction := "wercker-production/"
	if strings.HasPrefix(artstr, prefixProduction) {
		artifact[0] = artstr[len(prefixProduction):]
	}

	// Get the PAR for this download.
	parname := "download-parname"
	// Create the derived value.
	byt := make([]byte, 16)
	_, err = rand.Read(byt)
	if err == nil {
		parname = fmt.Sprintf("download-%X-%X-%X-%X-%X", byt[0:4], byt[4:6], byt[6:8], byt[8:10], byt[10:])
	}
	artifactUrl, err := downloadServer.CreateOCIPAR(parname, artifact[0])
	if err != nil {
		msg := fmt.Sprintf("%s", err)
		http.Error(w, msg, 500)
		return
	}

	// Issue the GET using the preauthenticated URL and stream the result back
	stream, err := http.Get(artifactUrl)
	if err != nil {
		errstr := fmt.Sprintf("%s", err)
		http.Error(w, errstr, 500)
		return
	}
	index := strings.LastIndex(artifact[0], "/")
	filename := artifact[0][index+1:]
	header := fmt.Sprintf("attachment; filename=%s", filename)
	w.Header().Set("Content-Disposition", header)
	w.Header().Set("Content-Type", "binary/octet-stream")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", stream.Header.Get("Content-Length"))
	nbytes, err := io.Copy(w, stream.Body)
	if err != nil {
		if downloadServer.Debug {
			// See broken pipe messages
			errstr := fmt.Sprintf("%s", err)
			log.Error(errstr)
		}
		return
	}
	if downloadServer.Debug {
		msg := fmt.Sprintf("OCI download complete (%d bytes) - %s", nbytes, artifact[0])
		log.Debugln(msg)
	}
	r.Body.Close()
}

// Stream the artifact from the local file system back to the web-api where it is
// downloaded to the user's machine. This provides support to unmanaged runners with
// the optional download service (this component) ties to the runner.
func (ds *DownloadServer) streamTheArtifact(w http.ResponseWriter, r *http.Request, artifact string, storepath string) error {
	artifactPath := fmt.Sprintf("%s/%s", storepath, artifact)
	if ds.Debug {
		log.Debugln(fmt.Sprintf("Downloading local file from %s", artifactPath))
	}
	f, err := os.Open(artifactPath)
	if err != nil {
		return err
	}
	defer f.Close()
	index := strings.LastIndex(artifact, "/")
	filename := artifact[index+1:]
	header := fmt.Sprintf("attachment; filename=%s", filename)
	w.Header().Set("Content-Disposition", header)
	w.Header().Set("Content-Type", "binary/octet-stream")
	w.Header().Set("Accept-Ranges", "bytes")
	stat, err := f.Stat()
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	nbytes, err := io.Copy(w, f)
	if err != nil {
		if ds.Debug {
			// See broken pipe signals
			errstr := fmt.Sprintf("%s", err)
			log.Error(errstr)
		}
		return nil
	}
	if ds.Debug {
		msg := fmt.Sprintf("Local File download complete (%d bytes) - %s", nbytes, artifact)
		log.Debugln(msg)
	}
	return nil
}
