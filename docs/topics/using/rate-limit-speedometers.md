# Rate Limit Speedometers

The Edge Policy Console (EPC) Dashboard tab shows two gauges to help users monitor their throughput rates for Rate Limited and for Authenticated traffic with AES.  Evaluate your traffic and license needs before you commit to ensure your optimal capacity.

![](../../../images/speedometers.png)

The LEFT Speedometer

- Shows current and maximum rates per second (RPS) for AES Rate Limited traffic.
- Only traffic that has an AES rate limit applied will be monitored.  Select the [Rate Limits](../../using/rate-limits/rate-limits) tab in the EPC and apply at least one limit. If no rate limit has been set this gauge will show zero.
- Current values are actual current values and are updated every second.
- Max values are the highest rate of traffic over a 24 hr period.

The RIGHT Speedometer

- Shows current and maxiumum RPS for all Authenticated user traffic through AES.
- Authenticated usage is separate from the rate limited usage shown in the left hand speedometer.
- A user's authenticated traffic limit is determined by their license, which is demonstrated by the light red zone on the speedometer arc.  If traffic exceeds the licensed limit, the speedometer hands turn red.
- Evaluation users have a limit of 1 RPS and are encouraged to upgrade to a free community license which provides a limit of 5 RPS.

Authenticated traffic volumes are a key factor in determining capacity needs and we want to help you select the Ambassador license optimal for your purposes.