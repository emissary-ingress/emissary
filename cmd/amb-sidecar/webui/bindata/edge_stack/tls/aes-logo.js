export default {
	template: `<img v-bind:src="url" />`,
	data: function() {
		return {
			ambassadorClusterID: null,
		};
	},
	computed: {
		url: function() {
			if (this.ambassadorClusterID === null) {
				return ""
			}
			return "https://getambassador.io/images/ambassador-logo.svg?" + this.ambassadorClusterID;
		}
	},
	beforeMount: function() {
		let req = new XMLHttpRequest();
		req.open("GET", "../api/config/ambassador-cluster-id");
		req.onload = () => {
			if (req.status == 200) {
				this.ambassadorClusterID = req.response;
			}
		};
		req.send();
	},
	methods: {},
};
