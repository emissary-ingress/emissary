package detectlicense

var (
	reISC = reCompile(reCaseInsensitive(`` +
		`(?:ISC License\s*)?` +
		`(?:Copyright [^\n]*\n)+\s*` +
		reWrap(``+
			`Permission to use, copy, modify, and(?:/or)? distribute this software for any`+"\n"+
			`purpose with or without fee is hereby granted, provided that the above`+"\n"+
			`copyright notice and this permission notice appear in all copies\.`+"\n"+
			"\n"+
			`THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES`+"\n"+
			`WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF`+"\n"+
			`MERCHANTABILITY AND FITNESS\. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR`+"\n"+
			`ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES`+"\n"+
			`WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN`+"\n"+
			`ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF`+"\n"+
			`OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE\.\s*`)))
)
