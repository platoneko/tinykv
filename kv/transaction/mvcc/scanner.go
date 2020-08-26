package mvcc

import (
	"bytes"

	"github.com/pingcap-incubator/tinykv/kv/util/engine_util"
)

// Scanner is used for reading multiple sequential key/value pairs from the storage layer. It is aware of the implementation
// of the storage layer and returns results suitable for users.
// Invariant: either the scanner is finished and cannot be used, or it is ready to return a value immediately.
type Scanner struct {
	// Your Data Here (4C).
	nextKey  []byte
	txn      *MvccTxn
	iter     engine_util.DBIterator
	finished bool
}

// NewScanner creates a new scanner ready to read from the snapshot in txn.
func NewScanner(startKey []byte, txn *MvccTxn) *Scanner {
	// Your Code Here (4C).
	return &Scanner{
		nextKey: startKey,
		txn:     txn,
		iter:    txn.Reader.IterCF(engine_util.CfWrite),
	}
}

func (scan *Scanner) Close() {
	// Your Code Here (4C).
	scan.iter.Close()
}

// Next returns the next key/value pair from the scanner. If the scanner is exhausted, then it will return `nil, nil, nil`.
func (scan *Scanner) Next() ([]byte, []byte, error) {
	// Your Code Here (4C).
	if scan.finished {
		return nil, nil, nil
	}
	key := scan.nextKey
	scan.iter.Seek(EncodeKey(key, scan.txn.StartTS))
	if !scan.iter.Valid() {
		scan.finished = true
		return nil, nil, nil
	}
	item := scan.iter.Item()
	gotKey := item.KeyCopy(nil)
	userKey := DecodeUserKey(gotKey)
	if !bytes.Equal(key, userKey) {
		scan.nextKey = userKey
		return scan.Next()
	}
	for {
		scan.iter.Next()
		if !scan.iter.Valid() {
			scan.finished = true
			break
		}
		item := scan.iter.Item()
		gotKey := item.KeyCopy(nil)
		userKey := DecodeUserKey(gotKey)
		if !bytes.Equal(key, userKey) {
			scan.nextKey = userKey
			break
		}
	}
	writeVal, err := item.ValueCopy(nil)
	if err != nil {
		return key, nil, err
	}
	write, err := ParseWrite(writeVal)
	if err != nil {
		return key, nil, err
	}
	if write.Kind == WriteKindDelete {
		return key, nil, nil
	}
	value, err := scan.txn.Reader.GetCF(engine_util.CfDefault, EncodeKey(key, write.StartTS))
	return key, value, err
}
