// Datawire-internal note: I've written these docs from the perspective that we
// intend to move this code into Ambassador OSS in the (near) future. Datawire
// folks also have access to the saas_app repository, which contains the
// implementation of the AgentCom and Director. External folks do not, so I've
// glossed over some of the details. Please improve these docs, but keep that
// constraint in mind. -- Abhay (@ark3)

/*
Package agent implements the Agent component in Ambassador.

The Agent is responsible for communicating with a cloud service run by Datawire.
It was introduced in AES 1.7.0. Ultimately, the goal is to be able to present a
cloud-based UI similar to the Edge Policy Console, but for now we just want to
display some information about what this AES knows is running in the cluster.

Implementation Goals

Minimal impact when disabled. The Agent is optional. If the user does not turn
on the Agent, the associated code should do very little, thereby having almost
no impact on the rest of Ambassador. This is no different from any other opt-in
feature in Ambassador.

Tolerant of external factors. When the Agent is enabled, it talks to a cloud
service run by Datawire. This means it is possible for things outside the user’s
cluster, that have nothing to do with the user’s usage of Ambassador, to affect
or even break that installation. Datawire could make a mistake, or there could
be an outage of their infrastructure outside of their control, or... The point
is, things outside the cluster that are not chosen by the user have now become
possible sources of failure for the user’s Ambassador installation. The Agent
must be robust enough to avoid precipitating such failures.

This is different from other opt-in features, because there is the potential for
external factors to break Ambassador that were not introduced by the user, but
rather by Datawire.

Overview

Datawire runs a microservice called AgentCom that implements the Director gRPC
service. The client for that service is the Agent; it runs in the user’s
Ambassador. To enable the Agent, the user must add configuration in the
Ambassador module, including the Agent’s account ID, which the user must obtain
from the online application.

If the Agent is enabled, it sends a snapshot of the current state of the cluster
to the Director on startup and whenever things change. This is done via the
Director’s Report method. At present, the report snapshot includes identity
information about this Ambassador and a small amount of information about each
Kubernetes Service known to this Ambassador.

The Agent also pulls directives from the Director and executes them. This is
done via the Director’s Retrieve method, which establishes a gRPC stream of
Directive messages flowing from the Director to the Agent.

Each Directive includes some flow control information (to tell the Agent to stop
sending reports or send them less frequently) and a list of commands for the
Agent to execute. In the future, these commands will be the mechanism to allow
the cloud UI to configure Ambassador and the cluster on behalf of the user. For
now, aside from flow control, the only command implemented is to log a short
string to Ambassador's log.

Design layers

* Protocol Buffers for data

Messages between the Agent and the Director are implemented using Protocol
Buffers (Proto3). Protobuf presents a straightforward story around forward and
backward compatibility. Both endpoints need to be written with the following in
mind: every field in a message is optional; unrecognized fields are ignored.

This makes it possible to add or remove (really, stop using) fields. If you add
a field, old code simply ignores it when decoding and leaves it unset when
encoding. If you stop using a field, old code will keep setting it when encoding
and see that it is unset when decoding. New code must account for old code
behaving that way, but does not otherwise need to consider message versioning
explicitly.

Of course, not every field can really be optional. For example, a report from
the Agent is syntactically valid without an account ID, but it is not
semantically meaningful. It is up to the software at the endpoints to report an
error when a message is invalid.

* gRPC for communication

By using gRPC for the communication protocol between the Agent and the Director,
we gain a number of well-tested features for relatively low cost. gRPC is built
on HTTP/2, which is generally well-supported in locked-down environments and
works well with Envoy.

Generated code and the associated library together enable type-safe RPCs from
the Agent to the Director, offering a simple interface for handling
serialization, streaming messages to avoid polling, connection multiplexing,
automatic retries with exponential backoff, and TLS. The generated API is
straightforward imperative, blocking code even though there is a lot of
machinery running concurrently under the hood to make this fast and responsive.
As gRPC is built on top of Protocol Buffers, it has standard error types for
Proto-specific cases such as semantically invalid messages in addition to types
for typical RPC errors.

* Simple communication layer

There is a small set of Go code that uses the generated gRPC methods. The
RPCComm Go structure encapsulates the gRPC client state, including its Go
context, and tracks the Goroutine required to handle streaming responses from
the Retrieve call. Once it has been created, the RPCComm communicates with the
rest of the code via Go channels. RPCComm has a wrapper around the Report method
that makes sure the Retrieve call is running.

* Reporting layer

The main Agent code has to do several things, and thus is somewhat complicated.
However, it is written in an event-driven manner, and nearly every computation
it performs is contained in a separate function that can be tested
independently. Note that actual test coverage is very thin for now.

The main loop blocks on Go channels listening for events. When it wakes up, it
handles the received event, reports to the Director if appropriate, and loops.

The Agent decides to send a report if it is configured to do so, reporting has
not been stopped by the Director, new information is available to send, and no
prior Report RPC is running. It performs the RPC in a separate single-shot
Goroutine to avoid blocking the loop. That Goroutine performs the RPC, sleeps
for a period, and then sends the result of the RPC over a channel as an event to
the main loop.

The code will not launch multiple RPCs (or Goroutines); it insists on each RPC
finishing before launching a new one. There is no queue of pending reports; the
loop only remembers the most recent report. An RPC error or timeout does not end
the loop; the error is logged and the loop continues. The RPC Goroutine sleeps
after performing the RPC to provide a simple, adjustable rate limit to
reporting. The loop receives the RPC result as an event; that is its indication
that the RPC is done.

The loop also receives directives as events. The directive is executed right
away in the same Goroutine, so commands must be fast/non-blocking for now. As
the only command available is to log a simple string, this is not a problem.
Directives can also include a flag to tell the Agent to stop reporting and a
duration to modify the reporting rate.

Finally, the loop receives new Watt snapshots as events. It uses the snapshot,
which includes everything this Ambassador knows about the cluster, to generate a
new report. If the new report is different from the last report that was sent,
the Agent stores the new report as the next one to be sent. The snapshot also
includes the information needed to determine whether the user has enabled the
Agent (in the Ambassador Module). So the Agent must receive and process
snapshots, even if all it discovers is that it is not enabled and doesn’t need
to do anything else.

Connectivity to the Director is handled by the communication layer described
above. The RPCComm instance is first created when the Agent decides to report.
If the Agent never needs to report, e.g., because it is not enabled, then the
RPCComm is never created and no connection is ever made to the AgentCom and
Director. During snapshot processing, the Agent may discover that the Ambassador
Module has changed. In that case, the current RPCComm (if it exists) is closed
and discarded so that a new one can be created when needed.

* Snapshot layer

AES has a simple publish/subscribe mechanism for distributing Watt snapshots
throughout the Amb-Sidecar code. It pushes snapshots to subscribers as they come
in from Watt, discarding and replacing stale snapshots if they are not consumed.
As a result, if the Agent is unable to keep up with incoming snapshots, other
AES components will not be blocked or otherwise affected and there will be no
backlog. This mechanism has existed for a while; I’m only mentioning it because
this is the only non-Agent source for events into the Reporting layer.

Communication

Reporting and retrieving operations share an identity message that includes the
account ID, which is how the cloud app identifies this particular Ambassador,
and the version of Ambassador, just in case we want to send different commands
to different versions. It also includes other information that does not affect
the behavior of the Agent or the Director.

The identity message is constructed from the Ambassador Module received in the
Watt snapshot (accounting for the Ambassador ID for this Ambassador). This code
cannot return an identity if the Agent is not enabled. The lack of an identity
short-circuits further evaluation of the snapshot, which means no report is
generated, no reporting happens, and no connection is initiated to the Director.

Reports to the Director also include a list of Service messages, which are
essentially stripped-down copies of Kubernetes Services manifests. The message
includes the name, namespace, and labels of the service, as well as the subset
of the annotations that have keys starting with app.getambassador.io.

The Agent retrieves and executes directives. Each directive includes a list of
commands. We could stream commands individually, but doing so in batches allows
for basic all-or-nothing communication. Each directive can also have two flow
control fields to allow the Director to adjust the Agent’s rate of reporting or
turn it off entirely. This allows the Director to force some or all Agents to
slow down their rate of reporting if cloud service is overwhelmed. The minimum
report period is implemented on the Agent side by sleeping in the RPC Goroutine
after the RPC completes; the Agent won’t launch a new RPC until that Goroutine
finishes and returns a result.

Interesting Cases

* Agent is disabled

When the Agent processes a snapshot, the first thing it does is attempt to
construct an identity, which requires pulling the account ID from the Ambassador
module. At this point, if the Agent is not enabled or the account ID is not
specified, the code will not construct an ID. This short circuits the rest of
snapshot processing, which means a new report cannot be generated, and so no
reporting is performed.

If the Agent is disabled right at startup, the above flow will happen with the
very first snapshot. Because a report is never generated, the Agent will not
even attempt to connect to the Director.

If the Agent is disabled sometime after startup, the above flow will cause no
further reports to be generated. An existing connection to the Director will
persist, but if that connection drops, the Agent will not connect again.

* Heavy load

The CPU and memory load the Agent can generate is limited by the size of the
Watt snapshot, specifically the number of Kubernetes Services. The Agent
effectively makes a very shallow copy of the Services in the snapshot, mostly
copying references and pointers. If the Agent decides to report, the generated
Protobuf/gRPC code must construct a serialized blob of bytes to send, which does
end up copying all the strings byte-by-byte, but that blob is short-lived. Other
than snapshot processing and reporting, the Agent’s workload is very brief and
does very little allocation.

Different components can fall behind under heavy CPU load (from the Agent, or
from other AES components). The reporting layer can fail to process Watt
snapshots as fast as they come in. The communications layer can fail to
serialize/deserialize reports as fast as they come in. If the network is slow,
then the communication layer could fall behind due to slow RPCs. This is all
okay, because none of the layers queues up a backlog or tries to do additional
work concurrently. Instead, each layer preserves only the most recent result and
eventually processes that result, or a subsequent one, in a serial manner.

* Slow or broken network

If the network is consistently slow (always, or for a stretch of time), some
layers may fall behind, and that is okay, as described above. If the network is
inconsistent, the Agent relies on the gRPC library's error reporting. The Agent
reacts to all errors in the same way: log the error and try again later. In all
cases, that later time is the next time the Agent decides to report.

Evolving the project

Users may run a given release of Ambassador for a very long time after future
versions have been released. Datawire may add new features to the AgentCom side
of things in the cloud app, or even roll back to older versions as the need
arises. Datawire may also choose to turn off the AgentCom side entirely.

This implementation of the Agent can handle those situations, so if a user
decides to run this release for a long time and leave the Agent enabled, they
should have no trouble regardless of what Datawire does with its cloud service.
If the AgentCom disappears entirely, or the Director loses its current gRPC
endpoints for some reason, this Agent’s communication layer will log errors but
will otherwise continue to function just fine. A future version of the Director
can choose to reject reports from this Agent, but that won’t cause any trouble
with this Ambassador. A future version of the Director can send commands that
this Agent doesn’t understand; it will simply ignore them thanks to the basic
compatibility properties of Protocol Buffers. Similarly, future versions of the
Agent can remain compatible with older versions of the Director.

The current design of the Agent does not take into consideration the fact that
multiple Ambassador Pods are likely to be running simultaneously. Every replica
runs an Agent that reports to the Director; it is the Director's responsibility
to de-duplicate reports as needed. Similarly, every replica executes all
directives retrieved. It is safe to do so in the current trivial implementation,
but adding commands that modify the cluster state will require considering how
to keep Agents from stepping on each other.

*/
package agent
