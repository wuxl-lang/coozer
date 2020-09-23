package coozer

import "errors"

var (
	// ErrNoAddrs indicates unknown ip address
	ErrNoAddrs 		= errors.New("unknown address")
	// ErrBadTag indicates bad tag
	ErrBadTag 		= errors.New("bad tag")
	// ErrClosed indicates ?
	ErrClosed 		= errors.New("closed")
	// ErrWaitTimeout indicates timeout
	ErrWaitTimeout 	= errors.New("wait timeout")
)

var (
	// ErrOther is default error
	ErrOther    Response_Err = Response_OTHER
	// ErrNotDir indicates it is not a directory
	ErrNotDir   Response_Err = Response_NOTDIR
	// ErrIsDir indicates it is a directory
	ErrIsDir    Response_Err = Response_ISDIR
	// ErrNotEnt ?
	ErrNotEnt   Response_Err = Response_NOENT
	// ErrRange ?
	ErrRange    Response_Err = Response_RANGE
	// ErrOldRev indicates the given revision is old
	ErrOldRev   Response_Err = Response_REV_MISMATCH
	// ErrTooLate ?
	ErrTooLate  Response_Err = Response_TOO_LATE
	// ErrReadOnly indicates it is read only
	ErrReadOnly Response_Err = Response_READONLY
)

// Error is customized structure to descripe errors 
type Error struct {
	Err error
	Detail string
}

/**
* Implement error interface type
**/
func (x Response_Err) Error() string {
	return x.String()
}

/**
* -- Return Pointer from a Function --
* Variable will have the memory allocated on the heap as 
* go compiler will perform escape analysis to escape the varaible from the local scope.
**/
func newError(t *txn) *Error {
	return &Error {
		Err 	: t.resp.ErrCode,
		Detail	: t.resp.GetErrDetail(),
	}
}

/**
* -- Named Return Values --
**/
func (e Error) Error() (s string) {
	s = e.Err.Error()
	if e.Detail != "" {
		s += ": " + e.Detail
	}

	return
}