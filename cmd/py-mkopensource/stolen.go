package main

// FIXME(lukeshu): This file is stuff copied from go-mkopensource.
// Move things around so that it can be shared.

type License struct {
	Name           string
	NoticeFile     bool // are NOTICE files "a thing" for this license?
	WeakCopyleft   bool // requires that library to be open-source
	StrongCopyleft bool // requires the resulting program to be open-source
}

var (
	Proprietary = License{Name: "proprietary"}

	PublicDomain = License{Name: "public domain"}

	Apache2 = License{Name: "Apache License 2.0", NoticeFile: true}
	BSD1    = License{Name: "1-clause BSD license"}
	BSD2    = License{Name: "2-clause BSD license"}
	BSD3    = License{Name: "3-clause BSD license"}
	ISC     = License{Name: "ISC license"}
	MIT     = License{Name: "MIT license"}
	MPL2    = License{Name: "Mozilla Public License 2.0", NoticeFile: true, WeakCopyleft: true}

	CcBySa40 = License{Name: "Creative Commons Attribution Share Alike 4.0 International", StrongCopyleft: true}
)
