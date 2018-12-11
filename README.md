# RUNNER-DOWNLOAD
artifact downloader for runners

This service is provided for runners (unmanaged or managed) to handle artifact downloads
from the OCI Object Storage or the local file system (unmanaged runners only). It is invoked
in response to the Download artifact button on the Wercker Run page. It operates  as a service
in a Pod under Kubernetes/OKE for managed runners. It can also run as a command on the same machine
that hosts a set of unmanaged runners (using the local file system for pipeline storage). The
HTTP GET redirect to this service will access the artifact according to its residence and stream it 
through the browser to download the artifact tar. 

Regardless of runner type, it is invoked by a redirect to the URL derived from the Run object
information by the Web-API local_modules/api-runsteps/get-runStep-artifact.js 

For OCI storage, the operating environment must be setup with all the required OCI information
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

Execution as a command for an unmanaged runner
---------------------------------------------

The command is invoked on the same host system as the Wercker CLI runner start is issued when the
local file system is used as persistant storage for the pipeline data. The Wercker CLI runner start 
command must provide the --oci-download option containing the URL used by the Wercker Web UI to start 
unmanaged runners (i.e  --oci-download=http://<hostname or ip>:<port number> ). The choice of port number
is optional and defaults to port 8091 when the runner-download command is executed.  

runner-download [--debug] server [--port=8091] 

The Wercker Web UI must be able to redirect to the runner-download command from the Internet. For this to
work it is necessary to setup suitable routing and/or open the chosen port number through your router. Artifacts
can only be downloaded provided the runner-command is running and accessible. 

When using OCI Object Storage for the persistant storage medium, it is not necesdsary to run the service on
the same host computer where the unmanaged runner is executing. For this type of configuration, the
runner-download can run anywhere that has access to the OCI Object Storage service and can be accessed as
described earlier. 

Execution as a Kubernetes or OKE Service
----------------------------------------

The runner-download command runs within a container managed by the Kubernetres or OKE system. This type of 
configuration is a more complicated setup and is intended ONLY for pipeline information persisted to OCI
object storage. 

1. Create a secret which contains all the OCI Object Storage access information and credentials. 

   kubectl create secret generic ocisecrets \
   --from-literal tenancy=<oci tenancy> \
   --from-literal userocid=<oci username> \
   --from-literal fingerprint=<fingerprint> \
   --from-literal namespace=<oci namespace> \
   --from-literal bucket=<bucket-name> \
   --from-literal region=<region> \
   --from-literal passphrase=<passphrase> \
   --from-file api-key=/Users/bihaber/.oci/oci_api_key.pem

   Note: The secret names must coincide with the secrets mapping to environment variables in the deployment.

2. Create a deployment to run the runner-download Docker image inside a Pod. This deployment requires   
requires more setup  

  apiVersion: extensions/v1beta1
  kind: Deployment
  metadata:
    name: runner-download
  spec:
    replicas: 1
    strategy:
      type: RollingUpdate
      rollingUpdate:
        maxUnavailable: 50%
        maxSurge: 0
    selector:
        matchLabels:
          app: runner-download
    template:
      metadata:
        labels:
          app: runner-download
      spec:
         containers:
         - name: runner-download
           image: runner-download:test
           ports:
           - containerPort: 8091
           imagePullPolicy: Always
           command: [
             "/runner-download",
             "server",
             "--port=8091"
           ]
           env:
           - name: WERCKER_OCI_TENANCY_OCID
             valueFrom:
                secretKeyRef:
                   name: ocisecrets
                   key: tenancy
           - name: WERCKER_OCI_USER_OCID
             valueFrom:
                secretKeyRef:
                   name: ocisecrets
                   key: userocid
           - name: WERCKER_OCI_REGION
             valueFrom:
                secretKeyRef:
                   name: ocisecrets
                   key: region

           - name: WERCKER_OCI_PRIVATE_KEY
             valueFrom:
                secretKeyRef:
                   name: ocisecrets
                   key: api-key
           - name: WERCKER_OCI_FINGERPRINT
             valueFrom:
                secretKeyRef:
                   name: ocisecrets
                   key: fingerprint
           - name: WERCKER_OCI_PRIVATE_KEY_PASSPHRASE
             valueFrom:
                secretKeyRef:
                   name: ocisecrets
                   key: passphrase
          - name: WERCKER_OCI_NAMESPACE
             valueFrom:
                secretKeyRef:
                   name: ocisecrets
                   key: namespace
           - name: WERCKER_OCI_BUCKETNAME
             valueFrom:
                secretKeyRef:
                   name: ocisecrets
                   key: bucket
           - name: MY_POD_NAME
             valueFrom:
               fieldRef:
                 fieldPath: metadata.name
           - name: MY_POD_IP
             valueFrom:
               fieldRef:
                 fieldPath: status.podIP

  -------------
  apiVersion: v1
  kind: Service
  metadata:
    name: runner-download-service

  spec:
    type: NodePort
    ports:
    -  port: 8091
    selector:
       app: runner-download

