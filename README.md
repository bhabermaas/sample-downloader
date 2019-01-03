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
and credentials. No sensitive information is passed over the wire. All necessary credentials are 
supplied to this program and not passed during the redirect.  
The following environment properties must be established before running this program: 

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

The command is executed by running the runner-download Docker image on the same host system as the Wercker CLI runner start command. This is necessary when the local file system is used as persistant storage for the pipeline data. The Wercker CLI runner start command must include the --oci-download option containing the URL to be used by the Wercker Web UI (i.e  --oci-download=http://<hostname or ip>:<port number> ). The choice of port number
is optional and defaults to port 8091 when omittted from the  runner-download command.  

First pull the latest Docker image for the runner-download service - 
   
   docker pull iad.ocir.io/odx-pipelines/wercker/runner-download:latest

Then run the image in a Docker container -
   
   docker run -it --rm -p 8091:8091 iad.ocir.io/odx-pipelines/wercker/runner-download:latest /runner-download --debug server

The --debug option, when included, will log every download request in the log. The desired port number is specified
with the Docker -p option and the --port= parameter on the command. 

Example 
   docker run -it --rm -p 13005:13005 iad.ocir.io/odx-pipelines/wercker/runner-download:latest /runner-download --debug server --port=13005

The Wercker Web UI must be able to redirect to the runner-download program from the Internet. For this to
work it is necessary to setup suitable routing and/or open the chosen port number through your router. Artifacts
can only be downloaded when the runner-download is running and accessible. 

When using OCI Object Storage for the persistant storage medium, it is not necessary to run the service on
the same host computer where the unmanaged runners are executing. For this type of configuration, the
runner-download can run anywhere that has access to the OCI Object Storage service and can be accessed as
described earlier.  For OCI to work, you must start the container with the --env-file option pointing to a file containing all the OCI environment variables.  

Execution as a Kubernetes or OKE Service
----------------------------------------

The runner-download command runs within a container managed by Kubernetres or the OKE system. This type of 
configuration is a more complicated setup and is intended ONLY for pipeline information persisted to the OCI
Object Storage service. This configuration is meant to compliment managed runners controlled by the 
Wercker Operator. The same secrets used to configure the Wercker Operator must be used for this deployment.

1. Create a secret which contains all the OCI Object Storage access information and credentials. The secrets 
   already created for the Wercker Operator can be used in lieu of defining as described here. 

   kubectl create secret generic ocisecrets \
   --from-literal tenancy=<oci tenancy> \
   --from-literal userocid=<oci username> \
   --from-literal fingerprint=<fingerprint> \
   --from-literal namespace=<oci namespace> \
   --from-literal bucket=<bucket-name> \
   --from-literal region=<region> \
   --from-literal passphrase=<passphrase> \
   --from-file <path to api_key.pem>

   Note: The secret key names must coincide with the secrets mapping to environment variables in the deployment.

2. Create a deployment to run the runner-download Docker image inside a Pod. The example deployment shown below 
   will start the runner-download application and service. 

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
           image: iad.ocir.io/odx-pipelines/wercker/runner-download:latest
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

3. Setup network access for the runner-dowkload service. For a single replica this can be easily accomodated by setting up port forwarding. When more than one replica is desired, it is necessary to create an ingress service to send the download recdirect requests into the service. 

HTTPS Support Operation
-----------------------

HTTPS support is enabled by two additional arguments on the runner-download command. 

   --certfile= specifies the location of the cert.pem file to be used.
   --keyfile=  specifies the location of the key.pem file to be used. 

   These files are used initialize HTTPS support and must both be specified. 

   Example:

   docker run -it --rm -p 443:443 iad.ocir.io/odx-pipelines/wercker/runner-download:latest /runner-download --debug server --port=443 --certfile=server.crt --keyfile=server.key

   Certificate and key generation 
   ------------------------------

   The following openssl commands are used to generate the key and certificate PEM files. 

  - Key considerations for algotithm "RSA" >= 2048-bit
   
      openssl genrsa -out server.key 2048

  - Key consideration for algorithm "ECDSA" >= secp384r1
    List ECDSA for supported curves (openssl ecparam -list curves)

      openssl ecparam -genkey -name secp384r1 -out server.key

  - Generation of self-signed (x509) public key (PEM encodoings .pem | .crt) based on the private (.key)

      openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650
 