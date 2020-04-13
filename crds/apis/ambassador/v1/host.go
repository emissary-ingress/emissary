package v1

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AmbassadorInstallationSpec defines the desired state of AmbassadorInstallation
// +k8s:deepcopy-gen=package,register
// +groupName=ambassador
type Host struct {
	// API version of the Host
	ApiVersion string `json:"apiVersion"`

	// Kind is Host
	Kind string `json:"kind"`

	// Metadata for Host
	Metadata struct{
		// Name of the Host
		Name string `json:"name"`
	} `json:"metadata"`

	// Spec for the Host
	Spec struct{
		// Hostname for the Host
		Hostname string `json:"hostname"`
		// AcmeProvider details
		AcmeProvider struct{
			// Email to use for the AcmeProvider
			Email string `json:"email"`
		}
	} `json:"spec"`
}
