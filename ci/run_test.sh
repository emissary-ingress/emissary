#!/bin/bash
printf "==Begin: Helm tests==\n"

export KUBECONFIG=$DEV_KUBECONFIG
export OSS_TAG=$(cat ../Chart.yaml | grep ossVersion | sed s/'ossVersion: '/''/)

echo $KUBECONFIG
kubectl get svc
## Generate values files
##
printf "==Generating values files==\n"

mkdir values
for file in $(ls values_templates)
do
  echo $(pwd)/values_templates/$file
  envsubst < $(pwd)/values_templates/$file > $(pwd)/values/$file
done

printf "==Bootstrapping Helm installs==\n"

## Bootstrap Helm 2
## 
kubectl apply -f helm-init.yaml
helm2 version
helm2 init --service-account=tiller --wait

## Bootstrap Ambassador release
##
helm version
helm install ambassador .. --wait -f ci-default-values.yaml 2>&1 > /dev/null

kubectl apply -f tls.yaml

kubectl apply -f backend.yaml
  
while [[ $(kubectl get po -l app=quote -o jsonpath='{.items[0].status.phase}') != 'Running' ]]
do
  echo waiting for backend
  sleep 5
done


success=1

printf "==Begin: Testing Helm 3 releases==\n"

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
    echo Test $v_file PASSED!
  else
    echo  Test $v_file FAILED!
    success=0
  fi

  pkill "kubectl"
done

printf "End: Testing Helm 3 releases==\n"

## Test Helm 2
##
printf "Begin: Testing Helm 2 release==\n"

helm2 install -n ambassador-helm2 .. -f helm2-values.yaml --wait 2>&1 > /dev/null

echo Release installed with Helm 2

kubectl port-forward service/ambassador-helm2 9443:443 2>&1 > /dev/null &

sleep 1

if [[ $(curl -kI https://localhost:9443/backend/ 2> /dev/null | grep OK) != '' ]]
then
  echo Success!
else
  echo Failure!
  success=0
fi

printf "==End: Testing Helm 2 releases==\n"

## Clean up
##
printf "==Begin: Cleanup==\n"
rm -rf values

kubectl delete -f backend.yaml

pkill "kubectl"

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

printf "==End: Cleanup==\n"

if [ $success != 1 ]
then
  exit 1
fi

printf "End: Helm tests==\n"
