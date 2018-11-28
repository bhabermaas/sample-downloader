// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.

package downloadserver

import (
	"fmt"
	"time"

	ocicommon "github.com/oracle/oci-go-sdk/common"
	ocistorage "github.com/oracle/oci-go-sdk/objectstorage"
	"github.com/wercker/pkg/log"
	"golang.org/x/net/context"
)

// CreateOCIPAR creates a pre-authenticated URL for a download artifact from
// OCI Object Storage. This handler will also delete expired PARs as a
// housekeeping function.
func (ds *DownloadServer) CreateOCIPAR(parname string, artifact string) (string, error) {
	ctx := context.Background()
	// Create the configuration
	configProvider := ocicommon.NewRawConfigurationProvider(ds.Tenancy,
		ds.User, ds.Region, ds.Fingerprint, ds.Privatekey, &ds.Passphrase)

	// Create the object storage client
	client, err := ocistorage.NewObjectStorageClientWithConfigurationProvider(configProvider)
	if err != nil {
		return "", err
	}

	// Get a list of the current pre-authenticated URLS. Delete any expired.
	listDetails := ocistorage.ListPreauthenticatedRequestsRequest{
		NamespaceName: &ds.Namespace,
		BucketName:    &ds.BucketName,
	}

	list, err := client.ListPreauthenticatedRequests(ctx, listDetails)
	if err != nil {
		return "", err
	}

	// Clean out (delete) any expired items
	for _, item := range list.Items {
		nowUTC := time.Now().UTC()
		if item.TimeExpires.Before(nowUTC) {
			deleteRequest := ocistorage.DeletePreauthenticatedRequestRequest{
				NamespaceName: &ds.Namespace,
				BucketName:    &ds.BucketName,
				ParId:         item.Id,
			}
			client.DeletePreauthenticatedRequest(ctx, deleteRequest)
		}
	}

	// Specify 5 minutes to live
	expires := ocicommon.SDKTime{
		time.Now().Add(time.Minute * 2),
	}

	// Setup the creation details
	details := ocistorage.CreatePreauthenticatedRequestDetails{
		Name:        &parname,
		ObjectName:  &artifact,
		TimeExpires: &expires,
		AccessType:  "ObjectRead",
	}
	request := ocistorage.CreatePreauthenticatedRequestRequest{
		NamespaceName:                        &ds.Namespace,
		BucketName:                           &ds.BucketName,
		CreatePreauthenticatedRequestDetails: details,
	}

	response, err := client.CreatePreauthenticatedRequest(ctx, request)
	if err != nil {
		return "", err
	}
	par := fmt.Sprintf("https://%s%s", client.BaseClient.Host, *response.AccessUri)
	if ds.Debug {
		log.Debug(fmt.Sprintf("OCI PAR is %s", par))
	}
	return par, nil
}
