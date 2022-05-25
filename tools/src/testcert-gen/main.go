// Command testcert-gen generates TLS certificates and keys for use in the Emissary test suite.  The
// program is named `testcert-gen`, not `cert-gen`, because you should absoulutely never ever use
// thi to generate a real cert.
//
// While it would be good if the certs were generated dynamically by the tests, that's not the way
// things are set up, so for the time being, they are certs are generated ahead-of-time by `make
// generate` and checked in to Git.  Because we do this in `make generate`, we need it to be
// deterministic.  Seriously, don't ever use this to generate a real cert.
//
// The cert's private key (despite appearing to be 2048 bits) is really just 64 bits strung out,
// where those 64 bits are deterministically determined by the requested attributes of the cert.
// Seriously, don't ever use this to generate a real cert.
package main

import (
	"context"
	crypto_rand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	math_rand "math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/datawire/dlib/dlog"
)

// deterministicRand is our own particularly-deterministic "randomness" source.
//
// It's even more deterministic than the nominally-deterministic math/rand.Rand, since the crypto
// routines go out of their way to try to break determinism.  Normally, that would be a good
// thing...  for us it's not.  Seriously, don't ever use this to generate a real cert.
type deterministicRand struct {
	inner io.Reader
}

func NewDeterministicRand(seed string) io.Reader {
	djb2Hash := func(str string) int64 {
		hash := uint64(5381)
		for _, c := range str {
			hash = (hash * 33) + uint64(c)
		}
		return int64(hash)
	}
	return &deterministicRand{
		inner: math_rand.New(math_rand.NewSource(djb2Hash(seed))),
	}
}

func (r *deterministicRand) Read(d []byte) (int, error) {
	if len(d) == 1 {
		// We want deterministic keys.  But rsa.GenerateMultiPrimeKey calls
		// crypto/internal/randutil.MaybeReadByte() to make them non-deterministic even when
		// the source of randomness is deterministic.  Shut that down!
		d[0] = 0xcc
		return 1, nil
	}
	return r.inner.Read(d)
}

func main() {
	ctx := context.Background()
	args, err := ParseArgs(os.Args[1:]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: usage error: %v\n", os.Args[0], err)
		os.Exit(2)
	}
	if err := Main(ctx, args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: runtime error: %v\n", os.Args[0], err)
		os.Exit(1)
	}
}

type CLIArgs struct {
	Hosts    []string
	IsCA     bool
	IsServer bool
	IsClient bool

	SignedBy string

	OutCert string
	OutKey  string
}

func ParseArgs(argStrs ...string) (CLIArgs, error) {
	var args CLIArgs
	argparser := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)

	argparser.StringSliceVar(&args.Hosts, "hosts", nil, "comma-separated list of hostnames and IPs to generate a cert for")
	argparser.BoolVar(&args.IsCA, "is-ca", false, "whether this cert should be a Certificate Authority cert")
	argparser.BoolVar(&args.IsServer, "is-server", true, "whether this cert should be a server cert")
	argparser.BoolVar(&args.IsClient, "is-client", false, "whether this cert should be a client cert")

	argparser.StringVar(&args.SignedBy, "signed-by", "self", `either "self" or a "cert.pem,key.pem" pair`)

	argparser.StringVar(&args.OutCert, "out-cert", "cert.pem", "filename to write the cert to")
	argparser.StringVar(&args.OutKey, "out-key", "key.pem", "filename to write the private key to")

	if err := argparser.Parse(argStrs); err != nil {
		return CLIArgs{}, err
	}

	if narg := argparser.NArg(); narg > 0 {
		return CLIArgs{}, fmt.Errorf("expected 0 positional arguments, but got %d", narg)
	}
	if len(args.Hosts) == 0 {
		return CLIArgs{}, fmt.Errorf("missing required --hosts parameter")
	}
	if args.SignedBy != "self" {
		parts := strings.Split(args.SignedBy, ",")
		if len(parts) != 2 {
			return CLIArgs{}, fmt.Errorf("invalid --signed-by: %q", args.SignedBy)
		}
	}

	return args, nil
}

func Main(ctx context.Context, args CLIArgs) (err error) {
	name := fmt.Sprintf("%v,%v,%v,%s", args.IsCA, args.IsServer, args.IsClient, args.Hosts)
	defer func() {
		if err != nil {
			err = fmt.Errorf("%q: %w", name, err)
		}
	}()

	var caCert *x509.Certificate
	var caKey *rsa.PrivateKey
	if args.SignedBy != "self" {
		parts := strings.Split(args.SignedBy, ",")
		caCertPEMBytes, err := ioutil.ReadFile(parts[0])
		if err != nil {
			return fmt.Errorf("read CA cert: %w", err)
		}
		caCertPEM, _ := pem.Decode(caCertPEMBytes)
		if caCertPEM == nil {
			return fmt.Errorf("decode CA cert")
		}
		caCert, err = x509.ParseCertificate(caCertPEM.Bytes)
		if err != nil {
			return fmt.Errorf("parse CA cert: %w", err)
		}

		caKeyPEMBytes, err := ioutil.ReadFile(parts[1])
		if err != nil {
			return fmt.Errorf("read CA key: %w", err)
		}
		caKeyPEM, _ := pem.Decode(caKeyPEMBytes)
		if caKeyPEM == nil {
			return fmt.Errorf("decode CA key")
		}
		_caKey, err := x509.ParsePKCS8PrivateKey(caKeyPEM.Bytes)
		if err != nil {
			return fmt.Errorf("parse CA key: %w", err)
		}
		var ok bool
		if caKey, ok = _caKey.(*rsa.PrivateKey); !ok {
			return fmt.Errorf("CA key is not an RSA key: %w", err)
		}
	}

	// Normally, you'd just use `crypto/rand.Reader`, but as explained above, we want it to be
	// deterministic (not random).  Again, seriously, don't ever do this to generate a real
	// cert.
	randReader := NewDeterministicRand(name)

	privKey, privKeyBytes, err := genKey(PrivArgs{
		Rand: randReader,
	})
	if err != nil {
		return err
	}

	pubBytes, err := genCert(CertArgs{
		CACert: caCert,
		CAKey:  caKey,

		Rand: randReader,

		Key:      privKey,
		IsCA:     args.IsCA,
		IsServer: args.IsServer,
		IsClient: args.IsClient,
		Hosts:    args.Hosts,
	})
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(args.OutCert, pubBytes, 0666); err != nil {
		return fmt.Errorf("writing cert to %q: %w", args.OutCert, err)
	}
	dlog.Printf(ctx, "%q: wrote cert to %q\n", name, args.OutCert)

	if err := ioutil.WriteFile(args.OutKey, privKeyBytes, 0666); err != nil {
		return fmt.Errorf("writing key to %q: %w", args.OutKey, err)
	}
	dlog.Printf(ctx, "%q: wrote key to %q\n", name, args.OutKey)

	return nil
}

type PrivArgs struct {
	Rand io.Reader
}

func genKey(args PrivArgs) (*rsa.PrivateKey, []byte, error) {
	key, err := rsa.GenerateKey(args.Rand, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key-pair: %w", err)
	}

	derBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal private key: %w", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derBytes,
	})

	return key, pemBytes, nil
}

type CertArgs struct {
	CACert *x509.Certificate
	CAKey  *rsa.PrivateKey

	Rand io.Reader

	Key      *rsa.PrivateKey
	IsCA     bool
	IsServer bool
	IsClient bool
	Hosts    []string
}

func genCert(args CertArgs) ([]byte, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := crypto_rand.Int(args.Rand, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:            []string{"US"},
			Province:           []string{"MA"},
			Locality:           []string{"Boston"},
			Organization:       []string{"Ambassador Labs"},
			OrganizationalUnit: []string{"Engineering"},

			CommonName: args.Hosts[0],
		},
		// Some clients get upset if the NotAfter date is too far in the future (in
		// particular: web browsers on macOS).  We don't care about those clients, we only
		// care about KAT and Envoy.
		NotBefore: time.Date(2021, 11, 10, 13, 12, 0, 0, time.UTC),
		NotAfter:  time.Date(2099, 11, 10, 13, 12, 0, 0, time.UTC),

		// If you ever extend this program to generate certs with non-RSA keys, be aware
		// that x509.KeyUsageKeyEncipherment is an RSA-specific thing.
		// https://github.com/golang/go/blob/go1.17.3/src/crypto/tls/generate_cert.go#L88-L93
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{}, // We'll adjust this below.
		BasicConstraintsValid: true,
	}

	for _, h := range args.Hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if args.IsCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}
	if args.IsServer {
		template.ExtKeyUsage = append(template.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
	}
	if args.IsClient {
		template.ExtKeyUsage = append(template.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
	}

	if args.CACert == nil {
		// self-signed
		args.CACert = &template
		args.CAKey = args.Key
	}

	derBytes, err := x509.CreateCertificate(
		args.Rand,           // rand
		&template,           // cert template
		args.CACert,         // parent cert
		&args.Key.PublicKey, // cert pubkey
		args.CAKey)          // parent privkey
	if err != nil {
		return nil, fmt.Errorf("generate certificate: %w", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	return pemBytes, nil
}
