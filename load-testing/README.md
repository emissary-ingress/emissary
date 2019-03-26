# Load Testing Apro with Vegeta

## Set Up

1. Stand Up Ambassador Pro and Backend (in a Kubernaut cluster)

   ```sh
   kubectl apply -f ambassador/
   ```

   This will deploy Ambassador Pro with the http-echo backend for load testing. No rate limits are applied currently.

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
echo "GET http://{NODE_IP}:{NODE_PORT}/http-echo/" | vegeta attack -rate 500 -duration 5s > result-no-pro.txt
```

### With Rate Limiting

**Note:** Of course make sure Pro is configured 

Configure ratelimiting on the `http-echo` service. Apply the yaml in the `ratelimiting/` directory to add `label` `generic_key: http` to requests to `http-echo` and configure a `RateLimit` of 1000 RPS.

```sh
kubectl apply -f ratelimiting/
```

```sh
echo "GET http://{NODE_IP}:{NODE_PORT}/http-echo/" | vegeta attack -rate 500 -duration 5s > result-rl.txt
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

Simply invoke it with

```
go run max_load.go
```

and it will start issuing attacks at 100 RPS for 10 seconds. 

- On success (defined as all 200 responses), it will issue another attack at 2x the rate. 
- On failure, it will do a binary search to find the largest limit rate where it is returned a 100% success rate

#### Issues

It seems it does not do a good job of cleaning up open files after an attack. This needs to be resolved to get an accurate result since all attacks start to fail after some time.