package extract_test

//go:generate impgen Enricher --dependency
//go:generate impgen Classifier --dependency
//go:generate impgen Reconciler --dependency
//go:generate impgen SessionOverlaps --dependency
//go:generate impgen Run --target
