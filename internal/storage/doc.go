// Package storage handles file-level persistence for all bujotui data:
// daily entries (monthly markdown files), collections, habits, and the
// future log. All writes are atomic via temp-file-then-rename.
package storage
