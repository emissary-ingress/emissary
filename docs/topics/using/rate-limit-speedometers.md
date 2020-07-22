# Rate Limit Speedometers

The Edge Policy Console (EPC) Dashboard tab shows two gauges to help users monitor their throughput rates for Rate Limited and for Authenticated traffic with AES.

![](../../../images/speedometers.png)

The LEFT Speedometer

- Shows requests per second (RPS) for AES Rate Limited traffic.
- Only traffic that has an AES Rate Limit applied will be monitored.  Select the [Rate Limits](../../using/rate-limits/rate-limits) tab in the EPC and apply at least one limit. If no rate limit has been set this gauge will show zero.
- Current values are updated every second.
- Max values are the highest rate of traffic over a 24 hour period.

The RIGHT Speedometer

- Shows RPS for all Authenticated user traffic through AES.
- Authenticated usage is separate from the Rate Limited usage shown in the left hand speedometer.
- A user's authenticated traffic limit is determined by their license, which is visualized by the green zone on the speedometer arc.

Authenticated traffic volumes are a key factor in determining capacity needs and we want to help you select the Ambassador license optimal for your purposes.