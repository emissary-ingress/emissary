export default {
	template: `<div v-if="state=='loading'">
	<p>Loading...</p>
</div>
<div v-else-if="state=='unauthorized'">
	<p>TODO: They get a page explaining that in order to
	authenticate they need to download edgectl and run
	<code>edgectl login</code>. The page includes a link they can
	click on to download edgectl.</p>
</div>
<div v-else-if="state=='authorized'">
	<slot></slot>
</div>
<div v-else>
	<p>Error: {{ error }}</p>
</div>`,
	data: function() {
		return {
			state: "loading",
			error: "",
		};
	},
	beforeMount: function() {
		let req = new XMLHttpRequest();
		req.open("GET", "api/empty");
		req.setRequestHeader("Authorization", "Bearer " + window.location.hash.slice(1));
		req.onload = () => {
			switch (req.status) {
			case 200:
				this.state = "authorized";
				break;
			case 403:
				this.state = "unauthorized";
				break;
			default:
				this.state = "error";
				this.error = "HTTP "+req.status+": "+req.response;
			}
		};
		req.onerror = () => {
			this.state = "error";
			this.error = "network error";
		};
		req.send();
	},
};
