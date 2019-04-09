package server

import (
	. "github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
	. "github.com/onsi/gomega"
	"sort"
	"testing"
)

func TestDiffCalculator(t *testing.T) {
	g := NewGomegaWithT(t)
	A, B := Service{Name: "a"}, Service{Name: "b"}
	C, D := Service{Name: "c"}, Service{Name: "d"}

	// Starting point: we know about A and B
	calc := NewDiffCalculator([]Service{A, B})

	// Round 1: we detect A and C. That means B should be marked as deleted.
	calc.Add(A)
	calc.Add(C)
	g.Expect(calc.NewRound()).To(Equal([]Service{B}))

	// Round 2: we detect A and C. That means no deletes.
	calc.Add(A)
	calc.Add(C)
	g.Expect(calc.NewRound()).To(Equal([]Service{}))

	// Round 3: we detect A and C and D. That means no deletes.
	calc.Add(A)
	calc.Add(C)
	calc.Add(D)
	g.Expect(calc.NewRound()).To(Equal([]Service{}))

	// Round 4: we detect B and C. That means A and D are deleted.
	calc.Add(B)
	calc.Add(C)
	result := calc.NewRound()
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	g.Expect(result).To(Equal([]Service{A, D}))

	// Round 5: we detect nothing. That means B and C are deleted.
	result = calc.NewRound()
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	g.Expect(result).To(Equal([]Service{B, C}))
}
