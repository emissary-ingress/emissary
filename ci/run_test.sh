#!/bin/bash
KUBECONFIG=$DEV_KUBECONFIG
export OSS_TAG=$(cat ../Chart.yaml | grep ossVersion | sed s/'ossVersion: '/''/)

## Generate values files
##
mkdir values
for file in $(ls values_templates)
do
  echo $(pwd)/values_templates/$file
  envsubst < $(pwd)/values_templates/$file > $(pwd)/values/$file
done

sleep 100
## Bootstrap Helm 2
## 
kubectl apply -f helm-init.yaml

helm2 init --service-account=tiller --wait

## Bootstrap Ambassador release
##

helm install ambassador .. --wait -f ci-default-values.yaml 2>&1 > /dev/null

kubectl apply -f tls.yaml

kubectl apply -f backend.yaml
  
while [[ $(kubectl get po -l app=quote -o jsonpath='{.items[0].status.phase}') != 'Running' ]]
do
  echo waiting for backend
  sleep 5
done


success=0

for v_file in $(ls values)
do

  ## Upgrade the Ambassador release with new values file
  ##
  helm upgrade ambassador .. --wait -f values/$v_file 2>&1 > /dev/null
  
  echo Release upgraded with $v_file
  
  ## Test install by sending a test request to a backend api
  ##

  kubectl port-forward service/ambassador 8443:443 2>&1 > /dev/null &

  sleep 1

  if [[ $(curl -kI https://localhost:8443/backend/ 2> /dev/null | grep OK) != '' ]]
  then
    echo Success!
    success=1
  else
    echo Failure!
    success=0
  fi

  pkill kubectl port-forward service/ambassador 8443:443
done

## Test Helm 2
##

helm2 install -n ambassador-helm2 .. -f helm2-values.yaml --wait 2>&1 > /dev/null

echo Release installed with Helm 2

kubectl port-forward service/ambassador-helm2 9443:443 2>&1 > /dev/null &

sleep 1

if [[ $(curl -kI https://localhost:9443/backend/ 2> /dev/null | grep OK) != '' ]]
then
  echo Success!
  success=1
else
  echo Failure!
  success=0
fi

sleep 30
## Clean up
##
rm -rf values

kubectl delete -f backend.yaml

pkill kubectl port-forward service/ambassador-helm2 9443:443

helm uninstall ambassador > /dev/null
helm2 del --purge ambassador-helm2

while [[ $(kubectl get po -l app.kubernetes.io/instance=ambassador -o name) != '' ]];
do
  echo Waiting for helm chart to be uninstalled
  sleep 10
done

echo helm 3 chart uninstalled 

while [[ $(kubectl get po -l app.kubernetes.io/instance=ambassador-helm2 -o name) != '' ]];
do                                                                              
  echo Waiting for helm chart to be uninstalled                                 
  sleep 10                                                                      
done                                                                            
                                                                                
echo helm 2 chart uninstalled


if [ $success != 1 ]
then
  exit 1
fi


