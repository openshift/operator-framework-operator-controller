#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

help="
build-test-registry.sh is a script to stand up an image registry within a cluster.
Usage:
  build-test-registry.sh [NAMESPACE] [NAME] [IMAGE]

Argument Descriptions:
  - NAMESPACE is the namespace that should be created and is the namespace in which the image registry will be created
  - NAME is the name that should be used for the image registry Deployment and Service
  - IMAGE is the name of the image that should be used to run the image registry
"

if [[ "$#" -ne 3 ]]; then
  echo "Illegal number of arguments passed"
  echo "${help}"
  exit 1
fi

namespace=$1
name=$2
image=$3

oc apply -f - << EOF
---
apiVersion: v1
kind: Namespace
metadata:
  name: ${namespace}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${name}
  namespace: ${namespace}
  labels:
    app: registry
spec:
  replicas: 1
  selector:
    matchLabels:
      app: registry
  template:
    metadata:
      labels:
        app: registry
    spec:
      containers:
      - name: registry
        image: registry:3
        imagePullPolicy: IfNotPresent
        volumeMounts:
        - mountPath: /var/certs
          name: operator-controller-e2e-certs
        env:
        - name: REGISTRY_HTTP_TLS_CERTIFICATE
          value: "/var/certs/tls.crt"
        - name: REGISTRY_HTTP_TLS_KEY
          value: "/var/certs/tls.key"
      volumes:
        - name: operator-controller-e2e-certs
          secret:
            optional: false
            secretName: operator-controller-e2e-certs
---
apiVersion: v1
kind: Service
metadata:
  name: ${name}
  namespace: ${namespace}
  annotations:
    service.beta.openshift.io/serving-cert-secret-name: operator-controller-e2e-certs
spec:
  selector:
    app: registry
  ports:
  - name: http
    port: 5000
    targetPort: 5000
  type: NodePort
EOF

oc wait --for=condition=Available -n "${namespace}" "deploy/${name}" --timeout=60s

oc apply -f - << EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: ${name}-push
  namespace: ${namespace}
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: push
        image: ${image}
        command:
        - /push
        args: 
        - "--registry-address=${name}.${namespace}.svc:5000"
        - "--images-path=/images"
        volumeMounts:
        - mountPath: /var/certs
          name: operator-controller-e2e-certs
        env:
        - name: SSL_CERT_DIR
          value: "/var/certs/"
      volumes:
        - name: operator-controller-e2e-certs
          secret:
            optional: false
            secretName: operator-controller-e2e-certs
EOF

oc wait --for=condition=Complete -n "${namespace}" "job/${name}-push" --timeout=60s
