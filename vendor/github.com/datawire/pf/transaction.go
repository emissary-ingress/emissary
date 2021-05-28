package pf

import (
	"fmt"
	"unsafe"
)

// #include <stdlib.h>
// #include <sys/ioctl.h>
// #include <net/if.h>
// #include <net/pfvar.h>
/*
int init_pfioc_trans(struct pfioc_trans *tx, int elements) {
	tx->size = elements;
	tx->esize = sizeof(struct pfioc_trans_e);
	tx->array = (struct pfioc_trans_e*)calloc(elements,
		sizeof(struct pfioc_trans_e));
	if (tx->array == NULL) {
		return 1;
	}
	return 0;
}

struct pfioc_trans_e* pfioc_trans_result_set(struct pfioc_trans *tx, int index) {
	// assumes the error case (index not in range) is already checked in go
	return &tx->array[index];
}

void free_pfioc_trans(struct pfioc_trans *tx) {
	if (tx->array != NULL) {
		free(tx->array);
	}
	tx->size = 0;
	tx->esize = 0;
}
*/
import "C"

// Transaction represents a pf transaction that can be used to
// add, change or remove rules and rulesets atomically
type Transaction struct {
	handle Handle
	wrap   C.struct_pfioc_trans
}

// NewTransaction creates a new transaction containing the passed number
// of rulesets. Transactions are reusable if the number of result sets
// is not changing. For resuable transactions every transaction must be
// closed by either Commit() or Rollback().
func (h Handle) NewTransaction(numRS int) *Transaction {
	if numRS < 0 {
		panic(fmt.Errorf("Negative number of rule sets invalid: %d", numRS))
	}
	tx := Transaction{handle: h}
	ok := int(C.init_pfioc_trans(&tx.wrap, C.int(numRS)))
	if ok != 0 {
		// TODO: would it be better to return nil in this case?
		panic("Unable to allocate enough memory for transaction rule sets")
	}

	return &tx
}

// Begin opens pf for transaction changes. This happens atomically
// and can fail, if there is currently a transaction open.
func (tx Transaction) Begin() error {
	err := tx.handle.ioctl(C.DIOCXBEGIN, unsafe.Pointer(&tx.wrap))
	if err != nil {
		return fmt.Errorf("DIOCXBEGIN: %s", err)
	}
	return nil
}

// Commit closes the transaction and applies the changes
// that where done since the last Begin() transaction
func (tx Transaction) Commit() error {
	defer C.free_pfioc_trans(&tx.wrap)
	err := tx.handle.ioctl(C.DIOCXCOMMIT, unsafe.Pointer(&tx.wrap))
	if err != nil {
		return fmt.Errorf("DIOCXCOMMIT: %s", err)
	}
	return nil
}

// Rollback removes the kernel side transaction and all
// chnages that where made since the last Begin() transaction are ignored
func (tx Transaction) Rollback() error {
	defer C.free_pfioc_trans(&tx.wrap)
	err := tx.handle.ioctl(C.DIOCXROLLBACK, unsafe.Pointer(&tx.wrap))
	if err != nil {
		return fmt.Errorf("DIOCXROLLBACK: %s", err)
	}
	return nil
}

// RuleSet returns the rule set of o the passed index
func (tx Transaction) RuleSet(index int) *RuleSet {
	if index >= int(tx.wrap.size) || index < 0 {
		panic(fmt.Errorf("RuleSet index out of bounds: %d", index))
	}

	return &RuleSet{tx: tx, wrap: C.pfioc_trans_result_set(&tx.wrap, C.int(index))}
}
