#!/bin/bash
{
    set -e

    if ! kubectl version --client=true > /dev/null; then
        echo "Please install kubectl 1.15+ and try again."
        exit 1
    fi

    REQUIRED_VERSION='1.5.*'
    PROJECTS_MANIFEST=https://getambassador-preview.netlify.app/yaml/projects.yaml
    AES_IMAGE=datawire/aes:1.5.0

    echo "Checking AES version..."
    eval "$(kubectl get deploy -n ambassador ambassador -o go-template='{{range .spec.template.spec.containers}}{{.name}}='\''{{.image}}'\''{{"\n"}}{{end}}' | grep '^ambassador=')"
    eval "$(kubectl exec -n ambassador deploy/ambassador -- grep -F 'BUILD_VERSION=' /buildroot/ambassador/python/apro.version)"

    if ! [[ ${BUILD_VERSION} == ${REQUIRED_VERSION} ]] ; then
        echo "This beta requires AES version ${REQUIRED_VERSION}, found \"${BUILD_VERSION}\""
        exit 1
    fi

    if ! [[ "${ambassador}" == */aes:* ]]; then
        echo "This beta requires AES version ${REQUIRED_VERSION}, found OSS \"${BUILD_VERSION}\""
        exit 1
    fi

    echo "Found AES image=${ambassador}"
    echo "Found BUILD_VERSION=${BUILD_VERSION}"

    if [ "${ambassador}" != ${AES_IMAGE} ]; then
       echo
       echo "Please note! Continuing will update your ambassador image to: ${AES_IMAGE}"
       echo

       # Use the redirect so that this works when piped from curl
       read -r -p "Type yes to procceed, anything else to abort: " < /dev/tty

       if [ "${REPLY}" != yes ]; then
           echo "Aborted"
           exit 1
       fi

       # Update the image and wait for it rollout
       kubectl set image -n ambassador deploy/ambassador ambassador=${AES_IMAGE}
       kubectl rollout status -w -n ambassador deploy/ambassador
    fi

    kubectl apply -f ${PROJECTS_MANIFEST}
    kubectl wait --for condition=established --timeout=90s crd -lproduct=aes

    kubectl apply -f - <<EOF
apiVersion: getambassador.io/v2
kind: ProjectController
metadata:
  labels:
    projects.getambassador.io/ambassador_id: default
  name: projectcontroller
  namespace: ambassador
EOF

    kubectl patch deployment ambassador -n ambassador -p "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"date\":\"$(date +'%s')\"}}}}}"
}
