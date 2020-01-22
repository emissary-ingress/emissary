package licensekeys

import (
	"crypto/rsa"
	"fmt"
	"math"
	"math/big"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/datawire/apro/lib/jwtsupport"
)

const (
	DefUnregisteredCustomerID = "unregistered"
)

type LicenseClaimsLatest = LicenseClaimsV2

type LicenseClaims interface {
	ToLatest() *LicenseClaimsLatest
	jwt.Claims
}

type LicenseClaimsV0 struct {
	ID interface{} `json:"id"`
	jwt.StandardClaims
}

func (v0 *LicenseClaimsV0) ToLatest() *LicenseClaimsLatest {
	v1 := &LicenseClaimsV1{
		LicenseKeyVersion: "v0",
		CustomerID:        fmt.Sprintf("%v", v0.ID),
		EnabledFeatures: []Feature{
			FeatureCertifiedEnvoy,
			FeatureFilter,
			FeatureRateLimit,
			FeatureTraffic,
		},
		StandardClaims: v0.StandardClaims,
	}
	return v1.ToLatest()
}

type LicenseClaimsV1 struct {
	LicenseKeyVersion string    `json:"license_key_version"`
	CustomerID        string    `json:"customer_id"`
	EnabledFeatures   []Feature `json:"enabled_features"`
	jwt.StandardClaims
}

func (v1 *LicenseClaimsV1) ToLatest() *LicenseClaimsLatest {
	v2 := &LicenseClaimsV2{
		LicenseKeyVersion: v1.LicenseKeyVersion,
		CustomerID:        v1.CustomerID,
		CustomerEmail:     "",
		EnabledFeatures:   v1.EnabledFeatures,
		StandardClaims:    v1.StandardClaims,
		EnforcedLimits: []LimitValue{
			// Make v1 license virtually unlimited.
			{LimitDevPortalServices, math.MaxUint32},
			{LimitRateLimitService, math.MaxUint32},
			{LimitAuthFilterService, math.MaxUint32},
		},
		Metadata: map[string]string{},
	}

	// Assuming all pre-v2 licenses have business-support, even if the feature flag was added afterwards
	shouldAddPaidSupport := true
	for _, feature := range v2.EnabledFeatures {
		if feature == FeatureSupportBusinessTier {
			shouldAddPaidSupport = false
		}
	}
	if shouldAddPaidSupport {
		v2.EnabledFeatures = append(v2.EnabledFeatures, FeatureSupportBusinessTier)
	}

	return v2.ToLatest()
}

type LicenseClaimsV2 struct {
	LicenseKeyVersion string            `json:"license_key_version"`
	CustomerID        string            `json:"customer_id"`
	CustomerEmail     string            `json:"customer_email"`
	EnabledFeatures   []Feature         `json:"enabled_features"`
	EnforcedLimits    []LimitValue      `json:"enforced_limits"`
	Metadata          map[string]string `json:"metadata"`
	jwt.StandardClaims
}

type LimitValue struct {
	Name  Limit `json:"l"`
	Value int   `json:"v"`
}

func (v2 *LicenseClaimsV2) ToLatest() *LicenseClaimsLatest {
	return v2
}

func (limit LimitValue) String() string {
	return fmt.Sprintf("%v=%v", limit.Name, limit.Value)
}

func newBigIntFromBytes(bs []byte) *big.Int {
	ret := big.NewInt(0)
	ret.SetBytes(bs)
	return ret
}

func ParseKey(licenseKey string) (*LicenseClaimsLatest, error) {
	var mapClaims jwt.MapClaims
	_, _, err := jwtsupport.SanitizeParseUnverified(new(jwt.Parser).ParseUnverified(licenseKey, &mapClaims))
	if err != nil {
		return nil, err
	}

	// these details should match the details in apictl-key
	var licenseClaims LicenseClaims
	var signingMethod string
	var signingKey interface{}
	if version, versionOK := mapClaims["license_key_version"].(string); !versionOK {
		licenseClaims = &LicenseClaimsV0{}
		signingMethod = "HS256"
		signingKey = []byte("1234")
	} else {
		switch version {
		case "v1":
			licenseClaims = &LicenseClaimsV1{}
		case "v2":
			licenseClaims = &LicenseClaimsV2{}
		default:
			return nil, errors.Errorf("invalid license key: unrecognized license key version %q", version)
		}
		signingMethod = "PS512"
		// `signingKey` is from the output of `apictl-key pubkey`
		signingKey = &rsa.PublicKey{
			//nolint:dupl
			N: newBigIntFromBytes([]byte{0xa0, 0xb, 0x79, 0xac, 0xa4, 0x43, 0x23, 0xcd, 0x26, 0xaf, 0xa2, 0x85, 0x5e, 0xe7, 0xcb, 0x2e, 0xc8, 0x89, 0x7f, 0x68, 0x39, 0x72, 0xfe, 0x68, 0x6a, 0xef, 0x9a, 0x3f, 0x65, 0xf1, 0x49, 0x93, 0x80, 0xb8, 0xd6, 0xe2, 0x4a, 0x2a, 0x60, 0xfe, 0xfb, 0xc8, 0xea, 0xf9, 0x8, 0xe5, 0x51, 0x69, 0xf1, 0xba, 0x11, 0xe1, 0xe8, 0xec, 0xf7, 0xb9, 0xff, 0x20, 0x60, 0x6, 0xed, 0x27, 0xc4, 0x8e, 0xe7, 0x15, 0x6a, 0xd9, 0x3b, 0x1c, 0x56, 0x1e, 0x58, 0xd2, 0xe6, 0x2d, 0xf9, 0xb7, 0xa6, 0x2f, 0x1b, 0xa, 0xb, 0x4a, 0x5d, 0x34, 0xd9, 0x14, 0x9f, 0x1d, 0x58, 0xd4, 0x3d, 0xfb, 0xfd, 0x87, 0x0, 0xe3, 0xaa, 0xd8, 0x2, 0x95, 0x52, 0x29, 0xe3, 0x2, 0x50, 0x1c, 0xb7, 0xa7, 0xc8, 0xbb, 0xcb, 0x20, 0x80, 0x7e, 0xa5, 0x28, 0x3d, 0x83, 0x14, 0xe8, 0x3f, 0x5c, 0xe6, 0x6b, 0xfa, 0xa8, 0x60, 0xbd, 0xc3, 0x92, 0xb5, 0xf2, 0x57, 0x66, 0x55, 0xae, 0xa3, 0xad, 0xa7, 0xcb, 0x17, 0xd4, 0xc3, 0xee, 0x77, 0x9c, 0xb6, 0x68, 0xe1, 0x4d, 0x64, 0xec, 0x7c, 0x73, 0x34, 0x92, 0xfc, 0x63, 0xf8, 0x92, 0x39, 0x10, 0x54, 0xc4, 0x43, 0x1f, 0xd5, 0x85, 0x7f, 0x44, 0xa4, 0x18, 0x6c, 0xc6, 0x75, 0x8c, 0x11, 0x6e, 0xa1, 0xa, 0xbb, 0xe1, 0x42, 0x9d, 0x16, 0x12, 0x18, 0x9f, 0x81, 0x12, 0xb7, 0xab, 0x97, 0x3a, 0x27, 0xa6, 0x43, 0x2d, 0xb0, 0xf7, 0xa2, 0x6b, 0xeb, 0x56, 0xd1, 0x94, 0x55, 0x7e, 0x59, 0x6c, 0x48, 0xcb, 0x49, 0x66, 0xe2, 0x6e, 0x4f, 0xa, 0xa7, 0x94, 0x2e, 0x4a, 0xe, 0xee, 0x8e, 0x85, 0xfe, 0xb5, 0x17, 0x38, 0xe0, 0x95, 0x5, 0x9c, 0xc3, 0x4b, 0x4e, 0x5, 0x50, 0x6e, 0x44, 0x5d, 0xf8, 0xc0, 0x83, 0x7, 0xbe, 0xec, 0xd1, 0x78, 0xe8, 0xcc, 0xa8, 0x8b}),
			E: 65537,
		}
	}

	jwtParser := &jwt.Parser{ValidMethods: []string{signingMethod}}
	_, err = jwtsupport.SanitizeParse(jwtParser.ParseWithClaims(licenseKey, licenseClaims, func(token *jwt.Token) (interface{}, error) {
		return signingKey, nil
	}))
	if err != nil {
		return nil, err
	}
	return licenseClaims.ToLatest(), nil
}

func NewCommunityLicenseClaims() *LicenseClaimsLatest {
	return &LicenseClaimsLatest{
		EnabledFeatures: []Feature{
			FeatureUnrecognized,
			FeatureFilter,
			FeatureRateLimit,
			FeatureTraffic,
			FeatureDevPortal,
		},
		EnforcedLimits: []LimitValue{
			{LimitDevPortalServices, 5},
			{LimitRateLimitService, 5},
			{LimitAuthFilterService, 5},
		},
		Metadata: map[string]string{},
	}
}
