# Diagnostics

If Ambassador Edge Stack is not routing your services as you'd expect, your first step should be the Ambassador Edge Stack Diagnostics 
in the Edge Policy Console. Login to your Edge Policy Console and select the "Debugging" tab from the left menu. 

Some of the most important information (your Ambassador Edge Stack version, how recently Ambassador Edge Stack's configuration was updated, and how recently Envoy last reported status to Ambassador Edge Stack) is right at the top. The diagnostics overview can show you what it sees in your configuration map, and which Envoy objects were created based on your configuration.

## Health status

Ambassador Edge Stack displays the health of your services on the Dashboard of your Edge Policy Console. Health is computed as successful requests / total requests and expressed as a percentage. The total requests comes from nvoy `upstream_rq_pending_total` stat. Successful requests is calculated by substracting `upstream_rq_4xx` and `upstream_rq_5xx` from the total.

* Red is used when the success rate ranges from 0% - 70%.
* Yellow is used when the success rate ranges from 70% - 90%.
* Green is used when the success rate is > 90%.

## Troubleshooting

If the diagnostics service does not provide sufficient information, Kubernetes and Envoy provide additional debugging information.

If Ambassador Edge Stack isn't working at all, start by looking at the data from the following:

* `kubectl describe pod <ambassador-pod>` will give you a list of all events on the Ambassador Edge Stack pod
* `kubectl logs <ambassador-pod> ambassador` will give you a log from Ambassador Edge Stack itself

If you need additional help, feel free to join our [Slack channel](https://d6e.co/slack) with the above information (along with your Kubernetes manifest).

You can also increase the debug of Envoy through the button in the diagnostics panel. Turn on debug logging, issue a request, and capture the log output from the Ambassador Edge Stack pod using `kubectl logs` as described above.
