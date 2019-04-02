# Certified Builds

Ambassador Pro uses certified Ambassador builds. These builds are based on Ambassador OSS builds, but undergo additional testing. In addition, bug fixes and security issues may be backported to Ambassador Pro builds under specific situations.

## Certified build testing

In general, certified builds undergo several types of testing.

* Community testing. All code in certified builds are first shipped as part of Ambassador OSS. With thousands of installs every week, the Ambassador community provides extensive testing.
* Integration testing. Ambassador certified builds are integration tested with popular integration points such as Prometheus, Consul, and Istio, to insure that Ambassador works as expected with other infrastructure software.
* Torture testing. Ambassador certified builds are subject to additional long-running torture tests designed to measure stability and reliability under various conditions.