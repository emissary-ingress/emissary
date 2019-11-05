# Getting Started with Ambassador Edge Stack

## 1. Deploying Ambassador Edge Stack to Kubernetes

<div style="border: thick solid red">
<!-- TODO: fix red bordered text -->
Note, the secret.yaml file is temporary during internal Datawire development and can be obtained from the 
<a href="https://drive.google.com/file/d/1q-fmSXU966UtAARrzyCnaKTVbcpkg2n-/view?usp=sharing">Google drive</a>.
</div>

<div class="gatsby-highlight" data-language="shell">
<pre class="language-shell">
<code class="language-shell" id="step1">
kubectl apply -f secret.yaml && \
kubectl apply -f https://deploy-preview-91--datawire-ambassador.netlify.com/yaml/aes-crds.yaml && \
kubectl wait --for condition=established --timeout=60s crd -lproduct=aes && \
kubectl apply -f https://deploy-preview-91--datawire-ambassador.netlify.com/yaml/aes.yaml && \
kubectl -n ambassador wait --for condition=available --timeout=60s deploy -lproduct=aes
</code>
</div>
</div>
<button onclick="copy_to_clipboard('step1')">Copy to Clipboard</button>
<script>
function copy_to_clipboard(the_id) {
  var copyText = document.getElementById(the_id);
  copyText.select();
  copyText.setSelectionRange(0, 99999); /*For mobile devices*/
  document.execCommand("copy");
}
</script>

## 2. Determine your IP Address

Note that it may take a while for your load balancer IP address to be provisioned. Repeat this command as necessary until you get an IP address:

<div class="gatsby-highlight" data-language="shell">
<pre class="language-shell">
<code class="language-shell" id="step2">
kubectl get -n ambassador service ambassador -o 'go-template={{range .status.loadBalancer.ingress}}{{print .ip "\n"}}{{end}}'
</code>
</div>
</div>
<button onclick="copy_to_clipboard('step2')">Copy to Clipboard</button>

## 3. Assign a DNS name (or not)

Navigate to your new IP address in your browser. Assign a DNS name using the providor of your choice to the IP address acquired in Step 2. If you can't/don't want to assign a DNS name, then you can use the IP address you acquired in step 2 instead.

## 4. Complete the install

Go to http://&lt;your-host-name&gt; and follow the instructions to complete the install.

## 5. Temporarily manually type the url

<div style="border: thick solid red">
<!-- TODO: fix red bordered text -->
Temporarily, due to a bug in AES, after the "Complete the install" page shows that it is complete,
you will need to manually enter http://&lt;your-host-name&gt;/admin to get to the next pages of
the user interface.
</div>


## Next Steps

<!-- TODO: should we include this? We've just done a quick tour of some of the core features of Ambassador Edge Stack: diagnostics, routing, configuration, and authentication. -->

- Join us on [Slack](https://d6e.co/slack);
- Learn how to [add authentication](/user-guide/auth-tutorial) to existing services; or
- Learn how to [add rate limiting](/user-guide/rate-limiting-tutorial) to existing services; or
- Learn how to [add tracing](/user-guide/tracing-tutorial); or
- Learn how to [use gRPC with Ambassador Edge Stack](/user-guide/grpc); or
- Read about [configuring Ambassador Edge Stack](/reference/configuration).
