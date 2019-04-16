# Model Cluster

The model cluster from the perspective of the developer using our tools. Please see [intent](#Intent) below.

Also, there is a [live-editable _copy_](https://hackmd.io/JMCN5Iz5SSyPSTiWmFrSIQ) of this on HackMD.

## Status

The standard APRo development deployment (what you get by running `make deploy`) includes four services in the default namespace (via `k8s-sidecar/04-model-cluster-app.yaml`).

- users
- posts
- comments
- events

Each service has a REST API to manage CRUD of objects of its type. The current implementation has POST (create) and GET (read), so individual objects are immutable and the store is append-only. Each service also supports a listing GET that returns a list of object IDs; some of them support filtering. Object IDs are strings (e.g, "user-37"). All data is represented in JSON format.

The services don't have durable storage, so restarting the deployment will blow away all the data. And yes, there is only one deployment at the moment, as all the services are basically identical.

### Example

```console
$ make deploy proxy
[...]

$ curl -s http://users/api/v1/users | jq .
[]

$ curl -s -d '{"email":"ark3@datawire.io", "name":"Abhay Saxena", "url":"https://datawire.io/"}' -H "Content-Type: application/json" -X POST http://users/api/v1/users
user-0%

[...]

$ curl -s http://users/api/v1/users | jq .
[
  "user-0",
  "user-1",
  "user-2",
  "user-3"
]

$ curl -s http://users/api/v1/users/user-0 | jq .
{
  "ID": "user-0",
  "Created": "2019-04-16T15:49:16.632591892Z",
  "Email": "ark3@datawire.io",
  "Name": "Abhay Saxena",
  "URL": "https://datawire.io/"
}

[...]

$ curl -s http://users/api/v1/posts | jq .
[
  "post-0"
]

$ curl -s http://users/api/v1/posts/post-0 | jq .
{
  "ID": "post-0",
  "Created": "2019-04-16T15:51:56.202068884Z",
  "Title": "Hello World",
  "AuthorID": "user-4",
  "Content": "This is the first post.\nYay!"
}

$ curl -s http://users/api/v1/comments?post=post-0 | jq .
[
  "comment-1",
  "comment-0"
]

$ curl -s http://users/api/v1/comments?post=post-1 | jq .
[]

$ curl -s http://users/api/v1/comments/comment-0 | jq .
{
  "ID": "comment-0",
  "Created": "2019-04-16T15:51:56.290286176Z",
  "AuthorID": "user-5",
  "PostID": "post-0",
  "Content": "First!!!!"
}

$ curl -s http://users/api/v1/users/user-5 | jq .
{
  "ID": "user-5",
  "Created": "2019-04-16T15:51:56.114848875Z",
  "Email": "rhs@datawire.io",
  "Name": "Rafi",
  "URL": ""
}

$ curl -s http://users/api/v1/comments/comment-1 | jq .
{
  "ID": "comment-1",
  "Created": "2019-04-16T15:51:56.336762881Z",
  "AuthorID": "user-4",
  "PostID": "post-0",
  "Content": "You suck, Rafi."
}

```

### Next steps

- Add Swagger stuff or whatever is required to populate the APro dev portal.
- Every time a user, post, or comment is added, the associated service should add an event automatically (for "compliance" tracking). This requires doing a simple service-to-service call and is an opportunity to work on basic Teleproxy integration (`apictl connect` or whatever we choose to call it).
- The "render" APIs should exist (server-side rendering of the blog), which will require more service-to-service calls, plus some clarity on how each service will operate on data from the other services without sharing (much) code. This will exercise multiple simultaneous uses of `apictl intercept` as well as `apictl connect`.
- The services should have durable storage via Google Cloud SQL (or something like that). Some of the services should use environment variables and other in-cluster configuration to access the backing store. Others should use a sidecar container. The backing store should only be accessible from inside the cluster. This will force us to implement some _advanced_ intercept capabilities.
- Some sort of external client (browser-based GUI in JavaScript) should exist. This will require exposing the services to the outside world (via Ambassador). Then we can work on more interesting intercept scenarios (browser cookies or whatever). This will increase the size and scope of what "end-to-end" means for our user.
- The users service should mutate into something that hooks into APro's auth stuff. I haven't really thought this through yet.

## Application

The application is an append-only blog with users who add posts and then comment on those posts. There is a compliance service that records every event in the system; all events are of the form "object _blah_ was added."

The following is somewhat out of date but gets across the general idea.

### Users service

A user is (email address, name, url, join timestamp)

- Add a user
- Get a user's data
- (No edit support)
- Render a user's history (posts and comments)

How does this work with respect to APro auth stuff?

### Posts service

A post is (user, timestamp, title, content)

- Add a post
- Get a post's data
- List posts (by user)
- (No edit support)
- Render a post with comments

### Comments service

A comment is (user, post, timestamp, content)

- Add comment
- Get comments on a post
- Get comments by a user
- Render a single comment (talk to posts and users services)

### Compliance service

Tracks every event in the system. This is duplicative, but that's okay, because the compliance department is separate and has its own needs. As every action is performed by a user, every event has a user. Every event optionally has a post and/or a comment as is appropriate.

An event is (timestamp, user, post?, comment?, content)

- Record user added
- Record post added
- Record comment added
- Render compliance report

## Intent

This application is intended as a foil for development work, not as a useful set of services. We want to add features and complexity to this application using APro's developer-focused feature set ("service preview" etc.), most of which doesn't exist yet, and use that experience to define and then refine our offering.

The various services comprising the application are supposed to represent stuff written by different departments in a large organization. It's probably worthwhile to introduce some code duplication, slightly differing implementations of similar stuff, etc. That will help us _keep things real_.

The following is older content that talks about what the model cluster might be in the long run.

### Service footprint

The model cluster from the perspective of the developer using our tools.

> The idea of the model cluster is to create a "scale model" of our target user's application and the specific service they (in this case a developer) are working on.
> The idea being to capture just the details that are important to the problems we are trying to solve with pro (or in this case with the build subset of pro).
> And part of the rationale/motivation for this is that there are too many of these key details to keep in our heads.
> --- Rafi

- secrets
- volumes
- env vars
- tightly-coupled
  - calls other services
    - with side effects
    - (that are headless)
  - is called by other services
  - calls stuff only visible from the cluster

### Some requirements for the tool

- Intercept a subset of requests
- Access (read-only) portions of the pod filesystem
- Outbound as with swap deployment

### The value to users

- Realtime integration testing
- Multi-tenant dev cluster
