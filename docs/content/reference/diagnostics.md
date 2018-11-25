# Diagnostics

If Ambassador is not routing your services as you'd expect, your first step should be the Ambassador Diagnostics service. This is exposed on port 8877 by default. You'll need to use `kubectl port-forward` for access, e.g.,

```shell
kubectl port-forward ambassador-xxxx-yyy 8877
```

where you'll have to fill in the actual pod name of one of your Ambassador pods (any will do). Once you have that, you'll be able to point a web browser at

`http://localhost:8877/ambassador/v0/diag/`

for the diagnostics overview.

![Diagnostics](/images/diagnostics.png)

 Some of the most important information - your Ambassador version, how recently Ambassador's configuration was updated, and how recently Envoy last reported status to Ambassador - is right at the top. The diagnostics overview can show you what it sees in your configuration map, and which Envoy objects were created based on your configuration.

If needed, you can get JSON output from the diagnostic service, instead of HTML:

`curl http://localhost:8877/ambassador/v0/diag/?json=true`

## Health status

Ambassador displays the health of a service in the diagnostics UI. Health is computed as successful requests / total requests and expressed as a percentage. The total requests comes from nvoy `upstream_rq_pending_total` stat. Successful requests is calculated by substracting `upstream_rq_4xx` and `upstream_rq_5xx` from the total. 

Red is used when the success rate ranges from 0% - 70%.
Yellow is used when the success rate ranges from 70% - 90%.
Green is used when the success rate is > 90%.

## Troubleshooting

If the diagnostics service does not provide sufficient information, Kubernetes and Envoy provide additional debugging information.

If Ambassador isn't working at all, start by looking at the data from the following:

* `kubectl describe pod <ambassador-pod>` will give you a list of all events on the Ambassador pod
* `kubectl logs <ambassador-pod> ambassador` will give you a log from Ambassador itself

If you need additional help, feel free to join our [Slack channel](https://d6e.co/slack) with the above information (along with your Kubernetes manifest).

You can also increase the debug of Envoy through the button in the diagnostics panel. Turn on debug logging, issue a request, and capture the log output from the Ambassador pod using `kubectl logs` as described above.