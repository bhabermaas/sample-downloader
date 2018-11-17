// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.

package downloadserver

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

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
	Storepath   string
	IsUnmanaged bool
}

// JobEnvironment contains environment values passed by the Web-API from the Run object
type JobEnvironment struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Protected bool   `json:"protected"`
}

// DownloadRequest contains the required information to download the artifact.
type DownloadRequest struct {
	ArtifactURL string           `json:"artifacturl"`
	Environment []JobEnvironment `json:"environment"`
}

// NewDispatchServer creates a DispatchServe
func NewDownloadServer() *DownloadServer {
	downloadServer := &DownloadServer{}
	getOCICredentials(downloadServer)
	return downloadServer
}

// Fill DownloadServer with OCI credentials
func getOCICredentials(ds *DownloadServer) *DownloadServer {
	ds.Tenancy = os.Getenv("WERCKER_OCI_TENANCY_OCID")
	ds.User = os.Getenv("WERCKER_OCI_USER_OCID")
	ds.Region = os.Getenv("WERCKER_OCI_REGION")
	ds.Privatekey = os.Getenv("WERCKER_OCI_PRIVATE_KEY")
	if ds.Privatekey == "" {
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
	ds.IsUnmanaged = false
	ds.Storepath = ""
	return ds
}

// ProcessDownloadRequest is called with the DownloadRequest. This is called from the POST
// processor to take apart the request and invoke the OCI Object Store helper to generate
// the pre-allocated URL.
func (ds *DownloadServer) ProcessDownloadRequest(req *DownloadRequest) (string, error) {
	// Get runner type and local store path (if not using OCI)
	dlType := getEnv(req, "WERCKER_KP_RUNNER")
	if dlType == "UNMANAGED" {
		ds.IsUnmanaged = true
	}
	ds.Storepath = getEnv(req, "WERCKER_KP_STOREPATH")
	dlTenancy := getEnv(req, "WERCKER_KP_OCI_TENANCY")
	dlUser := getEnv(req, "WERCKER_KP_OCI_USER")
	dlRegion := getEnv(req, "WERCKER_KP_OCI_REGION")
	dlNamesp := getEnv(req, "WERCKER_KP_OCI_NAMESPACE")
	// The following are optional for unmanaged runners
	dlFinger := getEnv(req, "WERCKER_KP_OCI_FINGERPRINT")
	dlPhrase := getEnv(req, "WERCKER_KP_OCI_PRIVATE_KEY_PASSPHRASE")
	dlPvtkey := getEnv(req, "WERCKER_KP_OCI_PRIVATE_KEY")

	if dlPvtkey == "" {
		// This is for local testing.
		keyfile := getEnv(req, "WERCKER_KP_OCI_PRIVATE_KEY_PATH")
		if keyfile != "" {
			filekey, err := ioutil.ReadFile(keyfile)
			if err != nil {
				return "", err
			}
			dlPvtkey = string(filekey)
		}
	}

	if ds.IsUnmanaged {
		return "", nil
	}

	if dlTenancy != ds.Tenancy || dlUser != ds.User || dlRegion != ds.Region || dlNamesp != ds.Namespace {
		err := errors.New("mismatched credentials")
		return "", err
	}

	if dlFinger == "" && dlPhrase == "" && dlPvtkey == "" {
		dlFinger = ds.Fingerprint
		dlPhrase = ds.Passphrase
		dlPvtkey = ds.Privatekey
	}

	parname := "download-parname"
	// Create the derived value.
	byt := make([]byte, 16)
	_, err := rand.Read(byt)
	if err == nil {
		parname = fmt.Sprintf("download-%X-%X-%X-%X-%X", byt[0:4], byt[4:6], byt[6:8], byt[8:10], byt[10:])
	}
	return ds.CreateOCIPAR(parname, req.ArtifactURL, dlFinger, dlPvtkey, dlPhrase)
}

// Pickup the environment informqtion passed as an environment from the Run object
func getEnv(req *DownloadRequest, key string) string {
	for _, env := range req.Environment {
		if env.Key == key {
			return env.Value
		}
	}
	return ""
}
