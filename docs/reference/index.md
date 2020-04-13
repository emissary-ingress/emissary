<p>Packages:</p>
<ul>
<li>
<a href="#ambassador%2fv1">ambassador/v1</a>
</li>
</ul>
<h2 id="ambassador/v1">ambassador/v1</h2>
<p>
<p>Package v1 is the v1 version of the API.</p>
</p>
Resource Types:
<ul></ul>
<h3 id="ambassador/v1.Host">Host
</h3>
<p>
<p>AmbassadorInstallationSpec defines the desired state of AmbassadorInstallation</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
<em>
string
</em>
</td>
<td>
<p>API version of the Host</p>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
<em>
string
</em>
</td>
<td>
<p>Kind is Host</p>
</td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
struct{Name string &#34;json:\&#34;name\&#34;&#34;}
</em>
</td>
<td>
<p>Metadata for Host</p>
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
struct{Hostname string &#34;json:\&#34;hostname\&#34;&#34;; AcmeProvider struct{Email string &#34;json:\&#34;email\&#34;&#34;}}
</em>
</td>
<td>
<p>Spec for the Host</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>hostname</code></br>
<em>
string
</em>
</td>
<td>
<p>Hostname for the Host</p>
</td>
</tr>
<tr>
<td>
<code>AcmeProvider</code></br>
<em>
struct{Email string &#34;json:\&#34;email\&#34;&#34;}
</em>
</td>
<td>
<p>AcmeProvider details</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="ambassador/v1.Test">Test
</h3>
<p>
<p>AmbassadorInstallationSpec defines the desired state of AmbassadorInstallation</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
<em>
string
</em>
</td>
<td>
<p>API version of the Test</p>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
<em>
string
</em>
</td>
<td>
<p>Kind is Test</p>
</td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
struct{Name string &#34;json:\&#34;name\&#34;&#34;}
</em>
</td>
<td>
<p>Metadata for Test</p>
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
struct{Testname string &#34;json:\&#34;Testname\&#34;&#34;; AcmeProvider struct{Email string &#34;json:\&#34;email\&#34;&#34;}}
</em>
</td>
<td>
<p>Spec for the Test</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>Testname</code></br>
<em>
string
</em>
</td>
<td>
<p>Testname for the Test</p>
</td>
</tr>
<tr>
<td>
<code>AcmeProvider</code></br>
<em>
struct{Email string &#34;json:\&#34;email\&#34;&#34;}
</em>
</td>
<td>
<p>AcmeProvider details</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
.
</em></p>
