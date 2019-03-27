# Load Testing Apro with Vegeta

## Set Up

1. Stand Up Ambassador Pro and the load-testing Backend (in a Kubernaut cluster) from the reference architecture

   ```sh
   git clone https://github.com/datawire/pro-ref-arch.git
   
   kubectl apply -f pro-ref-arch/ambassador/
   kubectl apply -f pro-ref-arch/scaling/http-echo.yaml
   ```

   This will deploy Ambassador Pro with the http-echo backend for load testing. No rate limits are applied currently.

   **Note:** This README assumes you are not listening for cleartext. Using Vegeta over HTTPS is more complicated. The reference architecture deploys Ambassador secured with a self signed certificate. Apply the version of the ambassador service here to configure Ambassador to listen for cleartext.

   ```
   kubectl apply -f ambassador/ambassador-service.yaml
   ```

2. Install Vegeta

   - For OSX users
   
      ```sh
      brew update && brew install vegeta
      ```

   - From Source
   
      ```sh
      go get -u github.com/tsenart/vegeta
      ```

3. Get the IP of you kubernaut node and port of Ambassador service

   `NODE_IP` can be found in your KUBECONFIG

   ```sh
   cat ~/.kube/{KUBENAUT_CONFIG} | grep server
   ```

   `NODE_PORT` is found by running `kubectl get svc ambassador`
   

4. Attack the service with Vegeta

   ```sh
   echo "GET http://{NODE_IP}:{NODE_PORT}/http-echo/" | vegeta attack -rate 500 -duration 5s > result-500.txt
   ```

   - You can see a report of this attack by running
   
      ```sh
      vegeta report result-500.txt

      Requests      [total, rate]            2500, 500.21
      Duration      [total, attack, wait]    5.041813067s, 4.997891s, 43.922067ms
      Latencies     [mean, 50, 95, 99, max]  75.565539ms, 45.018334ms, 225.885843ms, 330.822946ms, 693.644303ms
      Bytes In      [total, mean]            35000, 14.00
      Bytes Out     [total, mean]            0, 0.00
      Success       [ratio]                  100.00%
      Status Codes  [code:count]             200:2500  
      Error Set:
      ```

   - Vegeta can also plot a graph of latency over time, viewable in an HTML file

      ```sh
      vegeta plot result-500.txt > plot.html
      ```

## Latency Testing

Vegeta records latency of each request when attacking. 

Get a text output of this data by running `vegeta report` 

```sh
vegeta report {ATTACK_OUTPUT}.txt
```

Or get a plot of latency over time by running `vegeta plot`

```sh
vegeta plot {ATTACK_OUTPUT}.txt > plot.html
```

### Without Pro

Remove the Ambassador Pro services to see latency from just Ambassador 

```
kubectl apply -f no-pro/
```

**Note:** You may need to restart Ambassador for the rate limiting configuration to change.


```sh
echo "GET http://{NODE_IP}:{NODE_PORT}/http-echo/" | vegeta attack -rate 500 -duration 5s | vegeta report
```

### With Rate Limiting

**Note:** Of course make sure Pro is configured 

Configure ratelimiting on the `http-echo` service. Apply the yaml in the `ratelimiting/` directory to add `label` `generic_key: http` to requests to `http-echo` and configure a `RateLimit` of 1000 RPS.

```sh
kubectl apply -f ratelimiting/
```

```sh
echo "GET http://{NODE_IP}:{NODE_PORT}/http-echo/" | vegeta attack -rate 500 -duration 5s | vegeta report
```

### Multiple configurations in one graph

You can change the duration of Vegeta to run for as long as you want. This allows for you to make configuration changes while Vegeta is running and get sequential latency reports.

```sh
echo "GET http://{NODE_IP}:{NODE_PORT}/http-echo/" | vegeta attack -rate 500 -duration 5m > result-multi.txt
```

This will start a Vegeta attack at 500 RPS for 5 minutes. While the attack is going on, make any configuration changes you would like to see how it affects latency.

After the attack is finished, plot the results and view the graph in a web browser

```sh 
vegeta plot result-multi.txt
```

## Find Limit Where Pro Fails

max_load.go can be used to find the point where Pro starts returning 500 responses.

First, edit the URL in line 15 to whatever IP and port you are using.

Then, simply invoke it with:

```
go run max_load.go
```

and it will start issuing attacks at 100 RPS for 5 seconds. 

- On success (defined as all 200 responses), it will issue another attack at 2x the rate. 
- On failure, it will do a binary search to find the largest limit rate where it is returned a 100% success rate


#Testing Scenarios

## Testing Environment

- Client Machine: 
   
   - 2014 MacBook Pro 
   - macOS Sierra
   - 2.6 GHz i5 Processor
   - 16 GB DDR3 Memory

- Kubernetes

   - Kubernaut Cluster

- Starting Ambassador Config

   - Single Pod Ambassador Pro 
   - No rate limiting
   - No filters
   - Ambassador 0.52.0
   - Ambassador Pro 0.2.3

## Base Config

```
kubectl apply -f base-config/
```

```
go run max_load.go
```

Run 1: 1653 RPS
Run 2: 1636 RPS
Run 3: 1649 RPS

Mean: ~1646 RPS

## Per second Rate Limiting

```
k apply -f per-second/
```

```
go run max_load.go
```

Run 1: 690 RPS
Run 2: 681 RPS
Run 3: 687 RPS

Mean: ~686

## Per second Rate Limiting Scaling

Increase the number of replicas of the Ambassador Pro deployment to 4

```
kubectl apply -f scaling/
```

```
go run max_load.go
```

Run 1: 577 RPS
Run 2: 575 RPS
Run 3: 573 RPS

Mean: ~575 RPS

## Per minute rate limiting

Scale Ambassador back down to 1 pod and use a per-minute rate limit.

```
kubectl apply -f base-config/
kubectl apply -f per-minute/
```

```
go run max_load.go
```

Run 1: 703 RPS
Run 2: 660 RPS
Run 3: 665 RPS

Mean: ~676 RPS

## Calling out to K8s Service vs local host

Configure Pro to create K8s services for calls to rate limiting.

```
kubectl apply -f k8s-service/
```

```
go run max_load.go
```

Run 1: 
Run 2: 
Run 3: 

Means: ~