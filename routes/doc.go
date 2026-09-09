// Package routes provides HTTP handlers over
// [github.com/aosanya/mwanachama-backend-agency.AgencyManager], following
// mwanachama-backend-actor's routes/ package shape exactly.
//
// Every handler here is decode -> one AgencyManager call -> encode. None
// carry session/capability/audit policy, request logging, or a mounting
// path prefix — those are the mounting process's job (e.g. the gateway,
// AG10), the same division of responsibility actor's own routes/ package
// documents. Path parameters use Go 1.22+ net/http.ServeMux wildcard
// syntax ({goalID}, etc.), read via r.PathValue.
package routes
