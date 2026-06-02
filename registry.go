package agenthooks

// adapterFor turns a unified handler of type F into a (route, RouteFunc) pair.
// Each platform package provides one per event category it supports; the
// translation between that platform's wire types and the unified vocabulary
// lives in the platform package, so this file stays a thin registry.
type adapterFor[F any] func(F) (string, func())

// registerAdapters wires every supplied platform adapter for one unified handler.
func registerAdapters[F any](fn F, adapters ...adapterFor[F]) {
	for _, mk := range adapters {
		name, run := mk(fn)
		AddRoute(name, run)
	}
}
