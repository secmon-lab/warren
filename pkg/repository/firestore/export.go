package firestore

import "cloud.google.com/go/firestore"

// GetClient returns the Firestore client for testing purposes
func (f *Firestore) GetClient() *firestore.Client {
	return f.db
}
