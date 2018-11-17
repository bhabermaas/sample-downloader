// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.

package downloadserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

/*
 * This module is the driver for the download feature. It will handle http post requests
 * for a download request from the Web API. The request payload is parsed into a DownloadRequest
 * object and then the download code is called. Either an error is returned or the PAR for the
 * download object. The PAR is returned in the POST response with the Referer header set to the
 * URL which causes the browser to download the artifact archive.
 */
func OCIdownloadServer(portNumber int) error {
	http.HandleFunc("/", download)
	log.Println("Starting Runner Artifact Download server")

	port := fmt.Sprintf(":%d", portNumber)
	if err := http.ListenAndServe(port, nil); err != nil {
		return err
	}
	return nil
}

// Setup the CORS headers so the POST will be homored by the browser
func setupCORSResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control")
	(*w).Header().Set("Access-Control-Max-Age", "86400")
}

// Download handler. Called by the http layer when a request is picked up. Verify the request
// and do the appropirate processing.
func download(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v3/operator/artifact/download" {
		http.Error(w, "404 not found", http.StatusNotFound)
		return
	}
	setupCORSResponse(&w, r)
	if r.Method == "OPTIONS" {
		// Headers already setup so just complete the handshake.
		return
	}

	downloadServer := NewDownloadServer()
	// GET is provided specifically for unmanaged runners to fetch the artifact directly
	// from the local file system and stream it back to the browser
	if r.Method == "GET" {

		urlstring := r.URL.Query().Get("artifact")
		tokens := strings.Split(urlstring, ",")

		err := downloadServer.streamTheArtifact(w, r, tokens[0], tokens[1])
		if err != nil {
			msg := fmt.Sprintf("%s", err)
			http.Error(w, msg, 500)
		}
		return
	}

	if r.Method != "POST" {
		// Not being called as a POST request, not allowed.
		http.Error(w, "Invalid method type", http.StatusMethodNotAllowed)
		return
	}

	// Get the payload
	decoder := json.NewDecoder(r.Body)
	req := DownloadRequest{}
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Get the PAR for this download.
	redirectUrl, err := downloadServer.ProcessDownloadRequest(&req)
	if downloadServer.IsUnmanaged {
		// For unmanaged with a storepath, read the file and stream the contents
		// back to the requestor.
		err = downloadServer.streamTheArtifact(w, r, req.ArtifactURL, downloadServer.Storepath)
		if err == nil {
			return
		}
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
	} else {
		// Send the PAR back as the textual response and set Location header
		// to the artifact URL. The requestor should check for the Location
		// header and if present force the browser to redirect to that URL.
		msg := fmt.Sprintf("\nPOST Download url: %s", redirectUrl)
		log.Println(msg)

		//w.Header().Set("Content-Disposition", "attachment; filename=fred.tar")
		//w.Header().Set("Content-Type", "application/octet-stream")
		//http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Location", redirectUrl)
		io.WriteString(w, redirectUrl)
	}
}

// Stream the artifact from the local file system back to the web-api where it is
// downloaded to the user's machine.
func (ds *DownloadServer) streamTheArtifact(w http.ResponseWriter, r *http.Request, artifact string, storepath string) error {
	artifactPath := fmt.Sprintf("%s/%s", storepath, artifact)
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
	n, err := io.Copy(w, f)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("\nGET Wrote %d bytes sending %s", n, artifactPath)
	log.Print(msg)
	return nil
}
