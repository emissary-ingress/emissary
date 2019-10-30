import Vue from 'https://cdn.jsdelivr.net/npm/vue/dist/vue.esm.browser.min.js';

export default Vue.extend({
	template: `<div>
<form v-on:submit.prevent="onSubmit">
		<fieldset>
			<legend>Initial TLS setup</legend>
			<label>
				<span>Hostname to request a certificate for:</span>
				<div>
					<input type="text" name="hostname" v-model:value="hostname"
						v-on:input="onHostnameChange" />
					<p v-if="!hostnameChanged">We auto-filled this from the URL
						you used to access this web page.  Feel free to change it, though.</p>
				</div>
			</label>
			<label>
				<span>ACME provider:</span>
				<input type="url" name="provider" v-model:value="provider"
					v-on:input="onProviderChange" />
			</label>
			<label>
				<input type="checkbox" name="tos_agree" v-model:value="tosAgree" v-bind:disabled="tosReq || tosProvider !== provider || tosErr != null" />
				<div v-if="tosReq">
					<p>Loading spinner goes here</p>
				</div>
				<div v-else-if="tosErr">
					<p><q>{{ tosProvider }}</q> does not appear to refer to valid ACME provider: <q>{{ tosErr }}</q>. <button v-on:click="onProviderChange">Retry</button></p>
				</div>
				<div v-else>
					<p>I have agreed to to the Terms of Service at <a v-bind:href="tosURL">{{ tosURL }}</a>.</p>
				</div>
			</label>
			<label>
				<span>Email:</span>
				<input type="email" name="email" v-model:value="email"
					v-on:input="updateYAML" />
			</label>
		</fieldset>

		<pre>{{ yaml }}</pre>
		<input type="submit" value="Apply" />
	</form>
	<div>{{ output }}</div>
</div>`,
	data: function() {
		return {
			hostname: window.location.hostname,
			provider: "https://acme-staging-v02.api.letsencrypt.org/directory", // "https://acme-v02.api.letsencrypt.org/directory",
			tosAgree: false,
			email: "",

			hostnameChanged: false,

			tosReq: null,
			tosErr: null,
			tosProvider: "",
			tosURL: null,

			yamlReq: null,
			yaml: "",

			output: "",
		};
	},
	computed: {},
	beforeMount: function() {
		this.onProviderChange(null);
	},
	methods: {
		onHostnameChange: function(event) {
			this.hostnameChanged = true;
			this.updateYAML();
		},
		onProviderChange: function(event) {
			let url = new URL('tos-url', window.location);
			url.searchParams.set('ca-url', this.provider);

			let provider = this.provider; // capture for the closures below

			let req = new XMLHttpRequest();
			req.open("GET", url.toString());
			req.onload = () => {
				if (req.status == 200) {
					this.tosURL = req.response;
					this.tosErr = null;
				} else {
					this.tosURL = null;
					this.tosErr = req.response;
				}
				this.tosProvider = provider;
				this.tosReq = null;
			};
			req.onerror = () => {
				this.tosURL = null;
				this.tosErr = "XmlHttpRequest error";
				this.tosProvider = provider;
				this.tosReq = null;
			};
			if (this.tosReq != null) {
				this.tosReq.abort();
			}
			this.tosReq = req;
			this.tosAgree = false;
			req.send();
			this.updateYAML();
		},
		refreshStatus: function() {
			let url = new URL('status', window.location);
			url.searchParams.set('host', this.hostname);

			let req = new XMLHttpRequest();
			req.open("GET", url.toString());
			req.onload = () => {
				if (req.status == 200) {
					this.output = req.response;
				}
			};
			req.onloadend = () => {
				// recurse, to continually refresh the output status
				this.refreshStatus();
			};
			req.send();
		},
		onSubmit: function(event) {
			let url = new URL('yaml', window.location);
			url.searchParams.set('hostname', this.hostname);
			url.searchParams.set('acme_authority', this.provider);
			url.searchParams.set('acme_email', this.email);

			let req = new XMLHttpRequest();
			req.open("POST", url.toString());
			req.onload = () => {
				if (req.status == 201) {
					this.output = "Applying YAML...";
					this.refreshStatus();
				} else {
					this.output += "Error applying YAML: "+req.response;
				}
			};
			req.onerror = () => {
				this.output += "Error applying YAML: "+"XmlHttpRequestError";
			};
			req.send();
		},
		updateYAML: function() {
			let url = new URL('yaml', window.location);
			url.searchParams.set('hostname', this.hostname);
			url.searchParams.set('acme_authority', this.provider);
			url.searchParams.set('acme_email', this.email);

			let req = new XMLHttpRequest();
			req.open("GET", url.toString());
			req.onload = () => {
				if (req.status == 200) {
					this.yaml = req.response;
				}
				this.yamlReq = null;
			};
			if (this.yamlReq != null) {
				this.yamlReq.abort();
			}
			this.yamlReq = req;
			req.send();
		},
	},
});
