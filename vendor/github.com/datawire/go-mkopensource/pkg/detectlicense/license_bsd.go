package detectlicense

var (
	bsdHeader = `(?:BSD [123]-Clause License\n)?\s*` +
		`(?:Copyright [^\n]*(?:\s+All rights reserved\.)? *\n)+\s*`
	bsdPrefix = `` +
		`Redistribution and use in source and binary forms, with or without` + "\n" +
		`modification, are permitted provided that the following conditions are` + "\n" +
		`met:` + "\n"
	bsdClause1 = `` +
		` (?:1?[*.-] )?Redistributions of source code must retain the above copyright` + "\n" +
		`      notice, this list of conditions and the following disclaimer\.` + "\n"
	bsdClause2 = `` +
		` (?:2?[*.-] )?Redistributions in binary form must reproduce the above` + "\n" +
		`      copyright notice,? this list of conditions and the following disclaimer` + "\n" +
		`      in the documentation and/or other materials provided with the` + "\n" +
		`      distribution\.` + "\n"
	bsdClause3 = `` +
		` (?:3?[*.-] )?(?:Neither the .{1,80} nor the names of its contributors may|(?:My name, .{1,80}|The names of its contributors) may not)` + "\n" +
		`      be used to endorse or promote products derived from this software` + "\n" +
		`      without specific prior written permission\.` + "\n"
	bsdSuffix = `` +
		`THIS SOFTWARE IS PROVIDED BY .{1,80} AND CONTRIBUTORS` + "\n" +
		`"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT` + "\n" +
		`LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR` + "\n" +
		`A PARTICULAR PURPOSE ARE DISCLAIMED\. IN NO EVENT SHALL THE COPYRIGHT` + "\n" +
		`(OWNER|HOLDER)( OR CONTRIBUTORS)? BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,` + "\n" +
		`SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES \(INCLUDING, BUT NOT` + "\n" +
		`LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,` + "\n" +
		`DATA, OR PROFITS; OR BUSINESS INTERRUPTION\) HOWEVER CAUSED AND ON ANY` + "\n" +
		`THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT` + "\n" +
		`\(INCLUDING NEGLIGENCE OR OTHERWISE\) ARISING IN ANY WAY OUT OF THE USE` + "\n" +
		`OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE\.\s*`
)

var (
	reBSD3 = reCompile(reCaseInsensitive(`` +
		bsdHeader +
		reWrap(``+
			bsdPrefix+
			bsdClause1+
			bsdClause2+
			bsdClause3+
			bsdSuffix)))
	reBSD2 = reCompile(reCaseInsensitive(`` +
		bsdHeader +
		reWrap(``+
			bsdPrefix+
			bsdClause1+
			bsdClause2+
			bsdSuffix)))
)
