package rd

// This file has been split into multiple focused files:
// - activity_types.go: Activity type detection and emoji mapping
// - activity_parser.go: HTML parsing logic and ActivityInfo struct
// - activity_iterator.go: ActivityIterator implementation
//
// This approach follows Unix principles of doing one thing well
// and makes the code more maintainable and testable.

// Example usage:
// iter := NewActivityIterator(client, time.Now())
// for activity, ok := iter.Next(); ok; activity, ok = iter.Next() {
//     fmt.Println(activity)
// }
