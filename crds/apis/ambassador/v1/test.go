package v1

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AmbassadorInstallationSpec defines the desired state of AmbassadorInstallation
// +k8s:deepcopy-gen=package,register
// +groupName=ambassador
type Test struct {
	// API version of the Test
	ApiVersion string `json:"apiVersion"`

	// Kind is Test
	Kind string `json:"kind"`

	// Metadata for Test
	Metadata struct{
		// Name of the Test
		Name string `json:"name"`
	} `json:"metadata"`

	// Spec for the Test
	Spec struct{
		// Testname for the Test
		Testname string `json:"Testname"`
		// AcmeProvider details
		AcmeProvider struct{
			// Email to use for the AcmeProvider
			Email string `json:"email"`
		}
	} `json:"spec"`
}
