package servicehttp

import "net/http"

// SecurityHeaders sends the two headers Spring Security's HeaderWriterFilter
// adds to every response the JVM serves, so a path group moved off the JVM
// does not silently lose them.
//
// The headers are set before next runs rather than when it writes, so an
// error response written by any inner layer carries them too. It belongs
// outermost for the same reason: the JVM writes them ahead of its CORS filter,
// so even a refused origin gets them.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
