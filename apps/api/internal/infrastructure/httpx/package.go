// Package httpx holds the HTTP primitives shared by the composition layer and
// every module's HTTP handler: request correlation, success/error envelopes,
// error classification and path/query parsing.
//
// It must stay dependency-free of any business module and of the platform
// composition package so that platform can assemble handlers and handlers can
// use these primitives without an import cycle.
package httpx
