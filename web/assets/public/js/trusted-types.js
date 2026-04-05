/**
 * @fileoverview Trusted Types pass-through policy for CSP compliance.
 *
 * This is a hypermedia application — the server is the trusted source of all
 * HTML. HTMX and Alpine.morph both inject server-rendered markup into the DOM,
 * which triggers Trusted Types checks in browsers that enforce the
 * require-trusted-types-for CSP directive.
 *
 * The policy is intentionally a pass-through: it does not sanitize. Client-side
 * sanitization would duplicate (and conflict with) the server's template
 * escaping. The policy exists to satisfy the browser's Trusted Types gate, not
 * to add a second layer of filtering.
 */
if (window.trustedTypes) {
	trustedTypes.createPolicy('default', {
		createHTML: function(value) { return value; }
	});
}
