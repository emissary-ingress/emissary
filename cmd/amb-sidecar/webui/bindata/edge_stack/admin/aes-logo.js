export default {
	template: `<img v-bind:src="url" />`,
	data: function() {
		return {
			ambassadorClusterID: "",
		};
	},
	computed: {
		url: function() {
			if (this.ambassadorClusterID == "") {
				return ""
			}
			return "https://getambassador.io/images/ambassador-logo.svg?" + this.ambassadorClusterID;
		}
	},
	beforeMount: function() {
		let req = new XMLHttpRequest();
		req.open("GET", "api/ambassador_cluster_id");
		req.onload = () => {
			if (req.status == 200) {
				this.ambassadorClusterID = req.response;
			}
		};
		req.send();
	},
	methods: {},
};
