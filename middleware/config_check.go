package middleware

import (
	"fmt"
	"net/http"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
)

// CheckConfig verifies that configuration is set and display a friendly error
// message
type CheckConfig struct {
	Config *config.Config
}

func (c *CheckConfig) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	err := c.Config.Validate()
	if err != nil {
		body := []byte(fmt.Sprintf(`
<html>
<head>
</head>
<body>
<h2>Service is not Configured</h2>
The authentication service is not configured:
<blockquote>
<pre style="background: #eeeeee;display: inline-block;">
%v
</pre>
</blockquote
<p>
Please see <a href="https://www.getambassador.io/docs">https://www.getambassador.io/docs</a> for details on how
to configure this service.
</p>
</body>
</html>
`, err))
		rw.WriteHeader(http.StatusServiceUnavailable)
		rw.Write(body)
		return
	}

	next(rw, r)
}
