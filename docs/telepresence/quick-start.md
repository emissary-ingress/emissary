---
description: "Install Telepresence and learn to use it to intercept services running in your Kubernetes cluster, speeding up local development and debugging."
---

import Alert from '@material-ui/lab/Alert';
import QSTabs from './qs-tabs'
import QSCards from './qs-cards'

# Telepresence - Quick Start

## Prereqs
You need `kubectl` installed and configured to use a Kubernetes cluster, preferably an empty test cluster.  You must have RBAC permissions in the cluster to create and update deployments and services.

## 1. Install Telepresence

<QSTabs/>

## 2. Test Telepresence

Telepresence connects your local workstation to a remote Kubernetes cluster. 

1. Connect to the cluster:  
`telepresence connect`

  ```
  $ telepresence connect
    
    Launching Telepresence Daemon
    ...
    Connected to context default (https://<cluster-public-IP>)
  ```

  <Alert severity="info"> macOS users: If you receive an error when running Telepresence that the developer cannot be verified, open <b>System Preferences → Security & Privacy → General</b>. Click <b>Open Anyway</b> at the bottom to bypass the security block. Then retry the <code>telepresence connect</code> command.</Alert>

2. Test that Telepresence is working properly by connecting to the Kubernetes API server:  
`curl -ik https://kubernetes`

  ```
  $ curl -ik https://kubernetes
    
    HTTP/1.1 401 Unauthorized
    Cache-Control: no-cache, private
    Content-Type: application/json
    Www-Authenticate: Basic realm="kubernetes-master"
    Date: Tue, 09 Feb 2021 23:21:51 GMT
    Content-Length: 165  
    
    {
      "kind": "Status",
      "apiVersion": "v1",
      "metadata": {  
    
      },
      "status": "Failure",
      "message": "Unauthorized",
      "reason": "Unauthorized",
      "code": 401
    }%  

  ```
<Alert severity="info">The 401 response is expected.  What's important is that you were able to contact the API.</Alert>

<Alert severity="success"><b>Congratulations! You’ve just accessed your remote Kubernetes API server, as if you were on the same network!</b> With Telepresence, you’re able to use any tool that you have locally to connect to any service in the cluster.</Alert>

## 3. Install a sample application

Your local workstation may not have the compute or memory resources necessary to run all the services in a multi-service application. In this example, we’ll show you how Telepresence can give you a fast development loop, even in this situation.

1. Start by installing a sample application that consists of multiple services:  
`kubectl apply -f https://raw.githubusercontent.com/datawire/edgey-corp-nodejs/main/k8s-config/edgey-corp-web-app-no-mapping.yaml`

  ```
  $ kubectl apply -f https://raw.githubusercontent.com/datawire/edgey-corp-nodejs/main/k8s-config/edgey-corp-web-app-no-mapping.yaml
    
    deployment.apps/dataprocessingnodeservice created
    service/dataprocessingnodeservice created
    ...  

  ```

2. Give your cluster a few moments to deploy the sample application.

  Use `kubectl get pods --watch` to watch your pods:  

  ```
  $ kubectl get pods --watch
    
    NAME                                         READY   STATUS    RESTARTS   AGE
    traffic-manager-f8c64686-8f4jn               1/1     Running   0          2m47s
    verylargedatastore-855c8b8789-z8nhs          1/1     Running   0          78s
    verylargejavaservice-7dfddbc95c-696br        1/1     Running   0          78s
    dataprocessingnodeservice-5f6bfdcf7b-qvd27   1/1     Running   0          79s
  ```

3. Once all the pods are in a `Running` status, stop the `watch` command with `Ctrl+C`.  Then go to the frontend service in your browser at [http://verylargejavaservice:8080](http://verylargejavaservice:8080).

4. You should see the EdgyCorp WebApp with a <span style="color:green" class="bold">green</span> title and <span style="color:green" class="bold">green</span> pod in the diagram.

<Alert severity="success"><b>Congratulations, you can now access services running in your cluster by name from your laptop!</b></Alert>

## 4. Set up a local development environment
You will now download the repo containing the services' code and run the DataProcessingNodeService service locally. This version of the code has the UI color set to <span style="color:blue" class="bold">blue</span> instead of <span style="color:green" class="bold">green</span>.

1. Clone the web app’s GitHub repo:  
`git clone https://github.com/datawire/edgey-corp-nodejs.git`

  ```
  $ git clone https://github.com/datawire/edgey-corp-nodejs.git
    
    Cloning into 'edgey-corp-nodejs'...
    remote: Enumerating objects: 441, done.
    ...
  ```

2. Change into the repo directory, then into DataProcessingNodeService:  
`cd edgey-corp-nodejs/DataProcessingNodeService/`

3. Install the Node dependencies and start the Node server:  
`npm install && npm start`

  ```
  $ npm install && npm start
    
    ...
    Welcome to the DataProcessingNodeService!
    { _: [] }
    Server running on port 3000
  ```

  <Alert severity="info"><a href="https://nodejs.org/en/download/package-manager/">Install Node.js from here</a> if needed.</Alert>

4. In a **new terminal window**, curl the service running locally to confirm it’s set to <span style="color:blue" class="bold">blue</span>:  
`curl localhost:3000/color`

  ```
  $ curl localhost:3000/color
    
    “blue”
  ```

<Alert severity="success"><b>Victory, your local Node server is running a-ok!</b></Alert>

## 5. Intercept all traffic to the service
Next, we’ll create an intercept. An intercept is a rule that tells Telepresence where to send traffic. In this example, we will send all traffic destined for the DataProcessingNodeService to the version of the DataProcessingNodeService running locally instead: 

1. Start the intercept with the `intercept` command, setting the service name and port:  
`telepresence intercept dataprocessingnodeservice --port 3000`

  ```
  $ telepresence intercept dataprocessingnodeservice --port 3000
    
    Using deployment dataprocessingnodeservice
    intercepted
        State       : ACTIVE
        Destination : 127.0.0.1:3000
        Intercepting: all connections
  ```

2. Go to the frontend service again in your browser at [http://verylargejavaservice:8080](http://verylargejavaservice:8080). You will now see the <span style="color:blue" class="bold">blue</span> elements in the app.  

<Alert severity="success"><b>The frontend’s request to DataProcessingNodeService is being intercepted and rerouted to the Node server on your laptop!</b></Alert>

## 6. Make a code change
We’ve now set up a local development environment for the DataProcessingNodeService, and we’ve created an intercept that sends traffic in the cluster to our local environment. We can now combine these two concepts to show how we can quickly make and test changes.

1. Open `edgey-corp-nodejs/DataProcessingNodeService/app.js` in your editor and change line 6 from `blue` to `orange`. Save the file and the Node server will auto reload.

2. Now, visit [http://verylargejavaservice:8080](http://verylargejavaservice:8080) again in your browser. You will now see the orange elements in the application.

<Alert severity="success"><b>We’ve just shown how we can edit code locally, and immediately see these changes in the cluster.</b> Normally, this process would require a container build, push to registry, and deploy. With Telepresence, these changes happen instantly.</Alert>

## 7. Create a Preview URL
Create preview URLs to do selective intercepts, meaning only traffic coming from the preview URL will be intercepted, so you can easily share the services you’re working on with your teammates.

1. Clean up your previous intercept by removing it:  
`telepresence leave dataprocessingnodeservice`

2. Login to Ambassador Cloud, a web interface for managing and sharing preview URLs:  
`telepresence login`  

  This opens your browser; login with your GitHub account and choose your org.  

  ```
  $ telepresence login
    
    Launching browser authentication flow...
    <browser opens, login with GitHub>
    Login successful.
  ```

3. Start the intercept again:  
`telepresence intercept dataprocessingnodeservice --port 3000`  
  You will be asked for your ingress; specify the front end service: `verylargejavaservice.default`  
  Then when asked for the port, type `8080`.  
  Finally, type `n` for “Use TLS”.

  ```
    $ telepresence intercept dataprocessingnodeservice --port 3000
      
      Confirm the ingress to use for preview URL access
      Ingress service.namespace ? verylargejavaservice.default
      Port ? 8080
      Use TLS y/n ? n
      Using deployment dataprocessingnodeservice
      intercepted
          State       : ACTIVE
          Destination : 127.0.0.1:3000
          Intercepting: HTTP requests that match all of:
            header("x-telepresence-intercept-id") ~= regexp("86cb4a70-c7e1-1138-89c2-d8fed7a46cae:dataprocessingnodeservice")
          Preview URL : https://<random-subdomain>.preview.edgestack.me  
  ```

4. Wait a moment for the intercept to start; it will also output a preview URL.  Go to this URL in your browser, it will be the <span style="color:orange" class="bold">orange</span> version of the app.

5. Now go again to [http://verylargejavaservice:8080](http://verylargejavaservice:8080), it’s still <span style="color:green" class="bold">green</span>.

Normal traffic coming to your app gets the <span style="color:green" class="bold">green</span> cluster service, but traffic coming from the preview URL goes to your laptop and gets the <span style="color:orange" class="bold">orange</span> local service!
<Alert severity="success"><b>The Preview URL now shows exactly what is running on your local laptop -- in a way that can be securely shared with anyone you work with.</b></Alert>

## <img class="os-logo" src="../../images/logo.png"/> What's Next?

<QSCards/>