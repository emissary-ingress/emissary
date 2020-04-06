# edgectl install: requesting your email address
 
## The Problem

When you run `edgectl install` to install the Ambassador Edge Stack, the system requests your email address.
This email address is used to notify you before your TLS certificate and domain name expire. In order to
acquire the TLS certificate, we share this email with Let’s Encrypt.

For some reason this request failed.

## How to Resolve It

Please retry `edgectl install` and enter a valid email address.
