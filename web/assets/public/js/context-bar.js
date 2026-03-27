// setup:feature:demo
/**
 * Alpine.js component that populates the context bar from <link rel="related"> tags.
 * Reads link relations from the document head and renders them as navigation links.
 * @returns {AlpineComponent}
 */
function contextBar() {
  return {
    links: [],
    init() {
      this.links = Array.from(document.querySelectorAll('head link[rel="related"]'))
        .map(function(el) { return { href: el.getAttribute('href'), title: el.getAttribute('title') }; })
        .filter(function(l) { return l.href && l.title && l.href !== window.location.pathname; });
    }
  };
}
