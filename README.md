# RUNNER-DOWNLOAD
artifact downloader for runners

This service is provided for runners (unmanaged or managed) to handle artifact downloads
from OCI Object Storage or the local file system (unmanaged runners). It is run as a service
in a Pod under Kubernetes/OKE for managed runners. It is run as a command on the same machine
that hosts a set of unmanaged runners (using the local file system for pipeline storage). The
http GET directed to this service will access the artifact and stream it through the redirect
cuasing the browser to download the artifact tar. 

Regardless of runner type, it is invoked by a redirect to the URL derived from the Run object
information by the Web-API local_modules/api-runsteps/get-runStep-aretifact.js 

For OCI storage, the operating enviuronment must be setup with all the required OCI information
and credentials. No sensitive information is passed over the wire. The following environment
properties must be established before running this program: 

   WERCKER_OCI_TENANCY_OCID
   WERCKER_OCI_USER_OCID
   WERCKER_OCI_REGION
   WERCKER_OCI_PRIVATE_KEY_PATH
   WERCKER_OCI_PRIVATE_KEY_PASSPHRASE
   WERCKER_OCI_FINGERPRINT
   WERCKER_OCI_NAMESPACE
   WERCKER_OCI_BUCKETNAME  



Execution
--------

runner-download [--debug] server [--port=8091] 

The host address and port are used to form a URL that is supplied to kiddie-pool to bind the
download functionality to the output of runs processed by that runner. 
