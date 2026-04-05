/**
 * Attach the CSRF token to every outgoing HTMX request.
 * Reads the token from the <meta name="csrf-token"> tag injected
 * by the server and sets the X-CSRF-Token header.
 * @listens htmx:configRequest
 */
document.addEventListener("htmx:configRequest", function(evt) {
	/** @type {HTMLMetaElement|null} */
	var t = document.querySelector("meta[name=\"csrf-token\"]");
	if (t) evt.detail.headers["X-CSRF-Token"] = t.getAttribute("content");
});
