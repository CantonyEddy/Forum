//go:build sqlite_see
// +build sqlite_see

package sqlite3

/*
#cgo CFLAGS: -DSQLITE_HAS_CODEC
#cgo LDFLAGS: -lcrypto
#cgo LDFLAGS: -lsqlcipher
*/
import "C"
