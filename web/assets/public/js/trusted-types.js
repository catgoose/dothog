if (window.trustedTypes) {
	trustedTypes.createPolicy('default', {
		createHTML: function(value) { return value; }
	});
}
