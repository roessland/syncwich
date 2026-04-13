// Package errs provides tiny helpers for error paths that cannot happen under
// our own controlled inputs. Reach for them only when a non-nil error would
// indicate a programmer bug, not a runtime condition worth handling.
package errs

// Check panics when err is non-nil.
func Check(err error) {
	if err != nil {
		panic(err)
	}
}

// Check2 panics when err is non-nil, otherwise returns v.
func Check2[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
